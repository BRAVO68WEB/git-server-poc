package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents a user in the git server system
type Token struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Token     string         `json:"-" gorm:"not null;type:text"` // Hashed token
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	Scope     pq.StringArray `json:"scope" gorm:"type:text[]"` // e.g., "owner/repo", "owner2/repo2"
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for the PAT model
func (Token) TableName() string {
	return "tokens"
}
