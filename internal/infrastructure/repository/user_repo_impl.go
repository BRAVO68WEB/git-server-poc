package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	apperror "github.com/bravo68web/githut/pkg/errors"
	"github.com/google/uuid"
)

// UserRepoImpl implements the UserRepository interface using GORM
type UserRepoImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepoImpl instance
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &UserRepoImpl{db: db}
}

// Create creates a new user in the database
func (r *UserRepoImpl) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("user already exists", apperror.ErrUserExists)
		}
		return apperror.DatabaseError("create user", err)
	}
	return nil
}

// FindByID retrieves a user by their ID
func (r *UserRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find user by id", err)
	}
	return &user, nil
}

// FindByUsername retrieves a user by their username
func (r *UserRepoImpl) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find user by username", err)
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email address
func (r *UserRepoImpl) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find user by email", err)
	}
	return &user, nil
}

// Update updates an existing user's information
func (r *UserRepoImpl) Update(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return apperror.Conflict("username or email already exists", apperror.ErrUserExists)
		}
		return apperror.DatabaseError("update user", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("user", apperror.ErrNotFound)
	}
	return nil
}

// Delete removes a user from the database by their ID
func (r *UserRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete user", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("user", apperror.ErrNotFound)
	}
	return nil
}

// List retrieves all users with pagination
func (r *UserRepoImpl) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := r.db.WithContext(ctx).Order("id ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, apperror.DatabaseError("list users", err)
	}
	return users, nil
}

// Count returns the total number of users
func (r *UserRepoImpl) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, apperror.DatabaseError("count users", err)
	}
	return count, nil
}

// ExistsByUsername checks if a user with the given username exists
func (r *UserRepoImpl) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, apperror.DatabaseError("check user exists by username", err)
	}
	return count > 0, nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepoImpl) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, apperror.DatabaseError("check user exists by email", err)
	}
	return count > 0, nil
}
