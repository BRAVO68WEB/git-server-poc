package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	apperror "github.com/bravo68web/githut/pkg/errors"
	"github.com/google/uuid"
)

// SSHKeyRepoImpl implements the SSHKeyRepository interface using GORM
type SSHKeyRepoImpl struct {
	db *gorm.DB
}

// NewSSHKeyRepository creates a new SSHKeyRepoImpl instance
func NewSSHKeyRepository(db *gorm.DB) repository.SSHKeyRepository {
	return &SSHKeyRepoImpl{db: db}
}

// Create creates a new SSH key in the database
func (r *SSHKeyRepoImpl) Create(ctx context.Context, key *models.SSHKey) error {
	if err := r.db.WithContext(ctx).Create(key).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("ssh key already exists", apperror.ErrSSHKeyExists)
		}
		return apperror.DatabaseError("create ssh key", err)
	}
	return nil
}

// FindByID retrieves an SSH key by its ID
func (r *SSHKeyRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.SSHKey, error) {
	var key models.SSHKey
	if err := r.db.WithContext(ctx).First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ssh key", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ssh key by id", err)
	}
	return &key, nil
}

// FindByFingerprint retrieves an SSH key by its fingerprint
func (r *SSHKeyRepoImpl) FindByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error) {
	var key models.SSHKey
	if err := r.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ssh key", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ssh key by fingerprint", err)
	}
	return &key, nil
}

// FindByUserID retrieves all SSH keys for a user
func (r *SSHKeyRepoImpl) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error) {
	var keys []*models.SSHKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, apperror.DatabaseError("find ssh keys by user id", err)
	}
	return keys, nil
}

// Update updates an existing SSH key
func (r *SSHKeyRepoImpl) Update(ctx context.Context, key *models.SSHKey) error {
	result := r.db.WithContext(ctx).Save(key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("ssh key fingerprint already exists", apperror.ErrSSHKeyExists)
		}
		return apperror.DatabaseError("update ssh key", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("ssh key", apperror.ErrNotFound)
	}
	return nil
}

// Delete removes an SSH key from the database by its ID
func (r *SSHKeyRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.SSHKey{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete ssh key", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("ssh key", apperror.ErrNotFound)
	}
	return nil
}

// DeleteByUserID removes all SSH keys for a user
func (r *SSHKeyRepoImpl) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.SSHKey{})
	if result.Error != nil {
		return apperror.DatabaseError("delete ssh keys by user id", result.Error)
	}
	return nil
}

// ExistsByFingerprint checks if an SSH key with the given fingerprint exists
func (r *SSHKeyRepoImpl) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SSHKey{}).Where("fingerprint = ?", fingerprint).Count(&count).Error; err != nil {
		return false, apperror.DatabaseError("check ssh key exists by fingerprint", err)
	}
	return count > 0, nil
}

// UpdateLastUsed updates the last_used_at timestamp for an SSH key
func (r *SSHKeyRepoImpl) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.SSHKey{}).Where("id = ?", id).Update("last_used_at", now)
	if result.Error != nil {
		return apperror.DatabaseError("update ssh key last used", result.Error)
	}
	return nil
}

// CountByUserID returns the number of SSH keys for a user
func (r *SSHKeyRepoImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SSHKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, apperror.DatabaseError("count ssh keys by user id", err)
	}
	return count, nil
}
