package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
)

// TokenService handles personal access token operations
type TokenService struct {
	tokenRepo repository.TokenRepository
	userRepo  repository.UserRepository
}

// NewTokenService creates a new TokenService instance
func NewTokenService(
	tokenRepo repository.TokenRepository,
	userRepo repository.UserRepository,
) *TokenService {
	return &TokenService{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
	}
}

// CreateTokenRequest represents a request to create a new PAT
type CreateTokenRequest struct {
	UserID    uuid.UUID
	Name      string
	Scopes    []string   // empty = all repos access
	ExpiresAt *time.Time // nil = never expires
}

// CreateTokenResponse represents the response after creating a PAT
type CreateTokenResponse struct {
	Token    *models.Token
	RawToken string // The unhashed token to return to user (only shown once)
}

// CreateToken creates a new personal access token for a user
func (s *TokenService) CreateToken(ctx context.Context, req CreateTokenRequest) (*CreateTokenResponse, error) {
	// Validate name
	if req.Name == "" {
		return nil, apperrors.BadRequest("token name is required", apperrors.ErrInvalidInput)
	}

	// Generate random token: Sx{32 random hex chars}
	rawToken, err := generateRawToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token for storage
	hashedToken := hashToken(rawToken)

	// Create the token record
	token := &models.Token{
		Name:      req.Name,
		UserID:    req.UserID,
		Token:     hashedToken,
		Scope:     pq.StringArray(req.Scopes),
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	return &CreateTokenResponse{
		Token:    token,
		RawToken: rawToken,
	}, nil
}

// ListTokens returns all tokens for a user (with token values redacted)
func (s *TokenService) ListTokens(ctx context.Context, userID uuid.UUID) ([]*models.Token, error) {
	tokens, err := s.tokenRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	return tokens, nil
}

// DeleteToken removes a token
func (s *TokenService) DeleteToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	// Get the token first to verify ownership
	token, err := s.tokenRepo.FindByID(ctx, tokenID)
	if err != nil {
		return err
	}

	// Verify the token belongs to the user
	if token.UserID != userID {
		return apperrors.Forbidden("you do not own this token", apperrors.ErrForbidden)
	}

	return s.tokenRepo.Delete(ctx, tokenID)
}

// ValidateToken validates a raw token and returns the associated user if valid
func (s *TokenService) ValidateToken(ctx context.Context, rawToken string) (*models.User, *models.Token, error) {
	// Hash the raw token to look it up
	hashedToken := hashToken(rawToken)

	// Find the token
	token, err := s.tokenRepo.FindByHashedToken(ctx, hashedToken)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, nil, apperrors.Unauthorized("invalid token", apperrors.ErrInvalidCredentials)
		}
		return nil, nil, fmt.Errorf("failed to find token: %w", err)
	}

	// Check if token is expired
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, nil, apperrors.Unauthorized("token has expired", apperrors.ErrInvalidCredentials)
	}

	// Update last used timestamp (fire and forget)
	go func() {
		_ = s.tokenRepo.UpdateLastUsed(context.Background(), token.ID)
	}()

	// Get the user
	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, nil, apperrors.Unauthorized("user not found", apperrors.ErrInvalidCredentials)
		}
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, token, nil
}

// CheckTokenScope checks if a token has access to a specific repo
// Returns true if token has access (either explicit scope or empty scope = all access)
func (s *TokenService) CheckTokenScope(token *models.Token, ownerRepo string) bool {
	// Empty scope means access to all repos
	if len(token.Scope) == 0 {
		return true
	}

	// Check if the owner/repo is in the scope list
	return slices.Contains(token.Scope, ownerRepo)
}

// generateRawToken generates a new token in format Sx{32 random hex chars}
func generateRawToken() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 32 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "Sx" + hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA256 hash of the token for secure storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
