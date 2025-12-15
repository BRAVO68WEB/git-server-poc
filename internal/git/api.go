package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"githut/internal/config"
	"githut/internal/database"
)

func RegisterAPI(mux *http.ServeMux, cfg config.Config) {
	mux.HandleFunc("/api/repos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Exact match for /api/repos
		if r.URL.Path != "/api/repos" && r.URL.Path != "/api/repos/" {
			// This might be handled by the other handler if registered,
			// but if not, we should probably fall through or return 404?
			// With ServeMux, "/api/repos/" handler covers everything under it.
			// So this handler is likely only for "/api/repos" if we don't put trailing slash.
			// But wait, if I register "/api/repos/" it catches "/api/repos" too?
			// Standard ServeMux: "/api/repos/" matches "/api/repos/..."
			// "/api/repos" matches "/api/repos".
			// So I should keep them separate or use one.
			// I'll keep this one for list repos.
		}

		ctx := r.Context()
		db, err := database.Connect(ctx, cfg.PostgresDSN)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close(ctx)

		repos, err := db.ListAllRepos(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var out []Repo
		for _, r := range repos {
			out = append(out, Repo{
				Owner:       r.OwnerName,
				Name:        r.Name,
				Description: r.Description,
				Visibility:  r.Visibility,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/api/repos/", func(w http.ResponseWriter, r *http.Request) {
		// Path: /api/repos/{owner}/{repo}/{action}/{path...}
		path := strings.TrimPrefix(r.URL.Path, "/api/repos/")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		owner, repo := parts[0], parts[1]
		repoPath := ensureLocalRepo(owner, repo)

		if len(parts) == 2 {
			// Get Repo Info
			// TODO: Fetch from DB
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Repo{Owner: owner, Name: repo})
			return
		}

		action := parts[2]
		rest := ""
		if len(parts) > 3 {
			rest = strings.Join(parts[3:], "/")
		}

		switch action {
		case "tree":
			ref, dir, err := resolveRefPath(repoPath, rest)
			if err != nil {
				http.Error(w, "invalid ref", http.StatusBadRequest)
				return
			}
			if dir == "" {
				dir = "."
			}
			entries, err := listTree(repoPath, ref, dir)
			if err != nil {
				code := statusFromGitError(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ref": ref,
				"path": func() string {
					if dir == "." {
						return ""
					}
					return dir
				}(),
				"entries": entries,
			})

		case "blob":
			ref, filePath, err := resolveRefPath(repoPath, rest)
			if err != nil {
				http.Error(w, "invalid ref", http.StatusBadRequest)
				return
			}
			if filePath == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			content, err := getBlob(repoPath, ref, filePath)
			if err != nil {
				code := statusFromGitError(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ref":     ref,
				"path":    filePath,
				"content": string(content),
			})

		case "commits":
			ref, filePath, err := resolveRefPath(repoPath, rest)
			if err != nil {
				http.Error(w, "invalid ref", http.StatusBadRequest)
				return
			}
			commits, err := getCommits(repoPath, ref, filePath)
			if err != nil {
				code := statusFromGitError(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ref":     ref,
				"path":    filePath,
				"commits": commits,
			})

		case "branches":
			branches, err := getBranches(repoPath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(branches)

		case "diff":
			// diff/{hash}
			// rest is hash
			hash := rest
			if hash == "" {
				http.Error(w, "commit hash required", http.StatusBadRequest)
				return
			}
			diff, err := getDiff(repoPath, hash)
			if err != nil {
				code := statusFromGitError(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Diff{Content: string(diff)})

		case "blame":
			ref, filePath, err := resolveRefPath(repoPath, rest)
			if err != nil {
				http.Error(w, "invalid ref", http.StatusBadRequest)
				return
			}
			if filePath == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			blame, err := getBlame(repoPath, ref, filePath)
			if err != nil {
				code := statusFromGitError(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ref":   ref,
				"path":  filePath,
				"blame": blame,
			})

		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	})
}

func statusFromGitError(msg string) int {
	m := strings.ToLower(msg)
	if strings.Contains(m, "does not exist") ||
		strings.Contains(m, "unknown revision") ||
		strings.Contains(m, "unknown path") ||
		strings.Contains(m, "bad revision") ||
		strings.Contains(m, "not a valid object") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func resolveRefPath(repoPath, urlPath string) (string, string, error) {
	if urlPath == "" {
		return "HEAD", "", nil
	}
	parts := strings.Split(urlPath, "/")
	for i := 1; i <= len(parts); i++ {
		candidate := strings.Join(parts[:i], "/")
		if isValidRef(repoPath, candidate) {
			path := ""
			if i < len(parts) {
				path = strings.Join(parts[i:], "/")
			}
			return candidate, path, nil
		}
	}
	return "", "", fmt.Errorf("no valid ref found in path: %s", urlPath)
}

func isValidRef(repoPath, ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = repoPath
	return cmd.Run() == nil
}

func listTree(repoPath, ref, dir string) ([]FileEntry, error) {
	var cmd *exec.Cmd
	if dir == "." || dir == "" {
		cmd = exec.Command("git", "ls-tree", ref)
	} else {
		d := dir
		if !strings.HasSuffix(d, "/") {
			d = d + "/"
		}
		cmd = exec.Command("git", "ls-tree", ref, "--", d)
	}
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(string(out))
	}
	var entries []FileEntry
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// 100644 blob <hash>\t<name>
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		mode := parts[0]
		type_ := parts[1]
		hash := parts[2]
		// Use cut for tab.
		_, afterTab, _ := strings.Cut(line, "\t")
		entries = append(entries, FileEntry{
			Mode: mode,
			Type: type_,
			Hash: hash,
			Name: afterTab,
		})
	}
	return entries, nil
}

func getBlob(repoPath, ref, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, path))
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(string(out))
	}
	return out, nil
}

func getCommits(repoPath, ref, path string) ([]Commit, error) {
	// git log --pretty=format:"%H|%an|%aI|%s" -n 20 <ref> -- <path>
	args := []string{"log", "--pretty=format:%H|%an|%aI|%s", "-n", "20", ref}
	if path != "" {
		args = append(args, "--", path)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(string(out))
	}
	var commits []Commit
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    t,
			Message: parts[3],
		})
	}
	return commits, nil
}

