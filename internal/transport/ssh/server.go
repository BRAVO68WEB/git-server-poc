package ssh

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gossh "golang.org/x/crypto/ssh"

	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/models"
	domainservice "github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/internal/infrastructure/git"
	"github.com/bravo68web/stasis/pkg/logger"
)

// Server represents the SSH server for Git operations
type Server struct {
	server      *ssh.Server
	config      *config.SSHConfig
	authService domainservice.AuthService
	repoService *service.RepoService
	gitProtocol *git.GitProtocol
	storage     domainservice.StorageService
	log         *logger.Logger
}

// NewServer creates a new SSH server instance
func NewServer(
	cfg *config.SSHConfig,
	storageCfg *config.StorageConfig,
	authService domainservice.AuthService,
	repoService *service.RepoService,
	storage domainservice.StorageService,
) (*Server, error) {
	log := logger.Get().WithFields(logger.Component("ssh-server"))

	log.Info("Creating SSH server...",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("host_key_path", cfg.HostKeyPath),
	)

	s := &Server{
		config:      cfg,
		authService: authService,
		repoService: repoService,
		gitProtocol: git.NewGitProtocol(),
		storage:     storage,
		log:         log,
	}

	// Create the wish server with options
	server, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithPublicKeyAuth(s.publicKeyHandler),
		wish.WithMiddleware(
			s.gitMiddleware,
			s.loggingMiddleware,
		),
	)
	if err != nil {
		log.Error("Failed to create SSH server",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	s.server = server

	log.Info("SSH server created successfully",
		logger.String("address", cfg.Address()),
	)

	return s, nil
}

// loggingMiddleware logs SSH session information
func (s *Server) loggingMiddleware(next ssh.Handler) ssh.Handler {
	return func(sess ssh.Session) {
		start := time.Now()
		user := s.getUserFromSession(sess)

		username := "anonymous"
		if user != nil {
			username = user.Username
		}

		s.log.Info("SSH session started",
			logger.String("session_id", sess.Context().SessionID()),
			logger.String("remote_addr", sess.RemoteAddr().String()),
			logger.String("user", username),
			logger.Strings("command", sess.Command()),
		)

		// Call next handler
		next(sess)

		// Log session end
		duration := time.Since(start)
		s.log.Info("SSH session ended",
			logger.String("session_id", sess.Context().SessionID()),
			logger.String("user", username),
			logger.Duration("duration", duration),
		)
	}
}

// publicKeyHandler handles public key authentication
func (s *Server) publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	// Convert charmbracelet/ssh.PublicKey to golang.org/x/crypto/ssh.PublicKey
	// Both implement the same interface, so we can use gossh.FingerprintSHA256
	fingerprint := gossh.FingerprintSHA256(key)

	s.log.Debug("SSH authentication attempt",
		logger.String("fingerprint", fingerprint),
		logger.String("remote_addr", ctx.RemoteAddr().String()),
		logger.String("key_type", key.Type()),
	)

	// Authenticate using the fingerprint
	user, err := s.authService.AuthenticateSSH(context.Background(), []byte(fingerprint))
	if err != nil {
		s.log.Warn("SSH authentication failed",
			logger.String("fingerprint", fingerprint),
			logger.String("remote_addr", ctx.RemoteAddr().String()),
			logger.Error(err),
		)
		return false
	}

	// Store user info in context for later use
	ctx.SetValue("user", user)
	ctx.SetValue("fingerprint", fingerprint)

	s.log.Info("SSH authentication successful",
		logger.String("user", user.Username),
		logger.String("user_id", user.ID.String()),
		logger.String("fingerprint", fingerprint),
		logger.String("remote_addr", ctx.RemoteAddr().String()),
	)

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
			s.log.Warn("Invalid SSH command - insufficient arguments",
				logger.String("session_id", sess.Context().SessionID()),
				logger.Strings("command", cmd),
			)
			fmt.Fprintf(sess.Stderr(), "Invalid command\n")
			sess.Exit(1)
			return
		}

		gitCmd := cmd[0]
		repoPath := cmd[1]

		// Validate Git command
		if gitCmd != "git-upload-pack" && gitCmd != "git-receive-pack" && gitCmd != "git-upload-archive" {
			s.log.Warn("Unknown Git command",
				logger.String("session_id", sess.Context().SessionID()),
				logger.String("command", gitCmd),
			)
			fmt.Fprintf(sess.Stderr(), "Unknown command: %s\n", gitCmd)
			sess.Exit(1)
			return
		}

		s.log.Debug("Processing Git command",
			logger.String("session_id", sess.Context().SessionID()),
			logger.String("git_cmd", gitCmd),
			logger.String("repo_path", repoPath),
		)

		// Handle Git operation
		if err := s.handleGitCommand(sess, gitCmd, repoPath); err != nil {
			s.log.Error("Git command failed",
				logger.String("session_id", sess.Context().SessionID()),
				logger.String("git_cmd", gitCmd),
				logger.String("repo_path", repoPath),
				logger.Error(err),
			)
			fmt.Fprintf(sess.Stderr(), "Error: %v\n", err)
			sess.Exit(1)
			return
		}

		s.log.Info("Git command completed successfully",
			logger.String("session_id", sess.Context().SessionID()),
			logger.String("git_cmd", gitCmd),
			logger.String("repo_path", repoPath),
		)

		sess.Exit(0)
	}
}

