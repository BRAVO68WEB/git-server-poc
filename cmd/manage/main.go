package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"githut/internal/config"
	"githut/internal/database"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "githut manage",
		Short: "Management CLI",
	}

	root.AddCommand(usersCmd())
	root.AddCommand(reposCmd())
	root.AddCommand(adminCmd())
	root.AddCommand(statsCmd())
	root.AddCommand(dbCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func usersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
	}
	cmd.AddCommand(sshKeyCmd())
	token := &cobra.Command{
		Use:   "token",
		Short: "Manage user tokens",
	}
	tc := &cobra.Command{
		Use:   "create",
		Short: "Create token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			name, _ := cmd.Flags().GetString("name")
			if username == "" || name == "" {
				return fmt.Errorf("username and name are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			raw, err := db.CreateToken(ctx, username, name)
			if err != nil {
				return err
			}
			fmt.Println(raw)
			return nil
		},
	}
	tc.Flags().String("username", "", "username")
	tc.Flags().String("name", "", "token name")
	token.AddCommand(tc)

	tl := &cobra.Command{
		Use:   "list",
		Short: "List tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			if username == "" {
				return fmt.Errorf("username is required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			toks, err := db.ListTokensByUser(ctx, username)
			if err != nil {
				return err
			}
			for _, t := range toks {
				state := "active"
				if t.Revoked {
					state = "revoked"
				}
				fmt.Println(t.Name, state)
			}
			return nil
		},
	}
	tl.Flags().String("username", "", "username")
	token.AddCommand(tl)

	tr := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			raw, _ := cmd.Flags().GetString("token")
			if raw == "" {
				return fmt.Errorf("token is required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.RevokeToken(ctx, raw); err != nil {
				return err
			}
			fmt.Println("token revoked")
			return nil
		},
	}
	tr.Flags().String("token", "", "token raw value")
	token.AddCommand(tr)
	cmd.AddCommand(token)
	create := &cobra.Command{
		Use:   "create",
		Short: "Create user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			role, _ := cmd.Flags().GetString("role")
			password, _ := cmd.Flags().GetString("password")
			if username == "" || email == "" || role == "" {
				return fmt.Errorf("username, email, and role are required")
			}
			if password == "" {
				fmt.Print("Enter password: ")
				reader := bufio.NewReader(os.Stdin)
				pw, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				password = strings.TrimSpace(pw)
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.CreateUser(ctx, username, email, role, password); err != nil {
				return err
			}
			fmt.Println("user created:", username)
			return nil
		},
	}
	create.Flags().String("username", "", "username")
	create.Flags().String("email", "", "email")
	create.Flags().String("role", "", "role")
	create.Flags().String("password", "", "password (if empty, will prompt)")
	cmd.AddCommand(create)

	del := &cobra.Command{
		Use:   "delete",
		Short: "Delete user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			if username == "" {
				return fmt.Errorf("username is required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.DeleteUserByUsername(ctx, username); err != nil {
				return err
			}
			fmt.Println("user deleted:", username)
			return nil
		},
	}
	del.Flags().String("username", "", "username")
	cmd.AddCommand(del)

	list := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			us, err := db.ListUsers(ctx)
			if err != nil {
				return err
			}
			for _, u := range us {
				fmt.Println(u.Username)
			}
			return nil
		},
	}
	cmd.AddCommand(list)
	return cmd
}

func computeFingerprint(pubkey string) (string, error) {
	h := sha256.Sum256([]byte(strings.TrimSpace(pubkey)))
	return fmt.Sprintf("SHA256:%x", h[:]), nil
}

func sshKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sshkey",
		Short: "Manage SSH keys",
	}
	add := &cobra.Command{
		Use:   "add",
		Short: "Add SSH key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			name, _ := cmd.Flags().GetString("name")
			pubkey, _ := cmd.Flags().GetString("pubkey")
			if username == "" || name == "" || pubkey == "" {
				return fmt.Errorf("username, name, and pubkey are required")
			}
			fingerprint, err := computeFingerprint(pubkey)
			if err != nil {
				return err
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.AddSSHKey(ctx, username, name, pubkey, fingerprint); err != nil {
				return err
			}
			fmt.Println("ssh key added:", name, fingerprint)
			return nil
		},
	}
	add.Flags().String("username", "", "username")
	add.Flags().String("name", "", "key name")
	add.Flags().String("pubkey", "", "authorized_keys public key line")
	cmd.AddCommand(add)
	ls := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			if username == "" {
				return fmt.Errorf("username is required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			keys, err := db.ListSSHKeys(ctx, username)
			if err != nil {
				return err
			}
			for _, k := range keys {
				fmt.Println(k.Name, k.Fingerprint)
			}
			return nil
		},
	}
	ls.Flags().String("username", "", "username")
	cmd.AddCommand(ls)
	rm := &cobra.Command{
		Use:   "delete",
		Short: "Delete SSH key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			username, _ := cmd.Flags().GetString("username")
			name, _ := cmd.Flags().GetString("name")
			if username == "" || name == "" {
				return fmt.Errorf("username and name are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.DeleteSSHKeyByName(ctx, username, name); err != nil {
				return err
			}
			fmt.Println("ssh key deleted:", name)
			return nil
		},
	}
	rm.Flags().String("username", "", "username")
	rm.Flags().String("name", "", "key name")
	cmd.AddCommand(rm)
	return cmd
}

func reposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Manage repositories",
	}
	members := &cobra.Command{
		Use:   "members",
		Short: "Manage repository members",
	}
	add := &cobra.Command{
		Use:   "add",
		Short: "Add member",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			username, _ := cmd.Flags().GetString("username")
			role, _ := cmd.Flags().GetString("role")
			if owner == "" || name == "" || username == "" || role == "" {
				return fmt.Errorf("owner, name, username, and role are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.AddMember(ctx, owner, name, username, role); err != nil {
				return err
			}
			fmt.Println("member added:", username)
			return nil
		},
	}
	add.Flags().String("owner", "", "owner username")
	add.Flags().String("name", "", "repository name")
	add.Flags().String("username", "", "member username")
	add.Flags().String("role", "", "member role")
	members.AddCommand(add)
	rm := &cobra.Command{
		Use:   "remove",
		Short: "Remove member",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			username, _ := cmd.Flags().GetString("username")
			if owner == "" || name == "" || username == "" {
				return fmt.Errorf("owner, name, and username are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.RemoveMember(ctx, owner, name, username); err != nil {
				return err
			}
			fmt.Println("member removed:", username)
			return nil
		},
	}
	rm.Flags().String("owner", "", "owner username")
	rm.Flags().String("name", "", "repository name")
	rm.Flags().String("username", "", "member username")
	members.AddCommand(rm)
	ls := &cobra.Command{
		Use:   "list",
		Short: "List members",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			if owner == "" || name == "" {
				return fmt.Errorf("owner and name are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			ms, err := db.ListMembers(ctx, owner, name)
			if err != nil {
				return err
			}
			for _, m := range ms {
				fmt.Println(m.Username, m.Role)
			}
			return nil
		},
	}
	ls.Flags().String("owner", "", "owner username")
	ls.Flags().String("name", "", "repository name")
	members.AddCommand(ls)
	cmd.AddCommand(members)
	create := &cobra.Command{
		Use:   "create",
		Short: "Create repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			visibility, _ := cmd.Flags().GetString("visibility")
			if owner == "" || name == "" {
				return fmt.Errorf("owner and name are required")
			}
			cfg := config.Load()
			if cfg.PostgresDSN != "" {
				ctx := context.Background()
				db, err := database.Connect(ctx, cfg.PostgresDSN)
				if err != nil {
					return err
				}
				defer db.Close(ctx)
				if visibility == "" {
					visibility = "private"
				}
				if err := db.CreateRepo(ctx, owner, name, visibility); err != nil {
					return err
				}
			}
			base := filepath.Join("data", "repos", owner)
			_ = os.MkdirAll(base, 0o755)
			path := filepath.Join(base, name+".git")
			_ = os.MkdirAll(path, 0o755)
			cmdInit := exec.Command("git", "init", "--bare")
			cmdInit.Dir = path
			if err := cmdInit.Run(); err != nil {
				return err
			}
			fmt.Println("repo created:", owner+"/"+name)
			return nil
		},
	}
	create.Flags().String("owner", "", "owner username")
	create.Flags().String("name", "", "repository name")
	create.Flags().String("visibility", "", "repository visibility")
	cmd.AddCommand(create)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			if owner == "" || name == "" {
				return fmt.Errorf("owner and name are required")
			}
			cfg := config.Load()
			if cfg.PostgresDSN != "" {
				ctx := context.Background()
				db, err := database.Connect(ctx, cfg.PostgresDSN)
				if err != nil {
					return err
				}
				defer db.Close(ctx)
				if err := db.DeleteRepo(ctx, owner, name); err != nil {
					return err
				}
			}
			path := filepath.Join("data", "repos", owner, name+".git")
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			fmt.Println("repo deleted:", owner+"/"+name)
			return nil
		},
	}
	deleteCmd.Flags().String("owner", "", "owner username")
	deleteCmd.Flags().String("name", "", "repository name")
	cmd.AddCommand(deleteCmd)

	list := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			if owner == "" {
				return fmt.Errorf("owner is required")
			}
			cfg := config.Load()
			if cfg.PostgresDSN != "" {
				ctx := context.Background()
				db, err := database.Connect(ctx, cfg.PostgresDSN)
				if err != nil {
					return err
				}
				defer db.Close(ctx)
				rs, err := db.ListReposByOwner(ctx, owner)
				if err != nil {
					return err
				}
				if len(rs) == 0 {
					fmt.Println("no repos")
					return nil
				}
				for _, r := range rs {
					fmt.Println(r.Name)
				}
				return nil
			}
			dir := filepath.Join("data", "repos", owner)
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("no repos")
					return nil
				}
				return err
			}
			for _, e := range entries {
				if e.IsDir() && strings.HasSuffix(e.Name(), ".git") {
					info, _ := e.Info()
					if info != nil && info.Mode().IsDir() {
						fmt.Println(strings.TrimSuffix(e.Name(), ".git"))
					}
				}
			}
			return nil
		},
	}
	list.Flags().String("owner", "", "owner username")
	cmd.AddCommand(list)

	cmd.AddCommand(&cobra.Command{
		Use:   "fork",
		Short: "Fork repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("repo forked")
			return nil
		},
	})
	return cmd
}

func adminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "settings",
		Short: "Show system settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			fmt.Println("HTTP_ADDR:", cfg.HTTPAddr)
			fmt.Println("POSTGRES_DSN:", cfg.PostgresDSN)
			fmt.Println("S3_REGION:", cfg.S3Region)
			fmt.Println("S3_BUCKET:", cfg.S3Bucket)
			fmt.Println("S3_ENDPOINT:", cfg.S3Endpoint)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "backup",
		Short: "Run backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			if owner == "" || name == "" {
				return fmt.Errorf("owner and name are required")
			}
			_ = os.MkdirAll("backups", 0o755)
			out := filepath.Join("backups", fmt.Sprintf("%s-%s-%d.tar.gz", owner, name, time.Now().Unix()))
			base := filepath.Join("data", "repos", owner)
			cmdTar := exec.Command("tar", "-czf", out, "-C", base, name+".git")
			if err := cmdTar.Run(); err != nil {
				return err
			}
			fmt.Println("backup created:", out)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "audits",
		Short: "Show audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			owner, _ := cmd.Flags().GetString("owner")
			name, _ := cmd.Flags().GetString("name")
			if owner == "" || name == "" {
				return fmt.Errorf("owner and name are required")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			rows, err := db.ListAuditByOwnerRepo(ctx, owner, name, 20)
			if err != nil {
				return err
			}
			for _, r := range rows {
				actor := "unknown"
				if r.ActorID != nil {
					actor = fmt.Sprintf("%d", *r.ActorID)
				}
				fmt.Println(r.ID, actor, r.Action, r.IP)
			}
			return nil
		},
	})
	cmd.PersistentFlags().String("owner", "", "owner username")
	cmd.PersistentFlags().String("name", "", "repository name")
	return cmd
}

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations",
	}
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Database migrations",
	}
	status := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			ms, err := db.MigrationsStatus(ctx)
			if err != nil {
				return err
			}
			for _, m := range ms {
				state := "pending"
				if m.Applied {
					state = "applied"
				}
				fmt.Println(m.Name, state)
			}
			return nil
		},
	}
	up := &cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.ApplyMigrations(ctx); err != nil {
				return err
			}
			fmt.Println("migrations applied")
			return nil
		},
	}
	stash := &cobra.Command{
		Use:   "stash",
		Short: "Delete all tables and data",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN == "" {
				return fmt.Errorf("database not configured")
			}
			fmt.Print("This will DROP all tables and data. Continue? [y/N]: ")
			var resp string
			_, _ = fmt.Scanln(&resp)
			if strings.ToLower(strings.TrimSpace(resp)) != "y" {
				fmt.Println("aborted")
				return nil
			}
			ctx := context.Background()
			db, err := database.Connect(ctx, cfg.PostgresDSN)
			if err != nil {
				return err
			}
			defer db.Close(ctx)
			if err := db.StashAll(ctx); err != nil {
				return err
			}
			fmt.Println("database stashed (all tables dropped)")
			return nil
		},
	}
	migrate.AddCommand(status)
	migrate.AddCommand(up)
	migrate.AddCommand(stash)
	cmd.AddCommand(migrate)
	return cmd
}

func statsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.PostgresDSN != "" {
				ctx := context.Background()
				db, err := database.Connect(ctx, cfg.PostgresDSN)
				if err != nil {
					return err
				}
				defer db.Close(ctx)
				var users, repos, keys, tokens, audits int
				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err == nil {
					fmt.Println("users", users)
				}
				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM repos`).Scan(&repos); err == nil {
					fmt.Println("repos", repos)
				}
				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM ssh_keys`).Scan(&keys); err == nil {
					fmt.Println("ssh_keys", keys)
				}
				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM tokens`).Scan(&tokens); err == nil {
					fmt.Println("tokens", tokens)
				}
				if err := db.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&audits); err == nil {
					fmt.Println("audit_logs", audits)
				}
			}
			size := int64(0)
			count := 0
			root := filepath.Join("data", "repos")
			var walk func(string) error
			walk = func(p string) error {
				ents, err := os.ReadDir(p)
				if err != nil {
					if os.IsNotExist(err) {
						return nil
					}
					return err
				}
				for _, e := range ents {
					fp := filepath.Join(p, e.Name())
					if e.IsDir() {
						if strings.HasSuffix(e.Name(), ".git") {
							count++
						}
						if err := walk(fp); err != nil {
							return err
						}
					} else {
						fi, err := os.Stat(fp)
						if err == nil {
							size += fi.Size()
						}
					}
				}
				return nil
			}
			if err := walk(root); err != nil {
				return err
			}
			fmt.Println("fs_repos", count)
			fmt.Println("fs_bytes", size)
			return nil
		},
	}
	return cmd
}
