package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"githut/internal/config"
	"githut/internal/database"
	"githut/internal/observability"
)

func RegisterHTTP(mux *http.ServeMux, cfg config.Config) {
	mux.HandleFunc("/git/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/git/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		owner, repo, action := parts[0], parts[1], parts[2]
		repoPath := ensureLocalRepo(owner, repo)
		switch action {
		case "info":
			if len(parts) < 4 || parts[3] != "refs" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			service := r.URL.Query().Get("service")
			switch service {
			case "git-upload-pack":
				if cfg.PostgresDSN != "" && !allowUploadPack(r.Context(), cfg.PostgresDSN, owner, repo, r) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
				w.Header().Set("Cache-Control", "no-cache")
				_ = runGitAdvertise(w, r, "git-upload-pack", repoPath)
				return
			case "git-receive-pack":
				var actor *database.UserRow
				if cfg.PostgresDSN != "" {
					if u := validateBearerDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u != nil && !u.Disabled {
						actor = u
					} else if u2 := validateBasicDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u2 != nil && !u2.Disabled {
						actor = u2
					}
				}
				if actor == nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				ok, err := dbConnectNoErr(r.Context(), cfg.PostgresDSN).HasPushAccess(r.Context(), actor.Username, owner, repo)
				if err != nil || !ok {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				w.Header().Set("Content-Type", "application/x-git-receive-pack-advertisement")
				w.Header().Set("Cache-Control", "no-cache")
				_ = runGitAdvertise(w, r, "git-receive-pack", repoPath)
				return
			default:
				http.Error(w, "bad service", http.StatusBadRequest)
				return
			}
		case "git-upload-pack":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if cfg.PostgresDSN != "" && !allowUploadPack(r.Context(), cfg.PostgresDSN, owner, repo, r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			start := time.Now()
			err := runGitService(w, r, "git-upload-pack", repoPath)
			observability.RecordUploadPack(time.Since(start), err != nil)
			var actorID *int64
			if cfg.PostgresDSN != "" {
				if u := validateBearerDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u != nil {
					actorID = &u.ID
				} else if u2 := validateBasicDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u2 != nil {
					actorID = &u2.ID
				}
			}
			logAudit(r.Context(), cfg.PostgresDSN, owner, repo, actorID, "http_upload_pack", r)
			return
		case "git-receive-pack":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var actor *database.UserRow
			if cfg.PostgresDSN != "" {
				if u := validateBearerDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u != nil && !u.Disabled {
					actor = u
				} else if u2 := validateBasicDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u2 != nil && !u2.Disabled {
					actor = u2
				}
			}
			if actor == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ok, err := dbConnectNoErr(r.Context(), cfg.PostgresDSN).HasPushAccess(r.Context(), actor.Username, owner, repo)
			if err != nil || !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
			start := time.Now()
			err = runGitService(w, r, "git-receive-pack", repoPath)
			observability.RecordReceivePack(time.Since(start), err != nil)
			var actorID *int64
			if cfg.PostgresDSN != "" {
				if u := validateBearerDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u != nil {
					actorID = &u.ID
				} else if u2 := validateBasicDB(r.Context(), dbConnectNoErr(r.Context(), cfg.PostgresDSN), r); u2 != nil {
					actorID = &u2.ID
				}
			}
			logAudit(r.Context(), cfg.PostgresDSN, owner, repo, actorID, "http_receive_pack", r)
			return
		}
	})
}

func allowUploadPack(ctx context.Context, dsn, owner, repo string, r *http.Request) bool {
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		return false
	}
	defer db.Close(ctx)
	rr, err := db.GetRepoByOwnerAndName(ctx, owner, repo)
	if err != nil {
		return false
	}
	if rr.Visibility == "public" || rr.Visibility == "internal" {
		// Try to authenticate for audit purposes; still allow if not provided
		if u := validateBearerDB(ctx, db, r); u != nil && !u.Disabled {
			return true
		}
		if u2 := validateBasicDB(ctx, db, r); u2 != nil && !u2.Disabled {
			return true
		}
		return true
	}
	if u := validateBearerDB(ctx, db, r); u != nil && !u.Disabled {
		ok, err := db.HasPullAccess(ctx, u.Username, owner, repo)
		return err == nil && ok
	}
	if u2 := validateBasicDB(ctx, db, r); u2 != nil && !u2.Disabled {
		ok, err := db.HasPullAccess(ctx, u2.Username, owner, repo)
		return err == nil && ok
	}
	return false
}

