package repository

import (
	"context"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/google/uuid"
)

// SSHKeyRepository defines the interface for SSH key data access operations
type SSHKeyRepository interface {
	// Create creates a new SSH key in the database
	Create(ctx context.Context, key *models.SSHKey) error

	// FindByID retrieves an SSH key by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.SSHKey, error)

	// FindByFingerprint retrieves an SSH key by its fingerprint
	FindByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error)

	// FindByUserID retrieves all SSH keys for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error)

	// Update updates an existing SSH key
	Update(ctx context.Context, key *models.SSHKey) error

	// Delete removes an SSH key from the database by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID removes all SSH keys for a user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// ExistsByFingerprint checks if an SSH key with the given fingerprint exists
	ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error)

	// UpdateLastUsed updates the last_used_at timestamp for an SSH key
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// CountByUserID returns the number of SSH keys for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
