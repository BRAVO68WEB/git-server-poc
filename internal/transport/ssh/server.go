package ssh

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"

	"github.com/bravo68web/githut/internal/application/service"
	"github.com/bravo68web/githut/internal/config"
	"github.com/bravo68web/githut/internal/domain/models"
	domainservice "github.com/bravo68web/githut/internal/domain/service"
	"github.com/bravo68web/githut/internal/infrastructure/git"
)

// Server represents the SSH server for Git operations
type Server struct {
	server      *ssh.Server
	config      *config.SSHConfig
	authService domainservice.AuthService
	repoService *service.RepoService
	gitProtocol *git.GitProtocol
	storage     domainservice.StorageService
}

// NewServer creates a new SSH server instance
func NewServer(
	cfg *config.SSHConfig,
	storageCfg *config.StorageConfig,
	authService domainservice.AuthService,
	repoService *service.RepoService,
	storage domainservice.StorageService,
) (*Server, error) {
	s := &Server{
		config:      cfg,
		authService: authService,
		repoService: repoService,
		gitProtocol: git.NewGitProtocol(),
		storage:     storage,
	}

	// Create the wish server with options
	server, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithPublicKeyAuth(s.publicKeyHandler),
		wish.WithMiddleware(
			s.gitMiddleware,
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	s.server = server
	return s, nil
}

// publicKeyHandler handles public key authentication
func (s *Server) publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	// Convert charmbracelet/ssh.PublicKey to golang.org/x/crypto/ssh.PublicKey
	// Both implement the same interface, so we can use gossh.FingerprintSHA256
	fingerprint := gossh.FingerprintSHA256(key)

	// Authenticate using the fingerprint
	user, err := s.authService.AuthenticateSSH(context.Background(), []byte(fingerprint))
	if err != nil {
		log.Printf("SSH auth failed for key %s: %v", fingerprint, err)
		return false
	}

	// Store user info in context for later use
	ctx.SetValue("user", user)
	ctx.SetValue("fingerprint", fingerprint)

	log.Printf("SSH auth successful for user %s with key %s", user.Username, fingerprint)
	return true
}

// gitMiddleware handles Git SSH protocol commands
func (s *Server) gitMiddleware(next ssh.Handler) ssh.Handler {
	return func(sess ssh.Session) {
		cmd := sess.Command()

		// If no command, show welcome message
		if len(cmd) == 0 {
			s.handleWelcome(sess)
			return
		}

		// Parse Git command
		if len(cmd) < 2 {
			fmt.Fprintf(sess.Stderr(), "Invalid command\n")
			sess.Exit(1)
			return
		}

		gitCmd := cmd[0]
		repoPath := cmd[1]

		// Validate Git command
		if gitCmd != "git-upload-pack" && gitCmd != "git-receive-pack" && gitCmd != "git-upload-archive" {
			fmt.Fprintf(sess.Stderr(), "Unknown command: %s\n", gitCmd)
			sess.Exit(1)
			return
		}

		// Handle Git operation
		if err := s.handleGitCommand(sess, gitCmd, repoPath); err != nil {
			log.Printf("Git command error: %v", err)
			fmt.Fprintf(sess.Stderr(), "Error: %v\n", err)
			sess.Exit(1)
			return
		}

		sess.Exit(0)
	}
}

// handleWelcome displays a welcome message for interactive SSH connections
func (s *Server) handleWelcome(sess ssh.Session) {
	user := s.getUserFromSession(sess)
	if user != nil {
		fmt.Fprintf(sess, "Hi %s! You've successfully authenticated.\n", user.Username)
	} else {
		fmt.Fprintf(sess, "Welcome to Git SSH Server!\n")
	}
	fmt.Fprintf(sess, "This server only accepts Git commands.\n")
	fmt.Fprintf(sess, "\nUsage:\n")
	fmt.Fprintf(sess, "  git clone ssh://<host>:<port>/<owner>/<repo>.git\n")
	fmt.Fprintf(sess, "  git push origin <branch>\n")
	fmt.Fprintf(sess, "  git pull origin <branch>\n")
}

// handleGitCommand processes Git SSH protocol commands
func (s *Server) handleGitCommand(sess ssh.Session, gitCmd, repoPath string) error {
	ctx := context.Background()

	// Parse repository path (format: /owner/repo.git or owner/repo.git)
	repoPath = strings.TrimPrefix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	repoPath = strings.Trim(repoPath, "'\"")

	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository path: %s (expected owner/repo)", repoPath)
	}

	owner := parts[0]
	repoName := parts[1]

	// Get repository
	repo, err := s.repoService.GetRepository(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("repository not found: %s/%s", owner, repoName)
	}

	// Check access permissions
	user := s.getUserFromSession(sess)
	isWriteOperation := gitCmd == "git-receive-pack"

	if !s.checkRepoAccess(user, repo, isWriteOperation) {
		if repo.IsPrivate {
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("permission denied")
	}

	// Execute Git command
	switch gitCmd {
	case "git-upload-pack":
		return s.gitProtocol.HandleUploadPackSSH(ctx, repo.GitPath, sess, sess)
	case "git-receive-pack":
		return s.gitProtocol.HandleReceivePackSSH(ctx, repo.GitPath, sess, sess)
	case "git-upload-archive":
		// git-upload-archive is less common but can be supported
		return fmt.Errorf("git-upload-archive is not supported")
	default:
		return fmt.Errorf("unknown git command: %s", gitCmd)
	}
}

// getUserFromSession retrieves the authenticated user from the session context
func (s *Server) getUserFromSession(sess ssh.Session) *models.User {
	ctx := sess.Context()
	if user, ok := ctx.Value("user").(*models.User); ok {
		return user
	}
	return nil
}

// checkRepoAccess checks if the user can access the repository
func (s *Server) checkRepoAccess(user *models.User, repo *models.Repository, isWrite bool) bool {
	// Public repos allow read access to everyone
	if !repo.IsPrivate && !isWrite {
		return true
	}

	// Must be authenticated for private repos or write access
	if user == nil {
		return false
	}

	// Check if user has access
	if user.IsAdmin {
		return true
	}

	if user.ID == repo.OwnerID {
		return true
	}

	// Read access to public repos for authenticated users
	if !isWrite && !repo.IsPrivate {
		return true
	}

	// TODO: Check collaborator permissions

	return false
}

// ListenAndServe starts the SSH server
func (s *Server) ListenAndServe() error {
	log.Printf("Starting SSH server on %s", s.config.Address())
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the SSH server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down SSH server...")
	return s.server.Shutdown(ctx)
}

// Address returns the server address
func (s *Server) Address() string {
	return s.config.Address()
}

// ShutdownWithTimeout shuts down the server with a timeout
func (s *Server) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}