// handleWelcome displays a welcome message for interactive SSH connections
func (s *Server) handleWelcome(sess ssh.Session) {
	user := s.getUserFromSession(sess)

	s.log.Debug("Displaying welcome message",
		logger.String("session_id", sess.Context().SessionID()),
	)

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
		s.log.Warn("Invalid repository path format",
			logger.String("session_id", sess.Context().SessionID()),
			logger.String("repo_path", repoPath),
		)
		return fmt.Errorf("invalid repository path: %s (expected owner/repo)", repoPath)
	}

	owner := parts[0]
	repoName := parts[1]

	s.log.Debug("Looking up repository",
		logger.String("owner", owner),
		logger.String("repo", repoName),
	)

	// Get repository
	repo, err := s.repoService.GetRepository(ctx, owner, repoName)
	if err != nil {
		s.log.Warn("Repository not found",
			logger.String("owner", owner),
			logger.String("repo", repoName),
			logger.Error(err),
		)
		return fmt.Errorf("repository not found: %s/%s", owner, repoName)
	}

	// Check access permissions
	user := s.getUserFromSession(sess)
	isWriteOperation := gitCmd == "git-receive-pack"

	username := "anonymous"
	if user != nil {
		username = user.Username
	}

	s.log.Debug("Checking repository access",
		logger.String("user", username),
		logger.String("repo", repo.Name),
		logger.Bool("is_private", repo.IsPrivate),
		logger.Bool("is_write", isWriteOperation),
	)

	if !s.checkRepoAccess(user, repo, isWriteOperation) {
		s.log.Warn("Repository access denied",
			logger.String("user", username),
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
			logger.Bool("is_write", isWriteOperation),
			logger.Bool("is_private", repo.IsPrivate),
		)
		if repo.IsPrivate {
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("permission denied")
	}

	s.log.Info("Executing Git operation",
		logger.String("user", username),
		logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
		logger.String("operation", gitCmd),
		logger.String("git_path", repo.GitPath),
	)

	// Execute Git command
	switch gitCmd {
	case "git-upload-pack":
		return s.gitProtocol.HandleUploadPackSSH(ctx, repo.GitPath, sess, sess)
	case "git-receive-pack":
		return s.gitProtocol.HandleReceivePackSSH(ctx, repo.GitPath, sess, sess)
	case "git-upload-archive":
		s.log.Warn("Unsupported Git command: git-upload-archive",
			logger.String("session_id", sess.Context().SessionID()),
		)
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
	s.log.Info("Starting SSH server",
		logger.String("address", s.config.Address()),
	)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the SSH server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down SSH server...")

	if err := s.server.Shutdown(ctx); err != nil {
		s.log.Error("Error shutting down SSH server",
			logger.Error(err),
		)
		return err
	}

	s.log.Info("SSH server shutdown complete")
	return nil
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
