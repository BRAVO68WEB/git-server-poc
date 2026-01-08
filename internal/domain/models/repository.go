package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
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

	// Mirror configuration
	MirrorEnabled      bool       `json:"mirror_enabled" gorm:"default:false"`         // Enable/disable mirror sync
	MirrorDirection    string     `json:"mirror_direction,omitempty"`                  // "upstream", "downstream", "both"
	UpstreamURL        string     `json:"upstream_url,omitempty"`                      // URL to pull from (upstream)
	UpstreamUsername   string     `json:"upstream_username,omitempty"`                 // Auth for upstream
	UpstreamPassword   string     `json:"upstream_password,omitempty"`                 // Auth for upstream (encrypted)
	DownstreamURL      string     `json:"downstream_url,omitempty"`                    // URL to push to (downstream)
	DownstreamUsername string     `json:"downstream_username,omitempty"`               // Auth for downstream
	DownstreamPassword string     `json:"downstream_password,omitempty"`               // Auth for downstream (encrypted)
	SyncInterval       int        `json:"sync_interval" gorm:"default:3600"`           // Sync interval in seconds (default: 1 hour) - deprecated, use SyncSchedule
	SyncSchedule       string     `json:"sync_schedule,omitempty"`                     // Cron expression for sync schedule (e.g., "0 */1 * * *" for hourly)
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`                    // Last successful sync time
	SyncStatus         string     `json:"sync_status,omitempty" gorm:"default:'idle'"` // "idle", "syncing", "success", "failed"
	SyncError          string     `json:"sync_error,omitempty"`                        // Last sync error message

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// CanSync returns true if the repository can be synced
func (r *Repository) CanSync() bool {
	return r.MirrorEnabled && (r.HasUpstream() || r.HasDownstream())
}

// HasUpstream returns true if repository has upstream mirror configured
func (r *Repository) HasUpstream() bool {
	return (r.MirrorDirection == "upstream" || r.MirrorDirection == "both") && r.UpstreamURL != ""
}

// HasDownstream returns true if repository has downstream mirror configured
func (r *Repository) HasDownstream() bool {
	return (r.MirrorDirection == "downstream" || r.MirrorDirection == "both") && r.DownstreamURL != ""
}

// IsSyncing returns true if the repository is currently syncing
func (r *Repository) IsSyncing() bool {
	return r.SyncStatus == "syncing"
}

// GetSyncIntervalDuration returns the sync interval as a time.Duration
func (r *Repository) GetSyncIntervalDuration() time.Duration {
	if r.SyncInterval <= 0 {
		return time.Hour // Default to 1 hour
	}
	return time.Duration(r.SyncInterval) * time.Second
}

// HasCronSchedule returns true if repository has a cron schedule configured
func (r *Repository) HasCronSchedule() bool {
	return r.SyncSchedule != ""
}

// GetNextSyncTime returns the next scheduled sync time based on cron schedule or interval
func (r *Repository) GetNextSyncTime() *time.Time {
	if !r.MirrorEnabled {
		return nil
	}

	// Use cron schedule if available
	if r.HasCronSchedule() {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(r.SyncSchedule)
		if err == nil {
			// Use last sync time as reference, or now if never synced
			var referenceTime time.Time
			if r.LastSyncedAt != nil {
				referenceTime = *r.LastSyncedAt
			} else {
				referenceTime = time.Now()
			}
			nextTime := schedule.Next(referenceTime)
			return &nextTime
		}
		// Fall through to interval-based calculation if cron parse fails
	}

	// Fall back to interval-based calculation
	if r.SyncInterval > 0 {
		var nextTime time.Time
		if r.LastSyncedAt != nil {
			nextTime = r.LastSyncedAt.Add(r.GetSyncIntervalDuration())
		} else {
			// If never synced, next sync is now + interval
			nextTime = time.Now().Add(r.GetSyncIntervalDuration())
		}
		return &nextTime
	}

	return nil
}
