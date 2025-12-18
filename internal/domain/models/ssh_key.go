package models

import (
	"time"

	"github.com/google/uuid"
)

// SSHKey represents an SSH public key associated with a user
type SSHKey struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	User        User       `json:"-" gorm:"foreignKey:UserID"`
	Title       string     `json:"title" gorm:"not null;size:255"`
	PublicKey   string     `json:"-" gorm:"not null;type:text"`
	Fingerprint string     `json:"fingerprint" gorm:"uniqueIndex;not null;size:255"`
	KeyType     string     `json:"key_type" gorm:"not null;size:50"` // ssh-rsa, ssh-ed25519, ecdsa-sha2-nistp256, etc.
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for the SSHKey model
func (SSHKey) TableName() string {
	return "ssh_keys"
}

// IsExpired checks if the key has not been used for a long time (optional feature)
func (k *SSHKey) IsExpired(maxInactiveDays int) bool {
	if k.LastUsedAt == nil {
		return false
	}
	expirationTime := k.LastUsedAt.AddDate(0, 0, maxInactiveDays)
	return time.Now().After(expirationTime)
}
