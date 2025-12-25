package service

import (
	"context"

	"github.com/bravo68web/stasis/internal/domain/models"
)

// AuthService defines the interface for authentication and authorization operations
type AuthService interface {
	// User authentication methods

	// AuthenticateToken authenticates a user using an access token (PAT)
	// Returns the authenticated user or an error if the token is invalid or expired
	AuthenticateToken(ctx context.Context, token string) (*models.User, error)

	// AuthenticateSSH authenticates a user using their SSH public key
	// Returns the authenticated user or an error if the key is not recognized
	AuthenticateSSH(ctx context.Context, publicKey []byte) (*models.User, error)

	// AuthenticateSession authenticates a user using a session JWT (from OIDC login)
	// Returns the authenticated user or an error if the session is invalid or expired
	AuthenticateSession(ctx context.Context, sessionToken string) (*models.User, error)
}
