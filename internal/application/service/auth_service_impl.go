package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bravo68web/githut/internal/config"
	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	"github.com/bravo68web/githut/internal/domain/service"
	apperrors "github.com/bravo68web/githut/pkg/errors"
	"github.com/google/uuid"
)

// AuthServiceImpl implements the AuthService interface
type AuthServiceImpl struct {
	userRepo    repository.UserRepository
	sshKeyRepo  repository.SSHKeyRepository
	oidcService *OIDCService
	config      *config.OIDCConfig
}

// NewAuthService creates a new AuthServiceImpl instance
func NewAuthService(
	userRepo repository.UserRepository,
	sshKeyRepo repository.SSHKeyRepository,
	oidcService *OIDCService,
	oidcConfig *config.OIDCConfig,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:    userRepo,
		sshKeyRepo:  sshKeyRepo,
		oidcService: oidcService,
		config:      oidcConfig,
	}
}

// AuthenticateToken authenticates a user using an access token (PAT)
func (s *AuthServiceImpl) AuthenticateToken(ctx context.Context, token string) (*models.User, error) {
	// TODO: Implement PAT (Personal Access Token) authentication
	// For now, this is a placeholder that will be implemented when PAT feature is added
	// The token should be hashed and looked up in a tokens table

	// Hash the token to look it up
	_ = s.hashToken(token)

	// Placeholder: PAT authentication not yet implemented
	return nil, apperrors.Unauthorized("token authentication not implemented", apperrors.ErrInvalidCredentials)
}

// AuthenticateSSH authenticates a user using their SSH public key fingerprint
// The publicKey parameter should be the SSH key fingerprint (e.g., SHA256:xxx format)
func (s *AuthServiceImpl) AuthenticateSSH(ctx context.Context, publicKey []byte) (*models.User, error) {
	fingerprint := string(publicKey)

	// Find SSH key by fingerprint
	sshKey, err := s.sshKeyRepo.FindByFingerprint(ctx, fingerprint)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.Unauthorized("ssh key not recognized", apperrors.ErrInvalidCredentials)
		}
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
			return nil, apperrors.Unauthorized("user not found for ssh key", apperrors.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("failed to find user for ssh key: %w", err)
	}

	return user, nil
}

// AuthenticateSSHByFingerprint authenticates a user using their SSH public key fingerprint string
func (s *AuthServiceImpl) AuthenticateSSHByFingerprint(ctx context.Context, fingerprint string) (*models.User, error) {
	return s.AuthenticateSSH(ctx, []byte(fingerprint))
}

// AuthenticateSession authenticates a user using a session JWT (from OIDC login)
func (s *AuthServiceImpl) AuthenticateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	if s.oidcService == nil {
		return nil, apperrors.Unauthorized("OIDC not configured", apperrors.ErrInvalidCredentials)
	}

	// Validate the session token
	claims, err := s.oidcService.ValidateSessionToken(sessionToken)
	if err != nil {
		return nil, apperrors.Unauthorized("invalid session token", err)
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperrors.Unauthorized("invalid user ID in session", err)
	}

	// Get the user from database
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.Unauthorized("user not found", apperrors.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// hashToken creates a SHA256 hash of the token for secure storage
func (s *AuthServiceImpl) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Verify interface compliance at compile time
var _ service.AuthService = (*AuthServiceImpl)(nil)
