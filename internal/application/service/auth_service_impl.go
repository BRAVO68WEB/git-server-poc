package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/internal/domain/service"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/google/uuid"
)

// AuthServiceImpl implements the AuthService interface
type AuthServiceImpl struct {
	userRepo    repository.UserRepository
	sshKeyRepo  repository.SSHKeyRepository
	tokenRepo   repository.TokenRepository
	oidcService *OIDCService
	config      *config.OIDCConfig
	log         *logger.Logger
}

// NewAuthService creates a new AuthServiceImpl instance
func NewAuthService(
	userRepo repository.UserRepository,
	sshKeyRepo repository.SSHKeyRepository,
	tokenRepo repository.TokenRepository,
	oidcService *OIDCService,
	oidcConfig *config.OIDCConfig,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:    userRepo,
		sshKeyRepo:  sshKeyRepo,
		tokenRepo:   tokenRepo,
		oidcService: oidcService,
		config:      oidcConfig,
		log:         logger.Get().WithFields(logger.Component("auth-service")),
	}
}

// AuthenticateToken authenticates a user using an access token (PAT)
func (s *AuthServiceImpl) AuthenticateToken(ctx context.Context, token string) (*models.User, error) {
	s.log.Debug("Authenticating user via access token (PAT)")

	// Hash the token to look it up
	hashedToken := s.hashToken(token)

	// Find the token in the database
	tokenRecord, err := s.tokenRepo.FindByHashedToken(ctx, hashedToken)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Debug("Token not found in database")
			return nil, apperrors.Unauthorized("invalid token", apperrors.ErrInvalidCredentials)
		}
		s.log.Error("Failed to find token in database",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to find token: %w", err)
	}

	// Check if token is expired
	if tokenRecord.ExpiresAt != nil && tokenRecord.ExpiresAt.Before(time.Now()) {
		s.log.Debug("Token has expired",
			logger.String("token_id", tokenRecord.ID.String()),
			logger.Time("expired_at", *tokenRecord.ExpiresAt),
		)
		return nil, apperrors.Unauthorized("token has expired", apperrors.ErrInvalidCredentials)
	}

	// Update last used timestamp (fire and forget)
	go func() {
		_ = s.tokenRepo.UpdateLastUsed(context.Background(), tokenRecord.ID)
	}()

	// Get the user associated with this token
	user, err := s.userRepo.FindByID(ctx, tokenRecord.UserID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Warn("User not found for valid token",
				logger.String("token_id", tokenRecord.ID.String()),
				logger.String("user_id", tokenRecord.UserID.String()),
			)
			return nil, apperrors.Unauthorized("user not found for token", apperrors.ErrInvalidCredentials)
		}
		s.log.Error("Failed to find user for token",
			logger.Error(err),
			logger.String("user_id", tokenRecord.UserID.String()),
		)
		return nil, fmt.Errorf("failed to find user for token: %w", err)
	}

	s.log.Info("User authenticated via PAT",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
		logger.String("token_id", tokenRecord.ID.String()),
	)

	return user, nil
}

// AuthenticateSSH authenticates a user using their SSH public key fingerprint
// The publicKey parameter should be the SSH key fingerprint (e.g., SHA256:xxx format)
func (s *AuthServiceImpl) AuthenticateSSH(ctx context.Context, publicKey []byte) (*models.User, error) {
	fingerprint := string(publicKey)

	s.log.Debug("Authenticating user via SSH key",
		logger.String("fingerprint", fingerprint),
	)

	// Find SSH key by fingerprint
	sshKey, err := s.sshKeyRepo.FindByFingerprint(ctx, fingerprint)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Debug("SSH key not found",
				logger.String("fingerprint", fingerprint),
			)
			return nil, apperrors.Unauthorized("ssh key not recognized", apperrors.ErrInvalidCredentials)
		}
		s.log.Error("Failed to find SSH key",
			logger.Error(err),
			logger.String("fingerprint", fingerprint),
		)
		return nil, fmt.Errorf("failed to find ssh key: %w", err)
	}

	// Update last used timestamp (fire and forget, don't fail auth on this)
	go func() {
		_ = s.sshKeyRepo.UpdateLastUsed(context.Background(), sshKey.ID)
	}()

	// Get the user associated with this key
	user, err := s.userRepo.FindByID(ctx, sshKey.UserID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Warn("User not found for valid SSH key",
				logger.String("ssh_key_id", sshKey.ID.String()),
				logger.String("user_id", sshKey.UserID.String()),
			)
			return nil, apperrors.Unauthorized("user not found for ssh key", apperrors.ErrInvalidCredentials)
		}
		s.log.Error("Failed to find user for SSH key",
			logger.Error(err),
			logger.String("user_id", sshKey.UserID.String()),
		)
		return nil, fmt.Errorf("failed to find user for ssh key: %w", err)
	}

	s.log.Info("User authenticated via SSH key",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
		logger.String("ssh_key_id", sshKey.ID.String()),
	)

	return user, nil
}

// AuthenticateSSHByFingerprint authenticates a user using their SSH public key fingerprint string
func (s *AuthServiceImpl) AuthenticateSSHByFingerprint(ctx context.Context, fingerprint string) (*models.User, error) {
	return s.AuthenticateSSH(ctx, []byte(fingerprint))
}

// AuthenticateSession authenticates a user using a session JWT (from OIDC login)
func (s *AuthServiceImpl) AuthenticateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	s.log.Debug("Authenticating user via session token")

	if s.oidcService == nil {
		s.log.Debug("OIDC service not configured")
		return nil, apperrors.Unauthorized("OIDC not configured", apperrors.ErrInvalidCredentials)
	}

	// Validate the session token
	claims, err := s.oidcService.ValidateSessionToken(sessionToken)
	if err != nil {
		s.log.Debug("Invalid session token",
			logger.Error(err),
		)
		return nil, apperrors.Unauthorized("invalid session token", err)
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.log.Warn("Invalid user ID in session token",
			logger.String("user_id_claim", claims.UserID),
			logger.Error(err),
		)
		return nil, apperrors.Unauthorized("invalid user ID in session", err)
	}

	// Get the user from database
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Warn("User not found for valid session token",
				logger.String("user_id", userID.String()),
			)
			return nil, apperrors.Unauthorized("user not found", apperrors.ErrInvalidCredentials)
		}
		s.log.Error("Failed to find user for session token",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	s.log.Info("User authenticated via session token",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
	)

	return user, nil
}

// hashToken creates a SHA256 hash of the token for secure storage
func (s *AuthServiceImpl) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Verify interface compliance at compile time
var _ service.AuthService = (*AuthServiceImpl)(nil)
