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

// CIJobRepoImpl implements the CIJobRepository interface using GORM
type CIJobRepoImpl struct {
	db *gorm.DB
}

// NewCIJobRepository creates a new instance of CIJobRepoImpl
func NewCIJobRepository(db *gorm.DB) repository.CIJobRepository {
	return &CIJobRepoImpl{db: db}
}

// Create creates a new CI job in the database
func (r *CIJobRepoImpl) Create(ctx context.Context, job *models.CIJob) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return apperror.DatabaseError("create ci job", err)
	}
	return nil
}

// FindByID finds a CI job by its ID
func (r *CIJobRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.CIJob, error) {
	var job models.CIJob
	err := r.db.WithContext(ctx).
		Preload("Repository").
		Preload("Artifacts").
		First(&job, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ci job", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ci job", err)
	}
	return &job, nil
}

// FindByRunID finds a CI job by its run ID
func (r *CIJobRepoImpl) FindByRunID(ctx context.Context, runID uuid.UUID) (*models.CIJob, error) {
	var job models.CIJob
	err := r.db.WithContext(ctx).
		Preload("Repository").
		Preload("Artifacts").
		Where("run_id = ?", runID).
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ci job", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ci job by run id", err)
	}
	return &job, nil
}

// FindByRepository finds all CI jobs for a repository with pagination
func (r *CIJobRepoImpl) FindByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci jobs by repository", err)
	}
	return jobs, nil
}

// FindByRepositoryAndStatus finds CI jobs by repository and status
func (r *CIJobRepoImpl) FindByRepositoryAndStatus(ctx context.Context, repoID uuid.UUID, status models.CIJobStatus, limit, offset int) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND status = ?", repoID, status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci jobs by repository and status", err)
	}
	return jobs, nil
}

// FindByCommit finds CI jobs for a specific commit
func (r *CIJobRepoImpl) FindByCommit(ctx context.Context, repoID uuid.UUID, commitSHA string) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND commit_sha = ?", repoID, commitSHA).
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci jobs by commit", err)
	}
	return jobs, nil
}

// FindByRef finds CI jobs for a specific ref
func (r *CIJobRepoImpl) FindByRef(ctx context.Context, repoID uuid.UUID, refName string, limit, offset int) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND ref_name = ?", repoID, refName).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci jobs by ref", err)
	}
	return jobs, nil
}

// Update updates a CI job
func (r *CIJobRepoImpl) Update(ctx context.Context, job *models.CIJob) error {
	if err := r.db.WithContext(ctx).Save(job).Error; err != nil {
		return apperror.DatabaseError("update ci job", err)
	}
	return nil
}

// UpdateStatus updates the status of a CI job
func (r *CIJobRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status models.CIJobStatus, errorMsg *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != nil {
		updates["error"] = *errorMsg
	}

	// Set timestamps based on status
	now := time.Now()
	switch status {
	case models.CIJobStatusRunning:
		updates["started_at"] = now
	case models.CIJobStatusSuccess, models.CIJobStatusFailed, models.CIJobStatusCancelled, models.CIJobStatusTimedOut, models.CIJobStatusError:
		updates["finished_at"] = now
	}

	result := r.db.WithContext(ctx).Model(&models.CIJob{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return apperror.DatabaseError("update ci job status", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("ci job", apperror.ErrNotFound)
	}
	return nil
}

// Delete deletes a CI job by ID
func (r *CIJobRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CIJob{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete ci job", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("ci job", apperror.ErrNotFound)
	}
	return nil
}

// CountByRepository returns the count of CI jobs for a repository
func (r *CIJobRepoImpl) CountByRepository(ctx context.Context, repoID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CIJob{}).Where("repository_id = ?", repoID).Count(&count).Error
	if err != nil {
		return 0, apperror.DatabaseError("count ci jobs", err)
	}
	return count, nil
}

// CountByRepositoryAndStatus returns the count of CI jobs by repository and status
func (r *CIJobRepoImpl) CountByRepositoryAndStatus(ctx context.Context, repoID uuid.UUID, status models.CIJobStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CIJob{}).Where("repository_id = ? AND status = ?", repoID, status).Count(&count).Error
	if err != nil {
		return 0, apperror.DatabaseError("count ci jobs by status", err)
	}
	return count, nil
}

// FindPendingJobs finds all pending jobs
func (r *CIJobRepoImpl) FindPendingJobs(ctx context.Context, limit int) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("status = ?", models.CIJobStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find pending ci jobs", err)
	}
	return jobs, nil
}

// FindRunningJobs finds all running jobs
func (r *CIJobRepoImpl) FindRunningJobs(ctx context.Context, limit int) ([]*models.CIJob, error) {
	var jobs []*models.CIJob
	err := r.db.WithContext(ctx).
		Where("status = ?", models.CIJobStatusRunning).
		Order("started_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, apperror.DatabaseError("find running ci jobs", err)
	}
	return jobs, nil
}

