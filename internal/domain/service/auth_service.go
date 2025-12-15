package service

import (
	"context"

	"github.com/bravo68web/githut/internal/domain/models"
)

// AuthService defines the interface for authentication and authorization operations
type AuthService interface {
	// User authentication methods

	// AuthenticateBasic authenticates a user using username and password
	// Returns the authenticated user or an error if authentication fails
	AuthenticateBasic(ctx context.Context, username, password string) (*models.User, error)

	// AuthenticateToken authenticates a user using an access token
	// Returns the authenticated user or an error if the token is invalid or expired
	AuthenticateToken(ctx context.Context, token string) (*models.User, error)

	// AuthenticateSSH authenticates a user using their SSH public key
	// Returns the authenticated user or an error if the key is not recognized
	AuthenticateSSH(ctx context.Context, publicKey []byte) (*models.User, error)

	// Password operations

	// HashPassword generates a secure hash from a plain text password
	HashPassword(password string) (string, error)

	// VerifyPassword compares a hashed password with a plain text password
	// Returns nil if they match, or an error if they don't
	VerifyPassword(hash, password string) error
}
