package repository

import (
	"context"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access operations
type UserRepository interface {
	// Create creates a new user in the database
	Create(ctx context.Context, user *models.User) error

	// FindByID retrieves a user by their ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// FindByUsername retrieves a user by their username
	FindByUsername(ctx context.Context, username string) (*models.User, error)

	// FindByEmail retrieves a user by their email address
	FindByEmail(ctx context.Context, email string) (*models.User, error)

	// Update updates an existing user's information
	Update(ctx context.Context, user *models.User) error

	// Delete removes a user from the database by their ID
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all users with pagination
	List(ctx context.Context, limit, offset int) ([]*models.User, error)

	// Count returns the total number of users
	Count(ctx context.Context) (int64, error)

	// ExistsByUsername checks if a user with the given username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if a user with the given email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// FindByOIDCSubject retrieves a user by their OIDC subject and issuer
	FindByOIDCSubject(ctx context.Context, subject, issuer string) (*models.User, error)
}
