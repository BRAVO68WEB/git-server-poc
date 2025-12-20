package repository

import (
	"context"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/google/uuid"
)

// TokenRepository defines the interface for personal access token data access operations
type TokenRepository interface {
	// Create creates a new token in the database
	Create(ctx context.Context, token *models.Token) error

	// FindByID retrieves a token by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Token, error)

	// FindByHashedToken retrieves a token by its hashed value
	FindByHashedToken(ctx context.Context, hashedToken string) (*models.Token, error)

	// FindByUserID retrieves all tokens for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Token, error)

	// Delete removes a token from the database by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID removes all tokens for a user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// UpdateLastUsed updates the last_used timestamp for a token
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// CountByUserID returns the number of tokens for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
