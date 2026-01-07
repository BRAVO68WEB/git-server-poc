package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bravo68web/stasis/internal/domain/models"
	domainrepo "github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// MirrorSyncService handles synchronization of mirror repositories
type MirrorSyncService struct {
	repoRepo   domainrepo.RepoRepository
	gitService service.GitService
	log        *logger.Logger
}

// NewMirrorSyncService creates a new mirror sync service
func NewMirrorSyncService(
	repoRepo domainrepo.RepoRepository,
	gitService service.GitService,
) *MirrorSyncService {
	return &MirrorSyncService{
		repoRepo:   repoRepo,
		gitService: gitService,
		log:        logger.Get(),
	}
}

// SyncRepository syncs a single mirror repository
func (s *MirrorSyncService) SyncRepository(ctx context.Context, repoID uuid.UUID) error {
	s.log.Info("Starting mirror sync",
		logger.String("repo_id", repoID.String()),
	)

	// Get repository
	repo, err := s.repoRepo.FindByID(ctx, repoID)
	if err != nil {
		s.log.Error("Failed to find repository",
			logger.Error(err),
			logger.String("repo_id", repoID.String()),
		)
		return fmt.Errorf("failed to find repository: %w", err)
	}

	// Verify it's a mirror repository
	if !repo.CanSync() {
		s.log.Warn("Repository is not a mirror",
			logger.String("repo_id", repoID.String()),
			logger.String("repo_name", repo.Name),
		)
		return fmt.Errorf("repository is not a mirror")
	}

	// Check if already syncing
	if repo.IsSyncing() {
		s.log.Warn("Repository is already syncing",
			logger.String("repo_id", repoID.String()),
			logger.String("repo_name", repo.Name),
		)
		return fmt.Errorf("repository is already syncing")
	}

	// Update status to syncing
	repo.SyncStatus = "syncing"
	repo.SyncError = ""
	if err := s.repoRepo.Update(ctx, repo); err != nil {
		s.log.Error("Failed to update sync status",
			logger.Error(err),
			logger.String("repo_id", repoID.String()),
		)
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Perform sync in background
	go s.performSync(context.Background(), repo)

	return nil
}

// performSync performs the actual sync operation
func (s *MirrorSyncService) performSync(ctx context.Context, repo *models.Repository) {
	s.log.Info("Performing mirror sync",
		logger.String("repo_id", repo.ID.String()),
		logger.String("repo_name", repo.Name),
		logger.String("direction", repo.MirrorDirection),
	)

	var syncErr error

	// Perform upstream sync (pull from external source)
	if repo.HasUpstream() {
		s.log.Debug("Syncing upstream",
			logger.String("repo_id", repo.ID.String()),
			logger.String("upstream_url", repo.UpstreamURL),
		)
		if err := s.gitService.FetchMirror(ctx, repo.GitPath, repo.UpstreamURL); err != nil {
			s.log.Error("Upstream sync failed",
				logger.Error(err),
				logger.String("repo_id", repo.ID.String()),
			)
			syncErr = fmt.Errorf("upstream sync failed: %w", err)
		} else {
			s.log.Info("Upstream sync completed",
				logger.String("repo_id", repo.ID.String()),
			)
		}
	}

	// Perform downstream sync (push to external destination)
	if repo.HasDownstream() && syncErr == nil {
		s.log.Debug("Syncing downstream",
			logger.String("repo_id", repo.ID.String()),
			logger.String("downstream_url", repo.DownstreamURL),
		)
		if err := s.gitService.PushMirror(ctx, repo.GitPath, repo.DownstreamURL, repo.DownstreamUsername, repo.DownstreamPassword); err != nil {
			s.log.Error("Downstream sync failed",
				logger.Error(err),
				logger.String("repo_id", repo.ID.String()),
			)
			if syncErr != nil {
				syncErr = fmt.Errorf("%v; downstream sync failed: %w", syncErr, err)
			} else {
				syncErr = fmt.Errorf("downstream sync failed: %w", err)
			}
		} else {
			s.log.Info("Downstream sync completed",
				logger.String("repo_id", repo.ID.String()),
			)
		}
	}

	// Update repository status
	now := time.Now()
	repo.LastSyncedAt = &now

	if syncErr != nil {
		s.log.Error("Mirror sync failed",
			logger.Error(syncErr),
			logger.String("repo_id", repo.ID.String()),
			logger.String("repo_name", repo.Name),
		)
		repo.SyncStatus = "failed"
		repo.SyncError = syncErr.Error()
	} else {
		s.log.Info("Mirror sync completed successfully",
			logger.String("repo_id", repo.ID.String()),
			logger.String("repo_name", repo.Name),
		)
		repo.SyncStatus = "success"
		repo.SyncError = ""
	}

	// Save updated status
	if updateErr := s.repoRepo.Update(ctx, repo); updateErr != nil {
		s.log.Error("Failed to update repository after sync",
			logger.Error(updateErr),
			logger.String("repo_id", repo.ID.String()),
		)
	}
}

// SyncAllMirrors syncs all mirror repositories that are due for sync
func (s *MirrorSyncService) SyncAllMirrors(ctx context.Context) error {
	s.log.Debug("Checking mirror repositories for sync")

	// Get all mirror repositories
	mirrors, err := s.repoRepo.FindAllMirrors(ctx)
	if err != nil {
		s.log.Error("Failed to find mirror repositories",
			logger.Error(err),
		)
		return fmt.Errorf("failed to find mirror repositories: %w", err)
	}

	if len(mirrors) == 0 {
		s.log.Debug("No mirror repositories found")
		return nil
	}

	now := time.Now()
	syncCount := 0
	errorCount := 0
	skippedCount := 0

	for _, repo := range mirrors {
		// Skip if not enabled
		if !repo.MirrorEnabled {
			continue
		}

		// Skip if already syncing
		if repo.IsSyncing() {
			skippedCount++
			continue
		}

		// Check if sync is due based on cron schedule or interval
		if !s.isSyncDue(repo, now) {
			// Not yet time to sync
			continue
		}

		// Trigger sync
		if err := s.SyncRepository(ctx, repo.ID); err != nil {
			s.log.Error("Failed to sync mirror",
				logger.Error(err),
				logger.String("repo_id", repo.ID.String()),
				logger.String("repo_name", repo.Name),
			)
			errorCount++
		} else {
			syncCount++
		}
	}

	if syncCount > 0 || errorCount > 0 {
		s.log.Info("Mirror sync check completed",
			logger.Int("total_mirrors", len(mirrors)),
			logger.Int("synced", syncCount),
			logger.Int("skipped", skippedCount),
			logger.Int("errors", errorCount),
		)
	}

	return nil
}

// GetSyncStatus returns the sync status of a repository
func (s *MirrorSyncService) GetSyncStatus(ctx context.Context, repoID uuid.UUID) (*SyncStatus, error) {
	repo, err := s.repoRepo.FindByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if !repo.MirrorEnabled {
		return nil, fmt.Errorf("repository mirror is not enabled")
	}

	return &SyncStatus{
		MirrorEnabled:   repo.MirrorEnabled,
		MirrorDirection: repo.MirrorDirection,
		UpstreamURL:     repo.UpstreamURL,
		DownstreamURL:   repo.DownstreamURL,
		SyncInterval:    repo.SyncInterval,
		SyncSchedule:    repo.SyncSchedule,
		LastSyncedAt:    repo.LastSyncedAt,
		NextSyncAt:      repo.GetNextSyncTime(),
		SyncStatus:      repo.SyncStatus,
		SyncError:       repo.SyncError,
	}, nil
}

// SyncStatus represents the sync status of a mirror repository
type SyncStatus struct {
	MirrorEnabled   bool       `json:"mirror_enabled"`
	MirrorDirection string     `json:"mirror_direction"`
	UpstreamURL     string     `json:"upstream_url,omitempty"`
	DownstreamURL   string     `json:"downstream_url,omitempty"`
	SyncInterval    int        `json:"sync_interval"`
	SyncSchedule    string     `json:"sync_schedule,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at"`
	NextSyncAt      *time.Time `json:"next_sync_at,omitempty"`
	SyncStatus      string     `json:"sync_status"`
	SyncError       string     `json:"sync_error,omitempty"`
}

// isSyncDue checks if a repository is due for sync based on cron schedule or interval
func (s *MirrorSyncService) isSyncDue(repo *models.Repository, now time.Time) bool {
	// If never synced, it's due
	if repo.LastSyncedAt == nil {
		return true
	}

	// Check cron schedule first
	if repo.HasCronSchedule() {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(repo.SyncSchedule)
		if err == nil {
			nextSync := schedule.Next(*repo.LastSyncedAt)
			return now.After(nextSync) || now.Equal(nextSync)
		}
		// Fall through to interval check if cron parse fails
		s.log.Warn("Failed to parse cron schedule, falling back to interval",
			logger.Error(err),
			logger.String("repo_id", repo.ID.String()),
			logger.String("schedule", repo.SyncSchedule),
		)
	}

	// Fall back to interval-based check
	if repo.SyncInterval > 0 {
		nextSync := repo.LastSyncedAt.Add(repo.GetSyncIntervalDuration())
		return now.After(nextSync) || now.Equal(nextSync)
	}

	// If no schedule or interval, sync immediately
	return true
}
