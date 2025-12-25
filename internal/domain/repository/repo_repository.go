package repository

import (
	"context"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/google/uuid"
)

// RepoRepository defines the interface for repository data access
type RepoRepository interface {
	// Create creates a new repository
	Create(ctx context.Context, repo *models.Repository) error

	// FindByID finds a repository by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// FindByOwnerAndName finds a repository by owner ID and name
	FindByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error)

	// FindByOwnerUsernameAndName finds a repository by owner username and name
	FindByOwnerUsernameAndName(ctx context.Context, username, name string) (*models.Repository, error)

	// FindByOwner finds all repositories owned by a user
	FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Repository, error)

	// ListPublic lists public repositories with pagination
	ListPublic(ctx context.Context, limit, offset int) ([]*models.Repository, error)

	// ListAll lists all repositories with pagination (for admin use)
	ListAll(ctx context.Context, limit, offset int) ([]*models.Repository, error)

	// Update updates a repository
	Update(ctx context.Context, repo *models.Repository) error

	// Delete deletes a repository by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByOwnerAndName checks if a repository exists with the given owner and name
	ExistsByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (bool, error)

	// CountByOwner returns the count of repositories owned by a user
	CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
}
