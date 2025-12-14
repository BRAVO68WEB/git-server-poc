package lfs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"githut/internal/config"
	"githut/internal/database"
	"githut/internal/storage"
	"github.com/gin-gonic/gin"
)

type batchReq struct {
	Operation string `json:"operation"`
	Objects   []struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"objects"`
}

type batchResp struct {
	Objects []struct {
		OID    string                 `json:"oid"`
		Size   int64                  `json:"size"`
		Actions map[string]map[string]any `json:"actions"`
	} `json:"objects"`
}

func RegisterHTTP(mux *http.ServeMux, cfg config.Config) {
	store := storage.FromConfig(cfg)
	mux.HandleFunc("/lfs/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/lfs/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		owner, repo := parts[0], parts[1]
		action := parts[2]
		switch action {
		case "objects":
			// objects/{oid} GET/PUT or /objects/batch
			if len(parts) >= 4 && parts[3] == "batch" && r.Method == http.MethodPost {
				handleBatch(w, r, cfg, store, owner, repo)
				return
			}
			if len(parts) >= 4 {
				oid := parts[3]
				switch r.Method {
				case http.MethodGet:
					handleGetObject(w, r, cfg, store, owner, repo, oid)
					return
				case http.MethodPut:
					handlePutObject(w, r, cfg, store, owner, repo, oid)
					return
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
			}
			http.Error(w, "not found", http.StatusNotFound)
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	})
}

func RegisterHTTPGin(r *gin.Engine, cfg config.Config) {
	store := storage.FromConfig(cfg)
	r.POST("/lfs/:owner/:repo/objects/batch", func(c *gin.Context) {
		handleBatch(c.Writer, c.Request, cfg, store, c.Param("owner"), c.Param("repo"))
	})
	r.GET("/lfs/:owner/:repo/objects/:oid", func(c *gin.Context) {
		handleGetObject(c.Writer, c.Request, cfg, store, c.Param("owner"), c.Param("repo"), c.Param("oid"))
	})
	r.PUT("/lfs/:owner/:repo/objects/:oid", func(c *gin.Context) {
		handlePutObject(c.Writer, c.Request, cfg, store, c.Param("owner"), c.Param("repo"), c.Param("oid"))
	})
}

func handleBatch(w http.ResponseWriter, r *http.Request, cfg config.Config, store storage.Storage, owner, repo string) {
	var req batchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
	var resp batchResp
	for _, o := range req.Objects {
		actions := map[string]map[string]any{}
		switch req.Operation {
		case "download":
			actions["download"] = map[string]any{
				"href": "/lfs/" + owner + "/" + repo + "/objects/" + o.OID,
			}
		case "upload":
			// require push permissions
			if cfg.PostgresDSN != "" && !hasPush(r, cfg, owner, repo) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			actions["upload"] = map[string]any{
				"href": "/lfs/" + owner + "/" + repo + "/objects/" + o.OID,
			}
		default:
			http.Error(w, "bad operation", http.StatusBadRequest)
			return
		}
		resp.Objects = append(resp.Objects, struct {
			OID    string                 `json:"oid"`
			Size   int64                  `json:"size"`
			Actions map[string]map[string]any `json:"actions"`
		}{
			OID:    o.OID,
			Size:   o.Size,
			Actions: actions,
		})
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGetObject(w http.ResponseWriter, r *http.Request, cfg config.Config, store storage.Storage, owner, repo, oid string) {
	// authorize: public/internal OK, private requires pull
	if cfg.PostgresDSN != "" && !hasPull(r, cfg, owner, repo) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Git\"")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rc, err := store.Get(r.Context(), key(owner, repo, oid))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, rc)
}

func handlePutObject(w http.ResponseWriter, r *http.Request, cfg config.Config, store storage.Storage, owner, repo, oid string) {
	// require push
	if cfg.PostgresDSN != "" && !hasPush(r, cfg, owner, repo) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Git\"")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := store.Put(r.Context(), key(owner, repo, oid), r.Body); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func key(owner, repo, oid string) string {
	return owner + "/" + repo + "/" + oid
}

func hasPull(r *http.Request, cfg config.Config, owner, repo string) bool {
	ctx := r.Context()
	db, err := database.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		return false
	}
	defer db.Close(ctx)
	rr, err := db.GetRepoByOwnerAndName(ctx, owner, repo)
	if err != nil {
		return false
	}
	if rr.Visibility == "public" || rr.Visibility == "internal" {
		return true
	}
	u := validateBearerDB(ctx, db, r)
	if u == nil || u.Disabled {
		return false
	}
	ok, err := db.HasPullAccess(ctx, u.Username, owner, repo)
	return err == nil && ok
}

func hasPush(r *http.Request, cfg config.Config, owner, repo string) bool {
	ctx := r.Context()
	db, err := database.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		return false
	}
	defer db.Close(ctx)
	u := validateBearerDB(ctx, db, r)
	if u == nil || u.Disabled {
		return false
	}
	ok, err := db.HasPushAccess(ctx, u.Username, owner, repo)
	return err == nil && ok
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
