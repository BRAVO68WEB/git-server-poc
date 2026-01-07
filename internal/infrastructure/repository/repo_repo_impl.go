package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	apperror "github.com/bravo68web/stasis/pkg/errors"
	"github.com/google/uuid"
)

// RepoRepoImpl implements the RepoRepository interface using GORM
type RepoRepoImpl struct {
	db *gorm.DB
}

// NewRepoRepository creates a new instance of RepoRepoImpl
func NewRepoRepository(db *gorm.DB) repository.RepoRepository {
	return &RepoRepoImpl{db: db}
}

// Create creates a new repository in the database
func (r *RepoRepoImpl) Create(ctx context.Context, repo *models.Repository) error {
	if err := r.db.WithContext(ctx).Create(repo).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("repository already exists", apperror.ErrRepositoryExists)
		}
		return apperror.DatabaseError("create", err)
	}
	return nil
}

// FindByID retrieves a repository by its ID
func (r *RepoRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).Preload("Owner").First(&repo, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("repository", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find", err)
	}
	return &repo, nil
}

// FindByOwnerAndName finds a repository by owner ID and name
func (r *RepoRepoImpl) FindByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("owner_id = ? AND name = ?", ownerID, name).
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("repository", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find", err)
	}
	return &repo, nil
}

// FindByOwnerUsernameAndName finds a repository by owner username and repository name
func (r *RepoRepoImpl) FindByOwnerUsernameAndName(ctx context.Context, username, name string) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Joins("JOIN users ON users.id = repositories.owner_id").
		Where("users.username = ? AND repositories.name = ?", username, name).
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("repository", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find", err)
	}
	return &repo, nil
}

// FindByOwner finds all repositories owned by a user
func (r *RepoRepoImpl) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Repository, error) {
	var repos []*models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&repos).Error
	if err != nil {
		return nil, apperror.DatabaseError("find", err)
	}
	return repos, nil
}

// ListPublic lists public repositories with pagination
func (r *RepoRepoImpl) ListPublic(ctx context.Context, limit, offset int) ([]*models.Repository, error) {
	var repos []*models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("is_private = ?", false).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, apperror.DatabaseError("list", err)
	}
	return repos, nil
}

// ListAll lists all repositories with pagination (for admin use)
func (r *RepoRepoImpl) ListAll(ctx context.Context, limit, offset int) ([]*models.Repository, error) {
	var repos []*models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, apperror.DatabaseError("list", err)
	}
	return repos, nil
}

// Update updates a repository
func (r *RepoRepoImpl) Update(ctx context.Context, repo *models.Repository) error {
	result := r.db.WithContext(ctx).Save(repo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("repository name already exists", apperror.ErrRepositoryExists)
		}
		return apperror.DatabaseError("update", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("repository", apperror.ErrNotFound)
	}
	return nil
}

// Delete deletes a repository by ID
func (r *RepoRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Repository{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("repository", apperror.ErrNotFound)
	}
	return nil
}

// ExistsByOwnerAndName checks if a repository exists with the given owner and name
func (r *RepoRepoImpl) ExistsByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("owner_id = ? AND name = ?", ownerID, name).
		Count(&count).Error
	if err != nil {
		return false, apperror.DatabaseError("count", err)
	}
	return count > 0, nil
}

// CountByOwner returns the count of repositories owned by a user
func (r *RepoRepoImpl) CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("owner_id = ?", ownerID).
		Count(&count).Error
	if err != nil {
		return 0, apperror.DatabaseError("count", err)
	}
	return count, nil
}

// FindByGitPath finds a repository by its git path
func (r *RepoRepoImpl) FindByGitPath(ctx context.Context, gitPath string) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("git_path = ?", gitPath).
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("repository", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find", err)
	}
	return &repo, nil
}

// Search searches repositories by name or description
func (r *RepoRepoImpl) Search(ctx context.Context, query string, limit, offset int, includePrivate bool) ([]*models.Repository, error) {
	var repos []*models.Repository
	searchPattern := fmt.Sprintf("%%%s%%", query)

	db := r.db.WithContext(ctx).
		Preload("Owner").
		Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)

	if !includePrivate {
		db = db.Where("is_private = ?", false)
	}

	err := db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, apperror.DatabaseError("search", err)
	}
	return repos, nil
}

// UpdateVisibility updates the visibility of a repository
func (r *RepoRepoImpl) UpdateVisibility(ctx context.Context, id uuid.UUID, isPrivate bool) error {
	result := r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("id = ?", id).
		Update("is_private", isPrivate)
	if result.Error != nil {
		return apperror.DatabaseError("update", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("repository", apperror.ErrNotFound)
	}
	return nil
}

// UpdateDescription updates the description of a repository
func (r *RepoRepoImpl) UpdateDescription(ctx context.Context, id uuid.UUID, description string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("id = ?", id).
		Update("description", description)
	if result.Error != nil {
		return apperror.DatabaseError("update", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("repository", apperror.ErrNotFound)
	}
	return nil
}

// FindAllMirrors finds all mirror repositories
func (r *RepoRepoImpl) FindAllMirrors(ctx context.Context) ([]*models.Repository, error) {
	var repos []*models.Repository
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("mirror_enabled = ?", true).
		Order("created_at DESC").
		Find(&repos).Error
	if err != nil {
		return nil, apperror.DatabaseError("find", err)
	}
	return repos, nil
}