func validateBearerDB(ctx context.Context, db *database.DB, r *http.Request) *database.UserRow {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return nil
	}
	raw := strings.TrimPrefix(h, p)
	u, err := db.ValidateToken(ctx, raw)
	if err != nil {
		return nil
	}
	return u
}

func validateBasicDB(ctx context.Context, db *database.DB, r *http.Request) *database.UserRow {
	h := r.Header.Get("Authorization")
	const p = "Basic "
	if !strings.HasPrefix(h, p) {
		return nil
	}
	raw := strings.TrimPrefix(h, p)
	dec, err := decodeBasic(raw)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(dec, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	u, err := db.ValidateUserPassword(ctx, parts[0], parts[1])
	if err != nil {
		return nil
	}
	return u
}

func decodeBasic(b64 string) (string, error) {
	data, err := base64StdDecode(b64)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func base64StdDecode(s string) ([]byte, error) {
	// avoid adding new imports; use existing strings and encoding in stdlib via exec? Not possible.
	// We will import encoding/base64 at top.
	return base64.StdEncoding.DecodeString(s)
}

func dbConnectNoErr(ctx context.Context, dsn string) *database.DB {
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		return nil
	}
	return db
}

func ensureLocalRepo(owner, repo string) string {
	base := filepath.Join("data", "repos", owner)
	_ = os.MkdirAll(base, 0o755)
	path := filepath.Join(base, repo+".git")
	_ = os.MkdirAll(path, 0o755)
	objects := filepath.Join(path, "objects")
	_ = os.MkdirAll(objects, 0o755)
	head := filepath.Join(path, "HEAD")
	if _, err := os.Stat(head); os.IsNotExist(err) {
		_ = initBareRepo(path)
	}
	return path
}

func initBareRepo(path string) error {
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = path
	return cmd.Run()
}

func runGitService(w http.ResponseWriter, r *http.Request, svc string, repoPath string) error {
	cmd := exec.CommandContext(r.Context(), svc, "--stateless-rpc", repoPath)
	env := os.Environ()
	env = append(env, "GIT_HTTP_EXPORT_ALL=1")
	if p := r.Header.Get("Git-Protocol"); p != "" {
		env = append(env, "GIT_PROTOCOL="+p)
	}
	cmd.Env = env
	cmd.Stdin = r.Body
	cmd.Stdout = w
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func runGitAdvertise(w http.ResponseWriter, r *http.Request, svc string, repoPath string) error {
	prelude := "# service=" + svc + "\n"
	if err := writePktLine(w, prelude); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	if _, err := w.Write([]byte("0000")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	cmd := exec.CommandContext(r.Context(), svc, "--stateless-rpc", "--advertise-refs", repoPath)
	env := os.Environ()
	env = append(env, "GIT_HTTP_EXPORT_ALL=1")
	if p := r.Header.Get("Git-Protocol"); p != "" {
		env = append(env, "GIT_PROTOCOL="+p)
	}
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	if _, err := io.Copy(w, stdout); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	if err := cmd.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func writePktLine(w http.ResponseWriter, s string) error {
	n := len(s) + 4
	h := fmt.Sprintf("%04x", n)
	if _, err := w.Write([]byte(h)); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func logAudit(ctx context.Context, dsn, owner, repo string, actorID *int64, action string, r *http.Request) {
	if dsn == "" {
		return
	}
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	ip := clientIP(r)
	meta := map[string]any{
		"path":   r.URL.Path,
		"length": r.ContentLength,
	}
	b, _ := json.Marshal(meta)
	_ = db.InsertAuditByOwnerRepo(ctx, owner, repo, actorID, action, ip, string(b))
}

func clientIP(r *http.Request) net.IP {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		parts := strings.Split(x, ",")
		if len(parts) > 0 {
			return net.ParseIP(strings.TrimSpace(parts[0]))
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return net.ParseIP(host)
}
