package repository

import (
	"context"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/google/uuid"
)

// CIJobRepository defines the interface for CI job data access
type CIJobRepository interface {
	// Create creates a new CI job
	Create(ctx context.Context, job *models.CIJob) error

	// FindByID finds a CI job by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.CIJob, error)

	// FindByRunID finds a CI job by its run ID (from CI runner)
	FindByRunID(ctx context.Context, runID uuid.UUID) (*models.CIJob, error)

	// FindByRepository finds all CI jobs for a repository with pagination
	FindByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*models.CIJob, error)

	// FindByRepositoryAndStatus finds CI jobs by repository and status
	FindByRepositoryAndStatus(ctx context.Context, repoID uuid.UUID, status models.CIJobStatus, limit, offset int) ([]*models.CIJob, error)

	// FindByCommit finds CI jobs for a specific commit
	FindByCommit(ctx context.Context, repoID uuid.UUID, commitSHA string) ([]*models.CIJob, error)

	// FindByRef finds CI jobs for a specific ref (branch/tag)
	FindByRef(ctx context.Context, repoID uuid.UUID, refName string, limit, offset int) ([]*models.CIJob, error)

	// Update updates a CI job
	Update(ctx context.Context, job *models.CIJob) error

	// UpdateStatus updates the status of a CI job
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.CIJobStatus, errorMsg *string) error

	// Delete deletes a CI job by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByRepository returns the count of CI jobs for a repository
	CountByRepository(ctx context.Context, repoID uuid.UUID) (int64, error)

	// CountByRepositoryAndStatus returns the count of CI jobs by repository and status
	CountByRepositoryAndStatus(ctx context.Context, repoID uuid.UUID, status models.CIJobStatus) (int64, error)

	// FindPendingJobs finds all pending jobs (for cleanup/recovery)
	FindPendingJobs(ctx context.Context, limit int) ([]*models.CIJob, error)

	// FindRunningJobs finds all running jobs
	FindRunningJobs(ctx context.Context, limit int) ([]*models.CIJob, error)

	// DeleteOlderThan deletes jobs older than the specified duration
	DeleteOlderThan(ctx context.Context, days int) (int64, error)

	// GetLatestByRepository gets the latest job for a repository
	GetLatestByRepository(ctx context.Context, repoID uuid.UUID) (*models.CIJob, error)

	// GetLatestByRef gets the latest job for a specific ref
	GetLatestByRef(ctx context.Context, repoID uuid.UUID, refName string) (*models.CIJob, error)
}

// CIJobStepRepository defines the interface for CI job step data access
type CIJobStepRepository interface {
	// Create creates a new CI job step
	Create(ctx context.Context, step *models.CIJobStep) error

	// CreateBatch creates multiple steps at once
	CreateBatch(ctx context.Context, steps []*models.CIJobStep) error

	// FindByJobID finds all steps for a job
	FindByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.CIJobStep, error)

	// FindByID finds a step by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.CIJobStep, error)

	// Update updates a step
	Update(ctx context.Context, step *models.CIJobStep) error

	// UpdateStatus updates the status of a step
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.CIJobStatus, exitCode *int) error

	// DeleteByJobID deletes all steps for a job
	DeleteByJobID(ctx context.Context, jobID uuid.UUID) error
}

// CIJobLogRepository defines the interface for CI job log data access
type CIJobLogRepository interface {
	// Create creates a new log entry
	Create(ctx context.Context, log *models.CIJobLog) error

	// CreateBatch creates multiple log entries at once
	CreateBatch(ctx context.Context, logs []*models.CIJobLog) error

	// FindByJobID finds all logs for a job with pagination
	FindByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*models.CIJobLog, error)

	// FindByJobIDAndStep finds logs for a specific job and step
	FindByJobIDAndStep(ctx context.Context, jobID uuid.UUID, stepName string, limit, offset int) ([]*models.CIJobLog, error)

	// FindByJobIDAfterSequence finds logs after a specific sequence number (for streaming)
	FindByJobIDAfterSequence(ctx context.Context, jobID uuid.UUID, afterSequence uint64, limit int) ([]*models.CIJobLog, error)

	// CountByJobID returns the count of logs for a job
	CountByJobID(ctx context.Context, jobID uuid.UUID) (int64, error)

	// GetLatestSequence gets the latest sequence number for a job
	GetLatestSequence(ctx context.Context, jobID uuid.UUID) (uint64, error)

	// DeleteByJobID deletes all logs for a job
	DeleteByJobID(ctx context.Context, jobID uuid.UUID) error

	// DeleteOlderThan deletes logs older than the specified duration
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}

// CIArtifactRepository defines the interface for CI artifact data access
type CIArtifactRepository interface {
	// Create creates a new artifact record
	Create(ctx context.Context, artifact *models.CIArtifact) error

	// FindByID finds an artifact by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.CIArtifact, error)

	// FindByJobID finds all artifacts for a job
	FindByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.CIArtifact, error)

	// FindByJobIDAndName finds an artifact by job ID and name
	FindByJobIDAndName(ctx context.Context, jobID uuid.UUID, name string) (*models.CIArtifact, error)

	// Delete deletes an artifact by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByJobID deletes all artifacts for a job
	DeleteByJobID(ctx context.Context, jobID uuid.UUID) error

	// DeleteExpired deletes expired artifacts
	DeleteExpired(ctx context.Context) (int64, error)
}
