package main

// import (
// 	"bufio"
// 	"context"
// 	"crypto/sha256"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"githut/internal/config"
// 	"githut/internal/database"
// 	"githut/internal/git"
// 	"githut/internal/lfs"
// 	"githut/internal/observability"

// 	"github.com/spf13/cobra"
// )

// func main() {
// 	var cfgPath string
// 	root := &cobra.Command{
// 		Use:   "githut",
// 		Short: "Githut CLI",
// 		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
// 			if cfgPath != "" {
// 				config.SetConfigPath(cfgPath)
// 			}
// 			return nil
// 		},
// 	}
// 	root.PersistentFlags().StringVar(&cfgPath, "config", "", "path to githut.yaml")
// 	root.AddCommand(serveCmd())
// 	root.AddCommand(usersCmd())
// 	root.AddCommand(reposCmd())
// 	root.AddCommand(adminCmd())
// 	root.AddCommand(statsCmd())
// 	root.AddCommand(dbCmd())
// 	if err := root.Execute(); err != nil {
// 		fmt.Fprintln(os.Stderr, err.Error())
// 		os.Exit(1)
// 	}
// }

// func serveCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "serve",
// 		Short: "Run HTTP and optional SSH servers",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			mux := http.NewServeMux()
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" || cfg.HTTPAddr == "" {
// 				return fmt.Errorf("missing required configuration: postgres_dsn or http_addr")
// 			}
// 			addr := cfg.HTTPAddr
// 			git.RegisterHTTP(mux, cfg)
// 			git.RegisterAPI(mux, cfg)
// 			observability.RegisterMetrics(mux)
// 			lfs.RegisterHTTP(mux, cfg)
// 			mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
// 				w.WriteHeader(http.StatusOK)
// 				_, _ = w.Write([]byte("ok"))
// 			})
// 			mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
// 				w.WriteHeader(http.StatusOK)
// 				_, _ = w.Write([]byte("ready"))
// 			})
// 			log.Printf("serve listening on %s", addr)
// 			if cfg.SSHAddr != "" && cfg.PostgresDSN != "" {
// 				go func() {
// 					db, err := database.Connect(context.Background(), cfg.PostgresDSN)
// 					if err != nil {
// 						log.Printf("ssh start error: %v", err)
// 						return
// 					}
// 					defer db.Close(context.Background())
// 					if err := git.StartSSH(context.Background(), cfg.SSHAddr, db); err != nil {
// 						log.Printf("ssh start error: %v", err)
// 					}
// 				}()
// 			}
// 			err := http.ListenAndServe(addr, logMiddleware(mux))
// 			if err != nil {
// 				log.Fatalf("server error: %v", err)
// 			}
// 			return nil
// 		},
// 	}
// 	return cmd
// }

// func usersCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "users",
// 		Short: "Manage users",
// 	}
// 	create := &cobra.Command{
// 		Use:   "create",
// 		Short: "Create user",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			email, _ := cmd.Flags().GetString("email")
// 			role, _ := cmd.Flags().GetString("role")
// 			password, _ := cmd.Flags().GetString("password")
// 			if username == "" || email == "" || role == "" || password == "" {
// 				return fmt.Errorf("username, email, role, and password are required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.CreateUser(ctx, username, email, role, password); err != nil {
// 				return err
// 			}
// 			fmt.Println("created")
// 			return nil
// 		},
// 	}
// 	create.Flags().String("username", "", "username")
// 	create.Flags().String("email", "", "email")
// 	create.Flags().String("role", "developer", "role")
// 	create.Flags().String("password", "", "password")
// 	cmd.AddCommand(create)
// 	cmd.AddCommand(sshKeyCmd())
// 	token := &cobra.Command{
// 		Use:   "token",
// 		Short: "Manage user tokens",
// 	}
// 	tc := &cobra.Command{
// 		Use:   "create",
// 		Short: "Create token",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			name, _ := cmd.Flags().GetString("name")
// 			if username == "" || name == "" {
// 				return fmt.Errorf("username and name are required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			raw, err := db.CreateToken(ctx, username, name)
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Println(raw)
// 			return nil
// 		},
// 	}
// 	tc.Flags().String("username", "", "username")
// 	tc.Flags().String("name", "", "token name")
// 	token.AddCommand(tc)
// 	tl := &cobra.Command{
// 		Use:   "list",
// 		Short: "List tokens",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			if username == "" {
// 				return fmt.Errorf("username is required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			toks, err := db.ListTokensByUser(ctx, username)
// 			if err != nil {
// 				return err
// 			}
// 			for _, t := range toks {
// 				state := "active"
// 				if t.Revoked {
// 					state = "revoked"
// 				}
// 				fmt.Println(t.Name, state)
// 			}
// 			return nil
// 		},
// 	}
// 	tl.Flags().String("username", "", "username")
// 	token.AddCommand(tl)
// 	tr := &cobra.Command{
// 		Use:   "revoke",
// 		Short: "Revoke token",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			raw, _ := cmd.Flags().GetString("raw")
// 			if raw == "" {
// 				return fmt.Errorf("raw token required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.RevokeToken(ctx, raw); err != nil {
// 				return err
// 			}
// 			fmt.Println("revoked")
// 			return nil
// 		},
// 	}
// 	tr.Flags().String("raw", "", "raw token")
// 	token.AddCommand(tr)
// 	cmd.AddCommand(token)
// 	return cmd
// }

