package git

// import (
// 	"context"
// 	"crypto/rand"
// 	"crypto/rsa"
// 	"log"
// 	"net"
// 	"os/exec"
// 	"strings"

// 	"githut/internal/database"

// 	"golang.org/x/crypto/ssh"
// )

// func StartSSH(ctx context.Context, addr string, db *database.DB) error {
// 	if addr == "" {
// 		return nil
// 	}
// 	key, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		return err
// 	}
// 	signer, err := ssh.NewSignerFromKey(key)
// 	if err != nil {
// 		return err
// 	}
// 	conf := &ssh.ServerConfig{
// 		PublicKeyCallback: func(connMeta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
// 			authkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
// 			u, err := db.GetUserByPublicKey(ctx, authkey)
// 			if err != nil || u.Disabled {
// 				return nil, err
// 			}
// 			return &ssh.Permissions{
// 				Extensions: map[string]string{
// 					"user": u.Username,
// 				},
// 			}, nil
// 		},
// 	}
// 	conf.AddHostKey(signer)
// 	l, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		return err
// 	}
// 	go func() {
// 		log.Printf("ssh listening on %s", addr)
// 		for {
// 			nc, err := l.Accept()
// 			if err != nil {
// 				log.Printf("ssh accept error: %v", err)
// 				continue
// 			}
// 			go handleSSHConn(ctx, nc, conf, db)
// 		}
// 	}()
// 	return nil
// }

// func handleSSHConn(ctx context.Context, nc net.Conn, conf *ssh.ServerConfig, db *database.DB) {
// 	sconn, chans, reqs, err := ssh.NewServerConn(nc, conf)
// 	if err != nil {
// 		return
// 	}
// 	defer sconn.Close()
// 	go ssh.DiscardRequests(reqs)
// 	for ch := range chans {
// 		if ch.ChannelType() != "session" {
// 			ch.Reject(ssh.UnknownChannelType, "unknown channel type")
// 			continue
// 		}
// 		c, reqs, err := ch.Accept()
// 		if err != nil {
// 			continue
// 		}
// 		go func(in <-chan *ssh.Request) {
// 			for req := range in {
// 				switch req.Type {
// 				case "exec":
// 					var payload struct{ Value string }
// 					ssh.Unmarshal(req.Payload, &payload)
// 					cmdline := payload.Value
// 					req.Reply(true, nil)
// 					runSSHCommand(ctx, c, cmdline, db, sconn.Permissions)
// 				default:
// 					req.Reply(false, nil)
// 				}
// 			}
// 		}(reqs)
// 	}
// }

// func runSSHCommand(ctx context.Context, c ssh.Channel, cmdline string, db *database.DB, perm *ssh.Permissions) {
// 	defer c.Close()
// 	fields := strings.Fields(cmdline)
// 	if len(fields) < 2 {
// 		return
// 	}
// 	svc := fields[0]
// 	arg := strings.Trim(fields[1], "'\"")
// 	owner, repo := parseRepoArg(arg)
// 	repoPath := ensureLocalRepo(owner, repo)
// 	username := ""
// 	if perm != nil && perm.Extensions != nil {
// 		username = perm.Extensions["user"]
// 	}
// 	switch svc {
// 	case "git-upload-pack":
// 		ok, err := db.HasPullAccess(ctx, username, owner, repo)
// 		if err != nil || !ok {
// 			return
// 		}
// 		cmd := exec.CommandContext(ctx, "git-upload-pack", repoPath)
// 		cmd.Stdin = c
// 		cmd.Stdout = c
// 		_ = cmd.Run()
// 	case "git-receive-pack":
// 		ok, err := db.HasPushAccess(ctx, username, owner, repo)
// 		if err != nil || !ok {
// 			return
// 		}
// 		cmd := exec.CommandContext(ctx, "git-receive-pack", repoPath)
// 		cmd.Stdin = c
// 		cmd.Stdout = c
// 		_ = cmd.Run()
// 	}
// }

// func parseRepoArg(arg string) (string, string) {
// 	// expect '/repos/<owner>/<name>.git'
// 	arg = strings.TrimSpace(arg)
// 	arg = strings.TrimPrefix(arg, "/")
// 	parts := strings.Split(arg, "/")
// 	for i := range parts {
// 		parts[i] = strings.TrimSpace(parts[i])
// 	}
// 	var owner, name string
// 	for i := 0; i+1 < len(parts); i++ {
// 		if parts[i] == "repos" {
// 			owner = parts[i+1]
// 			if i+2 < len(parts) {
// 				name = strings.TrimSuffix(parts[i+2], ".git")
// 			}
// 			break
// 		}
// 	}
// 	return owner, name
// }