// DeleteOlderThan deletes jobs older than the specified duration
func (r *CIJobRepoImpl) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).
		Where("created_at < ? AND status IN ?", cutoff, []models.CIJobStatus{
			models.CIJobStatusSuccess,
			models.CIJobStatusFailed,
			models.CIJobStatusCancelled,
			models.CIJobStatusTimedOut,
			models.CIJobStatusError,
		}).
		Delete(&models.CIJob{})
	if result.Error != nil {
		return 0, apperror.DatabaseError("delete old ci jobs", result.Error)
	}
	return result.RowsAffected, nil
}

// GetLatestByRepository gets the latest job for a repository
func (r *CIJobRepoImpl) GetLatestByRepository(ctx context.Context, repoID uuid.UUID) (*models.CIJob, error) {
	var job models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No job found is not an error
		}
		return nil, apperror.DatabaseError("get latest ci job", err)
	}
	return &job, nil
}

// GetLatestByRef gets the latest job for a specific ref
func (r *CIJobRepoImpl) GetLatestByRef(ctx context.Context, repoID uuid.UUID, refName string) (*models.CIJob, error) {
	var job models.CIJob
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND ref_name = ?", repoID, refName).
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No job found is not an error
		}
		return nil, apperror.DatabaseError("get latest ci job by ref", err)
	}
	return &job, nil
}

// ============================================================
// CIJobStepRepoImpl implements CIJobStepRepository
// ============================================================

type CIJobStepRepoImpl struct {
	db *gorm.DB
}

// NewCIJobStepRepository creates a new instance of CIJobStepRepoImpl
func NewCIJobStepRepository(db *gorm.DB) repository.CIJobStepRepository {
	return &CIJobStepRepoImpl{db: db}
}

func (r *CIJobStepRepoImpl) Create(ctx context.Context, step *models.CIJobStep) error {
	if err := r.db.WithContext(ctx).Create(step).Error; err != nil {
		return apperror.DatabaseError("create ci job step", err)
	}
	return nil
}

func (r *CIJobStepRepoImpl) CreateBatch(ctx context.Context, steps []*models.CIJobStep) error {
	if len(steps) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&steps).Error; err != nil {
		return apperror.DatabaseError("create ci job steps batch", err)
	}
	return nil
}

func (r *CIJobStepRepoImpl) FindByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.CIJobStep, error) {
	var steps []*models.CIJobStep
	err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("\"order\" ASC").
		Find(&steps).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci job steps", err)
	}
	return steps, nil
}

func (r *CIJobStepRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.CIJobStep, error) {
	var step models.CIJobStep
	err := r.db.WithContext(ctx).First(&step, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ci job step", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ci job step", err)
	}
	return &step, nil
}

func (r *CIJobStepRepoImpl) Update(ctx context.Context, step *models.CIJobStep) error {
	if err := r.db.WithContext(ctx).Save(step).Error; err != nil {
		return apperror.DatabaseError("update ci job step", err)
	}
	return nil
}

func (r *CIJobStepRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status models.CIJobStatus, exitCode *int) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if exitCode != nil {
		updates["exit_code"] = *exitCode
	}

	now := time.Now()
	switch status {
	case models.CIJobStatusRunning:
		updates["started_at"] = now
	case models.CIJobStatusSuccess, models.CIJobStatusFailed, models.CIJobStatusCancelled, models.CIJobStatusTimedOut, models.CIJobStatusError:
		updates["finished_at"] = now
	}

	result := r.db.WithContext(ctx).Model(&models.CIJobStep{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return apperror.DatabaseError("update ci job step status", result.Error)
	}
	return nil
}

func (r *CIJobStepRepoImpl) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Delete(&models.CIJobStep{}).Error; err != nil {
		return apperror.DatabaseError("delete ci job steps", err)
	}
	return nil
}

// ============================================================
// CIJobLogRepoImpl implements CIJobLogRepository
// ============================================================

type CIJobLogRepoImpl struct {
	db *gorm.DB
}

// NewCIJobLogRepository creates a new instance of CIJobLogRepoImpl
func NewCIJobLogRepository(db *gorm.DB) repository.CIJobLogRepository {
	return &CIJobLogRepoImpl{db: db}
}

func (r *CIJobLogRepoImpl) Create(ctx context.Context, log *models.CIJobLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return apperror.DatabaseError("create ci job log", err)
	}
	return nil
}

func (r *CIJobLogRepoImpl) CreateBatch(ctx context.Context, logs []*models.CIJobLog) error {
	if len(logs) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&logs).Error; err != nil {
		return apperror.DatabaseError("create ci job logs batch", err)
	}
	return nil
}