// func sshKeyCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "ssh-key",
// 		Short: "Manage SSH keys",
// 	}
// 	add := &cobra.Command{
// 		Use:   "add",
// 		Short: "Add SSH key",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			name, _ := cmd.Flags().GetString("name")
// 			path, _ := cmd.Flags().GetString("path")
// 			if username == "" || name == "" || path == "" {
// 				return fmt.Errorf("username, name, and path are required")
// 			}
// 			f, err := os.Open(path)
// 			if err != nil {
// 				return err
// 			}
// 			defer f.Close()
// 			s := bufio.NewScanner(f)
// 			if !s.Scan() {
// 				return fmt.Errorf("empty key file")
// 			}
// 			pub := s.Text()
// 			sum := sha256.Sum256([]byte(pub))
// 			fp := fmt.Sprintf("%x", sum[:])
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.AddSSHKey(ctx, username, name, pub, fp); err != nil {
// 				return err
// 			}
// 			fmt.Println("ok")
// 			return nil
// 		},
// 	}
// 	add.Flags().String("username", "", "username")
// 	add.Flags().String("name", "", "key name")
// 	add.Flags().String("path", "", "public key path")
// 	list := &cobra.Command{
// 		Use:   "list",
// 		Short: "List SSH keys",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			if username == "" {
// 				return fmt.Errorf("username required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			keys, err := db.ListSSHKeys(ctx, username)
// 			if err != nil {
// 				return err
// 			}
// 			for _, k := range keys {
// 				fmt.Println(k.Name, k.Fingerprint)
// 			}
// 			return nil
// 		},
// 	}
// 	list.Flags().String("username", "", "username")
// 	del := &cobra.Command{
// 		Use:   "delete",
// 		Short: "Delete SSH key",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			username, _ := cmd.Flags().GetString("username")
// 			name, _ := cmd.Flags().GetString("name")
// 			if username == "" || name == "" {
// 				return fmt.Errorf("username and name required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.DeleteSSHKeyByName(ctx, username, name); err != nil {
// 				return err
// 			}
// 			fmt.Println("ok")
// 			return nil
// 		},
// 	}
// 	del.Flags().String("username", "", "username")
// 	del.Flags().String("name", "", "key name")
// 	cmd.AddCommand(add, list, del)
// 	return cmd
// }

// func reposCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "repos",
// 		Short: "Manage repositories",
// 	}
// 	create := &cobra.Command{
// 		Use:   "create",
// 		Short: "Create repository",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			owner, _ := cmd.Flags().GetString("owner")
// 			name, _ := cmd.Flags().GetString("name")
// 			visibility, _ := cmd.Flags().GetString("visibility")
// 			if owner == "" || name == "" || visibility == "" {
// 				return fmt.Errorf("owner, name, visibility required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			cmd2 := exec.Command("git", "init", "--bare", filepath.Join("data", "repos", owner, name+".git"))
// 			if err := cmd2.Run(); err != nil {
// 				return err
// 			}
// 			_, err = db.Conn.Exec(ctx, `INSERT INTO repos (owner_id, name, visibility, default_branch) VALUES ((SELECT id FROM users WHERE username=$1), $2, $3, 'main') ON CONFLICT DO NOTHING`, owner, name, visibility)
// 			return err
// 		},
// 	}
// 	create.Flags().String("owner", "", "owner username")
// 	create.Flags().String("name", "", "repository name")
// 	create.Flags().String("visibility", "", "visibility")
// 	cmd.AddCommand(create)
// 	members := &cobra.Command{
// 		Use:   "members",
// 		Short: "Manage repository members",
// 	}
// 	add := &cobra.Command{
// 		Use:   "add",
// 		Short: "Add member",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			owner, _ := cmd.Flags().GetString("owner")
// 			name, _ := cmd.Flags().GetString("name")
// 			username, _ := cmd.Flags().GetString("username")
// 			role, _ := cmd.Flags().GetString("role")
// 			if owner == "" || name == "" || username == "" || role == "" {
// 				return fmt.Errorf("owner, name, username, and role are required")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.AddMember(ctx, owner, name, username, role); err != nil {
// 				return err
// 			}
// 			fmt.Println("added")
// 			return nil
// 		},
// 	}
// 	add.Flags().String("owner", "", "owner username")
// 	add.Flags().String("name", "", "repository name")
// 	add.Flags().String("username", "", "member username")
// 	add.Flags().String("role", "", "role")
// 	members.AddCommand(add)
// 	cmd.AddCommand(members)
// 	return cmd
// }

// func adminCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "admin",
// 		Short: "Administrative operations",
// 	}
// 	settings := &cobra.Command{
// 		Use:   "settings",
// 		Short: "Show settings",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			fmt.Println("HTTP_ADDR:", cfg.HTTPAddr)
// 			fmt.Println("POSTGRES_DSN:", cfg.PostgresDSN)
// 			fmt.Println("S3_REGION:", cfg.S3Region)
// 			fmt.Println("S3_BUCKET:", cfg.S3Bucket)
// 			fmt.Println("S3_ENDPOINT:", cfg.S3Endpoint)
// 			return nil
// 		},
// 	}
// 	cmd.AddCommand(settings)
// 	cmd.PersistentFlags().String("owner", "", "owner username")
// 	cmd.PersistentFlags().String("name", "", "repository name")
// 	return cmd
// }

// func dbCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "db",
// 		Short: "Database operations",
// 	}
// 	migrate := &cobra.Command{
// 		Use:   "migrate",
// 		Short: "Database migrations",
// 	}
// 	status := &cobra.Command{
// 		Use:   "status",
// 		Short: "Show migration status",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			ms, err := db.MigrationsStatus(ctx)
// 			if err != nil {
// 				return err
// 			}
// 			for _, m := range ms {
// 				state := "pending"
// 				if m.Applied {
// 					state = "applied"
// 				}
// 				fmt.Println(m.Name, state)
// 			}
// 			return nil
// 		},
// 	}
// 	up := &cobra.Command{
// 		Use:   "up",
// 		Short: "Apply pending migrations",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.ApplyMigrations(ctx); err != nil {
// 				return err
// 			}
// 			fmt.Println("migrations applied")
// 			return nil
// 		},
// 	}
// 	stash := &cobra.Command{
// 		Use:   "stash",
// 		Short: "Delete all tables and data",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN == "" {
// 				return fmt.Errorf("database not configured")
// 			}
// 			fmt.Print("This will DROP all tables and data. Continue? [y/N]: ")
// 			var resp string
// 			_, _ = fmt.Scanln(&resp)
// 			if strings.ToLower(strings.TrimSpace(resp)) != "y" {
// 				fmt.Println("aborted")
// 				return nil
// 			}
// 			ctx := context.Background()
// 			db, err := database.Connect(ctx, cfg.PostgresDSN)
// 			if err != nil {
// 				return err
// 			}
// 			defer db.Close(ctx)
// 			if err := db.StashAll(ctx); err != nil {
// 				return err
// 			}
// 			fmt.Println("database stashed (all tables dropped)")
// 			return nil
// 		},
// 	}
// 	migrate.AddCommand(status)
// 	migrate.AddCommand(up)
// 	migrate.AddCommand(stash)
// 	cmd.AddCommand(migrate)
// 	return cmd
// }

// func statsCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "stats",
// 		Short: "Show statistics",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			cfg := config.Load()
// 			if cfg.PostgresDSN != "" {
// 				ctx := context.Background()
// 				db, err := database.Connect(ctx, cfg.PostgresDSN)
// 				if err != nil {
// 					return err
// 				}
// 				defer db.Close(ctx)
// 				var users, repos, keys, tokens, audits int
// 				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err == nil {
// 					fmt.Println("users", users)
// 				}
// 				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM repos`).Scan(&repos); err == nil {
// 					fmt.Println("repos", repos)
// 				}
// 				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM ssh_keys`).Scan(&keys); err == nil {
// 					fmt.Println("ssh_keys", keys)
// 				}
// 				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM tokens`).Scan(&tokens); err == nil {
// 					fmt.Println("tokens", tokens)
// 				}
// 				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&audits); err == nil {
// 					fmt.Println("audit_logs", audits)
// 				}
// 			}
// 			size := int64(0)
// 			count := 0
// 			root := filepath.Join("data", "repos")
// 			var walk func(string) error
// 			walk = func(p string) error {
// 				ents, err := os.ReadDir(p)
// 				if err != nil {
// 					if os.IsNotExist(err) {
// 						return nil
// 					}
// 					return err
// 				}
// 				for _, e := range ents {
// 					fp := filepath.Join(p, e.Name())
// 					if e.IsDir() {
// 						if strings.HasSuffix(e.Name(), ".git") {
// 							count++
// 						}
// 						if err := walk(fp); err != nil {
// 							return err
// 						}
// 					} else {
// 						fi, err := os.Stat(fp)
// 						if err == nil {
// 							size += fi.Size()
// 						}
// 					}
// 				}
// 				return nil
// 			}
// 			if err := walk(root); err != nil {
// 				return err
// 			}
// 			fmt.Println("fs_repos", count)
// 			fmt.Println("fs_bytes", size)
// 			return nil
// 		},
// 	}
// 	return cmd
// }

// func logMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		start := time.Now()
// 		lw := &loggingResponseWriter{ResponseWriter: w, status: 200}
// 		next.ServeHTTP(lw, r)
// 		d := time.Since(start)
// 		log.Printf(`{"ts":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d}`, time.Now().Format(time.RFC3339), r.Method, r.URL.Path, lw.status, d.Milliseconds())
// 	})
// }

// type loggingResponseWriter struct {
// 	http.ResponseWriter
// 	status int
// }

// func (lw *loggingResponseWriter) WriteHeader(code int) {
// 	lw.status = code
// 	lw.ResponseWriter.WriteHeader(code)
// }
