package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	"github.com/bravo68web/githut/internal/domain/service"
	apperrors "github.com/bravo68web/githut/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// AuthServiceImpl implements the AuthService interface
type AuthServiceImpl struct {
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthServiceImpl instance
func NewAuthService(
	userRepo repository.UserRepository,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo: userRepo,
	}
}

// AuthenticateBasic authenticates a user using username and password
func (s *AuthServiceImpl) AuthenticateBasic(ctx context.Context, username, password string) (*models.User, error) {
	// Find user by username
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.Unauthorized("invalid credentials", apperrors.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := s.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, apperrors.Unauthorized("invalid credentials", apperrors.ErrInvalidCredentials)
	}

	return user, nil
}

// AuthenticateToken authenticates a user using an access token
func (s *AuthServiceImpl) AuthenticateToken(ctx context.Context, token string) (*models.User, error) {
	panic("not implemented")
}

// AuthenticateSSH authenticates a user using their SSH public key
func (s *AuthServiceImpl) AuthenticateSSH(ctx context.Context, publicKey []byte) (*models.User, error) {
	panic("not implemented")
}

// HashPassword generates a secure hash from a plain text password
func (s *AuthServiceImpl) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a hashed password with a plain text password
func (s *AuthServiceImpl) VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// hashToken creates a SHA256 hash of the token for secure storage
func (s *AuthServiceImpl) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Verify interface compliance at compile time
var _ service.AuthService = (*AuthServiceImpl)(nil)