func (r *CIJobLogRepoImpl) FindByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*models.CIJobLog, error) {
	var logs []*models.CIJobLog
	query := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("sequence ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, apperror.DatabaseError("find ci job logs", err)
	}
	return logs, nil
}

func (r *CIJobLogRepoImpl) FindByJobIDAndStep(ctx context.Context, jobID uuid.UUID, stepName string, limit, offset int) ([]*models.CIJobLog, error) {
	var logs []*models.CIJobLog
	query := r.db.WithContext(ctx).
		Where("job_id = ? AND step_name = ?", jobID, stepName).
		Order("sequence ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, apperror.DatabaseError("find ci job logs by step", err)
	}
	return logs, nil
}

func (r *CIJobLogRepoImpl) FindByJobIDAfterSequence(ctx context.Context, jobID uuid.UUID, afterSequence uint64, limit int) ([]*models.CIJobLog, error) {
	var logs []*models.CIJobLog
	query := r.db.WithContext(ctx).
		Where("job_id = ? AND sequence > ?", jobID, afterSequence).
		Order("sequence ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, apperror.DatabaseError("find ci job logs after sequence", err)
	}
	return logs, nil
}

func (r *CIJobLogRepoImpl) CountByJobID(ctx context.Context, jobID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CIJobLog{}).Where("job_id = ?", jobID).Count(&count).Error
	if err != nil {
		return 0, apperror.DatabaseError("count ci job logs", err)
	}
	return count, nil
}

func (r *CIJobLogRepoImpl) GetLatestSequence(ctx context.Context, jobID uuid.UUID) (uint64, error) {
	var maxSeq struct {
		MaxSequence uint64
	}
	err := r.db.WithContext(ctx).
		Model(&models.CIJobLog{}).
		Select("COALESCE(MAX(sequence), 0) as max_sequence").
		Where("job_id = ?", jobID).
		Scan(&maxSeq).Error
	if err != nil {
		return 0, apperror.DatabaseError("get latest sequence", err)
	}
	return maxSeq.MaxSequence, nil
}

func (r *CIJobLogRepoImpl) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Delete(&models.CIJobLog{}).Error; err != nil {
		return apperror.DatabaseError("delete ci job logs", err)
	}
	return nil
}

func (r *CIJobLogRepoImpl) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).
		Where("timestamp < ?", cutoff).
		Delete(&models.CIJobLog{})
	if result.Error != nil {
		return 0, apperror.DatabaseError("delete old ci job logs", result.Error)
	}
	return result.RowsAffected, nil
}

// ============================================================
// CIArtifactRepoImpl implements CIArtifactRepository
// ============================================================

type CIArtifactRepoImpl struct {
	db *gorm.DB
}

// NewCIArtifactRepository creates a new instance of CIArtifactRepoImpl
func NewCIArtifactRepository(db *gorm.DB) repository.CIArtifactRepository {
	return &CIArtifactRepoImpl{db: db}
}

func (r *CIArtifactRepoImpl) Create(ctx context.Context, artifact *models.CIArtifact) error {
	if err := r.db.WithContext(ctx).Create(artifact).Error; err != nil {
		return apperror.DatabaseError("create ci artifact", err)
	}
	return nil
}

func (r *CIArtifactRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.CIArtifact, error) {
	var artifact models.CIArtifact
	err := r.db.WithContext(ctx).First(&artifact, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ci artifact", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ci artifact", err)
	}
	return &artifact, nil
}

func (r *CIArtifactRepoImpl) FindByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.CIArtifact, error) {
	var artifacts []*models.CIArtifact
	err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("created_at ASC").
		Find(&artifacts).Error
	if err != nil {
		return nil, apperror.DatabaseError("find ci artifacts", err)
	}
	return artifacts, nil
}

func (r *CIArtifactRepoImpl) FindByJobIDAndName(ctx context.Context, jobID uuid.UUID, name string) (*models.CIArtifact, error) {
	var artifact models.CIArtifact
	err := r.db.WithContext(ctx).
		Where("job_id = ? AND name = ?", jobID, name).
		First(&artifact).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("ci artifact", apperror.ErrNotFound)
		}
		return nil, apperror.DatabaseError("find ci artifact by name", err)
	}
	return &artifact, nil
}

func (r *CIArtifactRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CIArtifact{}, id)
	if result.Error != nil {
		return apperror.DatabaseError("delete ci artifact", result.Error)
	}
	return nil
}

func (r *CIArtifactRepoImpl) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Delete(&models.CIArtifact{}).Error; err != nil {
		return apperror.DatabaseError("delete ci artifacts", err)
	}
	return nil
}

func (r *CIArtifactRepoImpl) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&models.CIArtifact{})
	if result.Error != nil {
		return 0, apperror.DatabaseError("delete expired ci artifacts", result.Error)
	}
	return result.RowsAffected, nil
}
