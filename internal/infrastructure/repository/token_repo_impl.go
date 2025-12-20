package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	apperror "github.com/bravo68web/stasis/pkg/errors"
	"github.com/google/uuid"
)

// TokenRepoImpl implements the TokenRepository interface using GORM
type TokenRepoImpl struct {
	db *gorm.DB
}

// NewTokenRepository creates a new TokenRepoImpl instance
func NewTokenRepository(db *gorm.DB) repository.TokenRepository {
	return &TokenRepoImpl{db: db}
}

// Create creates a new token in the database
func (r *TokenRepoImpl) Create(ctx context.Context, token *models.Token) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return apperror.DatabaseError("create token", err)
	}
	return nil
}

// FindByID retrieves a token by its ID
func (r *TokenRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.Token, error) {
	var token models.Token
	if err := r.db.WithContext(ctx).First(&token, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("token", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find token by id", err)
	}
	return &token, nil
}

// FindByHashedToken retrieves a token by its hashed value
func (r *TokenRepoImpl) FindByHashedToken(ctx context.Context, hashedToken string) (*models.Token, error) {
	var token models.Token
	if err := r.db.WithContext(ctx).Where("token = ?", hashedToken).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("token", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find token by hash", err)
	}
	return &token, nil
}

// FindByUserID retrieves all tokens for a user
func (r *TokenRepoImpl) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Token, error) {
	var tokens []*models.Token
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, apperror.DatabaseError("find tokens by user id", err)
	}
	return tokens, nil
}

// Delete removes a token from the database by its ID
func (r *TokenRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Token{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete token", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("token", apperror.ErrNotFound)
	}
	return nil
}

// DeleteByUserID removes all tokens for a user
func (r *TokenRepoImpl) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.Token{})
	if result.Error != nil {
		return apperror.DatabaseError("delete tokens by user id", result.Error)
	}
	return nil
}

// UpdateLastUsed updates the last_used timestamp for a token
func (r *TokenRepoImpl) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Token{}).Where("id = ?", id).Update("last_used", now)
	if result.Error != nil {
		return apperror.DatabaseError("update token last used", result.Error)
	}
	return nil
}

// CountByUserID returns the number of tokens for a user
func (r *TokenRepoImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Token{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, apperror.DatabaseError("count tokens by user id", err)
	}
	return count, nil
}

// Verify interface compliance at compile time
var _ repository.TokenRepository = (*TokenRepoImpl)(nil)