func getBranches(repoPath string) ([]Branch, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)|%(HEAD)")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var branches []Branch
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		branches = append(branches, Branch{
			Name:   parts[0],
			IsHead: parts[1] == "*",
		})
	}
	return branches, nil
}

func getDiff(repoPath, hash string) ([]byte, error) {
	cmd := exec.Command("git", "show", hash)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(string(out))
	}
	return out, nil
}

func getBlame(repoPath, ref, path string) ([]BlameLine, error) {
	// git blame --line-porcelain <ref> -- <path>
	cmd := exec.Command("git", "blame", "--line-porcelain", ref, "--", path)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(string(out))
	}

	output := string(out)
	fileLines := strings.Split(output, "\n")

	var lines []BlameLine
	var current BlameLine

	for i := 0; i < len(fileLines); i++ {
		line := fileLines[i]
		if line == "" {
			continue
		}
		// First line of a block is SHA + line numbers
		// 46b9... 1 1 1
		parts := strings.Fields(line)
		if len(parts) >= 3 && len(parts[0]) == 40 { // Looks like a SHA line
			current = BlameLine{Commit: parts[0]}
			// Parse following headers until TAB
			for {
				i++
				if i >= len(fileLines) {
					break
				}
				header := fileLines[i]
				if strings.HasPrefix(header, "\t") {
					// Content line
					current.Content = strings.TrimPrefix(header, "\t")
					lines = append(lines, current)
					break
				}
				if strings.HasPrefix(header, "author ") {
					current.Author = strings.TrimPrefix(header, "author ")
				}
				if strings.HasPrefix(header, "author-time ") {
					var sec int64
					fmt.Sscanf(strings.TrimPrefix(header, "author-time "), "%d", &sec)
					current.Date = time.Unix(sec, 0)
				}
			}
		}
	}

	// Add line numbers
	for i := range lines {
		lines[i].LineNo = i + 1
	}

	return lines, nil
}
