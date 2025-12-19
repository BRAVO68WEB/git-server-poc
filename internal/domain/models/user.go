package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the git server system
type User struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username    string    `json:"username" gorm:"uniqueIndex;not null;size:255"`
	Email       string    `json:"email" gorm:"uniqueIndex;not null;size:255"`
	OIDCSubject string    `json:"-" gorm:"uniqueIndex:idx_oidc_subject_issuer;size:255"` // OIDC subject (sub claim)
	OIDCIssuer  string    `json:"-" gorm:"uniqueIndex:idx_oidc_subject_issuer;size:255"` // OIDC issuer URL
	IsAdmin     bool      `json:"is_admin" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for the User model
func (User) TableName() string {
	return "users"
}
