package models

import (
	"time"

	"github.com/google/uuid"
)

// Repository represents a Git repository in the system
type Repository struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name          string    `json:"name" gorm:"not null" `
	OwnerID       uuid.UUID `json:"owner_id" gorm:"not null" `
	Owner         User      `json:"owner,omitzero" gorm:"foreignKey:OwnerID" `
	IsPrivate     bool      `json:"is_private" gorm:"default:false" `
	Description   string    `json:"description"`
	DefaultBranch string    `json:"default_branch" gorm:"default:'main'" `
	GitPath       string    `json:"git_path" gorm:"uniqueIndex;not null" ` // Storage path
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName specifies the table name for Repository
func (Repository) TableName() string {
	return "repositories"
}

// IsPublic returns true if the repository is public
func (r *Repository) IsPublic() bool {
	return !r.IsPrivate
}

// GetFullName returns the full repository name in format owner/repo
func (r *Repository) GetFullName() string {
	if r.Owner.Username != "" {
		return r.Owner.Username + "/" + r.Name
	}
	return r.Name
}
