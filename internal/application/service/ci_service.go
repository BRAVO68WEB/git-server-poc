package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/google/uuid"
)

// CIService handles CI/CD integration with the CI runner
type CIService struct {
	config       *config.CIConfig
	httpClient   *http.Client
	jobRepo      repository.CIJobRepository
	stepRepo     repository.CIJobStepRepository
	logRepo      repository.CIJobLogRepository
	artifactRepo repository.CIArtifactRepository
	repoRepo     repository.RepoRepository
	log          *logger.Logger

	// SSE subscribers for real-time updates
	subscribers map[uuid.UUID][]chan *JobEvent
	subMu       sync.RWMutex
}

// JobEvent represents a real-time job event for SSE streaming
type JobEvent struct {
	Type      string          `json:"type"` // "status", "log", "step", "artifact"
	JobID     uuid.UUID       `json:"job_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// SubmitJobRequest represents a request to submit a job to the CI runner
type SubmitJobRequest struct {
	JobID      uuid.UUID      `json:"job_id"`
	RunID      uuid.UUID      `json:"run_id"`
	Repository RepositoryInfo `json:"repository"`
	Trigger    TriggerInfo    `json:"trigger"`
	ConfigPath string         `json:"config_path"`
	Timestamp  time.Time      `json:"timestamp"`
	Priority   string         `json:"priority"`
	Timeout    *int           `json:"timeout,omitempty"`
}

// RepositoryInfo contains repository information for the CI runner
type RepositoryInfo struct {
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	CloneURL  string `json:"clone_url"`
	CommitSHA string `json:"commit_sha"`
	RefName   string `json:"ref_name"`
	RefType   string `json:"ref_type"` // "Branch" or "Tag"
}

// TriggerInfo contains trigger information for the CI runner
type TriggerInfo struct {
	EventType string            `json:"event_type"` // "Push", "Tag", "PullRequest", "Manual"
	Actor     string            `json:"actor"`
	Metadata  map[string]string `json:"metadata"`
}

// CIRunnerJobResponse represents the response from CI runner for job status
type CIRunnerJobResponse struct {
	JobID      uuid.UUID `json:"job_id"`
	RunID      uuid.UUID `json:"run_id"`
	Status     string    `json:"status"`
	StartedAt  *string   `json:"started_at,omitempty"`
	FinishedAt *string   `json:"finished_at,omitempty"`
	Error      *string   `json:"error,omitempty"`
	Repository struct {
		Owner     string `json:"owner"`
		Name      string `json:"name"`
		CommitSHA string `json:"commit_sha"`
		RefName   string `json:"ref_name"`
	} `json:"repository"`
	Trigger struct {
		EventType string `json:"event_type"`
		Actor     string `json:"actor"`
	} `json:"trigger"`
	Result *struct {
		Status string `json:"status"`
		Steps  []struct {
			Name         string  `json:"name"`
			StepType     string  `json:"step_type"`
			ExitCode     int     `json:"exit_code"`
			DurationSecs float64 `json:"duration_secs"`
			StartedAt    string  `json:"started_at"`
			FinishedAt   string  `json:"finished_at"`
		} `json:"steps"`
		StartedAt    string  `json:"started_at"`
		FinishedAt   string  `json:"finished_at"`
		DurationSecs float64 `json:"duration_secs"`
	} `json:"result,omitempty"`
	Artifacts []struct {
		Name     string  `json:"name"`
		Size     int64   `json:"size"`
		Checksum string  `json:"checksum"`
		URL      *string `json:"url,omitempty"`
	} `json:"artifacts,omitempty"`
}

// LogEntry represents a log entry from the CI runner
type LogEntry struct {
	JobID     uuid.UUID `json:"job_id"`
	RunID     uuid.UUID `json:"run_id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	StepName  *string   `json:"step_name,omitempty"`
	Message   string    `json:"message"`
	Sequence  uint64    `json:"sequence"`
}

// NewCIService creates a new CI service instance
func NewCIService(
	cfg *config.CIConfig,
	jobRepo repository.CIJobRepository,
	stepRepo repository.CIJobStepRepository,
	logRepo repository.CIJobLogRepository,
	artifactRepo repository.CIArtifactRepository,
	repoRepo repository.RepoRepository,
) *CIService {
	return &CIService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout(),
		},
		jobRepo:      jobRepo,
		stepRepo:     stepRepo,
		logRepo:      logRepo,
		artifactRepo: artifactRepo,
		repoRepo:     repoRepo,
		log:          logger.Get(),
		subscribers:  make(map[uuid.UUID][]chan *JobEvent),
	}
}

// IsEnabled returns true if CI integration is enabled
func (s *CIService) IsEnabled() bool {
	return s.config.IsConfigured()
}

// TriggerJob creates a new CI job and submits it to the CI runner
func (s *CIService) TriggerJob(ctx context.Context, req *TriggerJobRequest) (*models.CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	// Create the job in the database
	job := &models.CIJob{
		ID:           uuid.New(),
		RunID:        uuid.New(),
		RepositoryID: req.RepositoryID,
		CommitSHA:    req.CommitSHA,
		RefName:      req.RefName,
		RefType:      req.RefType,
		TriggerType:  req.TriggerType,
		TriggerActor: req.TriggerActor,
		Status:       models.CIJobStatusPending,
		ConfigPath:   s.config.GetConfigPath(),
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		s.log.Error("Failed to create CI job in database",
			logger.Error(err),
			logger.String("repository_id", req.RepositoryID.String()),
		)
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Submit to CI runner asynchronously
	go func() {
		submitCtx, cancel := context.WithTimeout(context.Background(), s.config.Timeout())
		defer cancel()

		if err := s.submitToCIRunner(submitCtx, job, req); err != nil {
			s.log.Error("Failed to submit job to CI runner",
				logger.Error(err),
				logger.String("job_id", job.ID.String()),
			)
			// Update job status to error
			errMsg := err.Error()
			_ = s.jobRepo.UpdateStatus(context.Background(), job.ID, models.CIJobStatusError, &errMsg)
		}
	}()

	return job, nil
}

// TriggerJobRequest contains the information needed to trigger a CI job
type TriggerJobRequest struct {
	RepositoryID uuid.UUID
	Owner        string
	RepoName     string
	CloneURL     string
	CommitSHA    string
	RefName      string
	RefType      models.CIRefType
	TriggerType  models.CITriggerType
	TriggerActor string
	Metadata     map[string]string
}

// submitToCIRunner submits a job to the CI runner service
func (s *CIService) submitToCIRunner(ctx context.Context, job *models.CIJob, req *TriggerJobRequest) error {
	// Convert ref type to CI runner format
	refType := "Branch"
	if req.RefType == models.CIRefTypeTag {
		refType = "Tag"
	}

	// Convert trigger type to CI runner format
	eventType := "Push"
	switch req.TriggerType {
	case models.CITriggerTypeTag:
		eventType = "Tag"
	case models.CITriggerTypePullRequest:
		eventType = "PullRequest"
	case models.CITriggerTypeManual:
		eventType = "Manual"
	}

	submitReq := SubmitJobRequest{
		JobID: job.ID,
		RunID: job.RunID,
		Repository: RepositoryInfo{
			Owner:     req.Owner,
			Name:      req.RepoName,
			CloneURL:  req.CloneURL,
			CommitSHA: req.CommitSHA,
			RefName:   req.RefName,
			RefType:   refType,
		},
		Trigger: TriggerInfo{
			EventType: eventType,
			Actor:     req.TriggerActor,
			Metadata:  req.Metadata,
		},
		ConfigPath: s.config.GetConfigPath(),
		Timestamp:  time.Now().UTC(),
		Priority:   "Normal",
	}

	body, err := json.Marshal(submitReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/jobs", s.config.ServerURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if s.config.APIKey != "" {
		httpReq.Header.Set("X-API-Key", s.config.APIKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Update job status to queued
	if err := s.jobRepo.UpdateStatus(ctx, job.ID, models.CIJobStatusQueued, nil); err != nil {
		s.log.Warn("Failed to update job status to queued",
			logger.Error(err),
			logger.String("job_id", job.ID.String()),
		)
	}

	s.log.Info("Job submitted to CI runner",
		logger.String("job_id", job.ID.String()),
		logger.String("run_id", job.RunID.String()),
	)

	return nil
}

// GetJob retrieves a CI job by ID
func (s *CIService) GetJob(ctx context.Context, jobID uuid.UUID) (*models.CIJob, error) {
	return s.jobRepo.FindByID(ctx, jobID)
}

// GetJobByRunID retrieves a CI job by run ID
func (s *CIService) GetJobByRunID(ctx context.Context, runID uuid.UUID) (*models.CIJob, error) {
	return s.jobRepo.FindByRunID(ctx, runID)
}

// ListJobsByRepository lists CI jobs for a repository with pagination
func (s *CIService) ListJobsByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*models.CIJob, int64, error) {
	jobs, err := s.jobRepo.FindByRepository(ctx, repoID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.jobRepo.CountByRepository(ctx, repoID)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// ListJobsByRef lists CI jobs for a specific ref with pagination
func (s *CIService) ListJobsByRef(ctx context.Context, repoID uuid.UUID, refName string, limit, offset int) ([]*models.CIJob, error) {
	return s.jobRepo.FindByRef(ctx, repoID, refName, limit, offset)
}

// GetJobLogs retrieves logs for a CI job with pagination
func (s *CIService) GetJobLogs(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*models.CIJobLog, int64, error) {
	logs, err := s.logRepo.FindByJobID(ctx, jobID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.logRepo.CountByJobID(ctx, jobID)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetJobLogsAfterSequence retrieves logs after a specific sequence number (for streaming)
func (s *CIService) GetJobLogsAfterSequence(ctx context.Context, jobID uuid.UUID, afterSequence uint64, limit int) ([]*models.CIJobLog, error) {
	return s.logRepo.FindByJobIDAfterSequence(ctx, jobID, afterSequence, limit)
}

// ReceiveLogs receives log entries from the CI runner and stores them
func (s *CIService) ReceiveLogs(ctx context.Context, jobID uuid.UUID, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Convert to database models
	logs := make([]*models.CIJobLog, 0, len(entries))
	for _, entry := range entries {
		level := models.CILogLevelInfo
		switch entry.Level {
		case "debug":
			level = models.CILogLevelDebug
		case "warning", "warn":
			level = models.CILogLevelWarning
		case "error":
			level = models.CILogLevelError
		}

		logs = append(logs, &models.CIJobLog{
			JobID:     jobID,
			StepName:  entry.StepName,
			Level:     level,
			Message:   entry.Message,
			Sequence:  entry.Sequence,
			Timestamp: entry.Timestamp,
		})
	}

	// Store in database
	if err := s.logRepo.CreateBatch(ctx, logs); err != nil {
		return fmt.Errorf("failed to store logs: %w", err)
	}

	// Broadcast to subscribers
	for _, log := range logs {
		s.broadcastEvent(jobID, &JobEvent{
			Type:      "log",
			JobID:     jobID,
			Timestamp: log.Timestamp,
			Data:      s.mustMarshal(log),
		})
	}

	return nil
}

// UpdateJobStatus updates the status of a CI job
func (s *CIService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status models.CIJobStatus, errorMsg *string) error {
	if err := s.jobRepo.UpdateStatus(ctx, jobID, status, errorMsg); err != nil {
		return err
	}

	// Broadcast status update
	s.broadcastEvent(jobID, &JobEvent{
		Type:      "status",
		JobID:     jobID,
		Timestamp: time.Now(),
		Data: s.mustMarshal(map[string]interface{}{
			"status": status,
			"error":  errorMsg,
		}),
	})

	return nil
}

// UpdateJobCompletion updates a job with completion details including timestamps
func (s *CIService) UpdateJobCompletion(ctx context.Context, jobID uuid.UUID, status models.CIJobStatus, startedAt, finishedAt *time.Time, errorMsg *string) error {
	// Get the job first
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Update fields
	job.Status = status
	if startedAt != nil {
		job.StartedAt = startedAt
	}
	if finishedAt != nil {
		job.FinishedAt = finishedAt
	}
	if errorMsg != nil {
		job.Error = errorMsg
	}

	// Save the job
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	// Broadcast status update
	s.broadcastEvent(jobID, &JobEvent{
		Type:      "status",
		JobID:     jobID,
		Timestamp: time.Now(),
		Data: s.mustMarshal(map[string]interface{}{
			"status":      status,
			"error":       errorMsg,
			"started_at":  startedAt,
			"finished_at": finishedAt,
		}),
	})

	return nil
}

// UpdateJobFromRunner syncs job state from the CI runner response
func (s *CIService) UpdateJobFromRunner(ctx context.Context, jobID uuid.UUID, resp *CIRunnerJobResponse) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Map CI runner status to local status
	status := mapCIRunnerStatus(resp.Status)
	job.Status = status

	if resp.Error != nil {
		job.Error = resp.Error
	}

	// Parse timestamps if available
	if resp.StartedAt != nil {
		if t, err := time.Parse(time.RFC3339, *resp.StartedAt); err == nil {
			job.StartedAt = &t
		}
	}
	if resp.FinishedAt != nil {
		if t, err := time.Parse(time.RFC3339, *resp.FinishedAt); err == nil {
			job.FinishedAt = &t
		}
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	// Update steps if result is available
	if resp.Result != nil {
		for i, stepData := range resp.Result.Steps {
			step := &models.CIJobStep{
				JobID:    jobID,
				Name:     stepData.Name,
				StepType: mapStepType(stepData.StepType),
				Status:   mapStepStatus(stepData.ExitCode),
				ExitCode: &stepData.ExitCode,
				Order:    i,
			}

			if t, err := time.Parse(time.RFC3339, stepData.StartedAt); err == nil {
				step.StartedAt = &t
			}
			if t, err := time.Parse(time.RFC3339, stepData.FinishedAt); err == nil {
				step.FinishedAt = &t
			}

			if err := s.stepRepo.Create(ctx, step); err != nil {
				s.log.Warn("Failed to create step record",
					logger.Error(err),
					logger.String("step_name", stepData.Name),
				)
			}
		}
	}

	// Update artifacts
	for _, artifactData := range resp.Artifacts {
		artifact := &models.CIArtifact{
			JobID:    jobID,
			Name:     artifactData.Name,
			Size:     artifactData.Size,
			Checksum: artifactData.Checksum,
			URL:      artifactData.URL,
		}

		if err := s.artifactRepo.Create(ctx, artifact); err != nil {
			s.log.Warn("Failed to create artifact record",
				logger.Error(err),
				logger.String("artifact_name", artifactData.Name),
			)
		}
	}

	// Broadcast update
	s.broadcastEvent(jobID, &JobEvent{
		Type:      "status",
		JobID:     jobID,
		Timestamp: time.Now(),
		Data:      s.mustMarshal(resp),
	})

	return nil
}

// CancelJob cancels a running CI job
func (s *CIService) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	if !s.IsEnabled() {
		return fmt.Errorf("CI integration is not enabled")
	}

	// Send cancel request to CI runner
	url := fmt.Sprintf("%s/api/v1/jobs/%s/cancel", s.config.ServerURL, jobID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if s.config.APIKey != "" {
		httpReq.Header.Set("X-API-Key", s.config.APIKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Update local status
	if err := s.jobRepo.UpdateStatus(ctx, jobID, models.CIJobStatusCancelled, nil); err != nil {
		return err
	}

	return nil
}

// RetryJob retries a failed CI job
func (s *CIService) RetryJob(ctx context.Context, jobID uuid.UUID, actor string) (*models.CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	// Get the original job
	originalJob, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// Get repository info
	repo, err := s.repoRepo.FindByID(ctx, originalJob.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	// Create a new trigger request based on the original job
	req := &TriggerJobRequest{
		RepositoryID: originalJob.RepositoryID,
		Owner:        repo.Owner.Username,
		RepoName:     repo.Name,
		CloneURL:     s.buildCloneURL(repo),
		CommitSHA:    originalJob.CommitSHA,
		RefName:      originalJob.RefName,
		RefType:      originalJob.RefType,
		TriggerType:  models.CITriggerTypeManual, // Retry is always manual
		TriggerActor: actor,
		Metadata: map[string]string{
			"retry_of": jobID.String(),
		},
	}

	return s.TriggerJob(ctx, req)
}

// Subscribe subscribes to job events for real-time updates
func (s *CIService) Subscribe(jobID uuid.UUID) chan *JobEvent {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	ch := make(chan *JobEvent, 100)
	s.subscribers[jobID] = append(s.subscribers[jobID], ch)
	return ch
}

// Unsubscribe removes a subscription
func (s *CIService) Unsubscribe(jobID uuid.UUID, ch chan *JobEvent) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	subs := s.subscribers[jobID]
	for i, sub := range subs {
		if sub == ch {
			close(ch)
			s.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(s.subscribers[jobID]) == 0 {
		delete(s.subscribers, jobID)
	}
}

// broadcastEvent sends an event to all subscribers of a job
func (s *CIService) broadcastEvent(jobID uuid.UUID, event *JobEvent) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	for _, ch := range s.subscribers[jobID] {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this event
			s.log.Warn("Dropping event, subscriber channel full",
				logger.String("job_id", jobID.String()),
			)
		}
	}
}

// SyncJobStatus fetches the latest status from CI runner and updates local state
func (s *CIService) SyncJobStatus(ctx context.Context, jobID uuid.UUID) (*models.CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s", s.config.ServerURL, jobID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if s.config.APIKey != "" {
		httpReq.Header.Set("X-API-Key", s.config.APIKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("job not found on CI runner")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var runnerResp CIRunnerJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&runnerResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update local state
	if err := s.UpdateJobFromRunner(ctx, jobID, &runnerResp); err != nil {
		return nil, err
	}

	return s.jobRepo.FindByID(ctx, jobID)
}

// GetLatestJobByRepository gets the latest job for a repository
func (s *CIService) GetLatestJobByRepository(ctx context.Context, repoID uuid.UUID) (*models.CIJob, error) {
	return s.jobRepo.GetLatestByRepository(ctx, repoID)
}

// GetLatestJobByRef gets the latest job for a specific ref
func (s *CIService) GetLatestJobByRef(ctx context.Context, repoID uuid.UUID, refName string) (*models.CIJob, error) {
	return s.jobRepo.GetLatestByRef(ctx, repoID, refName)
}

// CleanupOldJobs removes old job records based on retention policy
func (s *CIService) CleanupOldJobs(ctx context.Context) (int64, error) {
	deleted, err := s.jobRepo.DeleteOlderThan(ctx, s.config.RetentionDays)
	if err != nil {
		return 0, err
	}

	s.log.Info("Cleaned up old CI jobs",
		logger.Int64("deleted_count", deleted),
		logger.Int("retention_days", s.config.RetentionDays),
	)

	return deleted, nil
}

// GetJobArtifacts retrieves artifacts for a CI job
func (s *CIService) GetJobArtifacts(ctx context.Context, jobID uuid.UUID) ([]*models.CIArtifact, error) {
	return s.artifactRepo.FindByJobID(ctx, jobID)
}

// SaveArtifact saves an artifact record to the database
func (s *CIService) SaveArtifact(ctx context.Context, artifact *models.CIArtifact) error {
	return s.artifactRepo.Create(ctx, artifact)
}

// DownloadArtifact downloads an artifact from the CI runner
func (s *CIService) DownloadArtifact(ctx context.Context, jobID uuid.UUID, artifactName string) ([]byte, string, error) {
	if !s.IsEnabled() {
		return nil, "", fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/artifacts/%s", s.config.ServerURL, jobID, artifactName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	if s.config.APIKey != "" {
		httpReq.Header.Set("X-API-Key", s.config.APIKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("artifact not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read artifact data: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}

// GetJobSteps retrieves steps for a CI job
func (s *CIService) GetJobSteps(ctx context.Context, jobID uuid.UUID) ([]*models.CIJobStep, error) {
	return s.stepRepo.FindByJobID(ctx, jobID)
}

// GetConfigPath returns the path to the CI config file in repositories
func (s *CIService) GetConfigPath() string {
	return s.config.GetConfigPath()
}

// BuildCloneURL constructs the clone URL for the CI runner to use
func (s *CIService) BuildCloneURL(owner, repoName string) string {
	baseURL := s.config.GetGitServerURL()
	if baseURL == "" {
		// If git_server_url is not configured, use the CI server URL as fallback
		// The caller should ideally configure git_server_url properly
		baseURL = s.config.ServerURL
	}
	return fmt.Sprintf("%s/%s/%s.git", baseURL, owner, repoName)
}

// Helper functions

func (s *CIService) buildCloneURL(repo *models.Repository) string {
	if s.config.GitServerURL != "" {
		return fmt.Sprintf("%s/%s/%s.git", s.config.GitServerURL, repo.Owner.Username, repo.Name)
	}
	// Fall back to using the clone URL from the repository
	return fmt.Sprintf("http://localhost:8080/%s/%s.git", repo.Owner.Username, repo.Name)
}

func (s *CIService) mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

func mapCIRunnerStatus(status string) models.CIJobStatus {
	switch status {
	case "pending":
		return models.CIJobStatusPending
	case "queued":
		return models.CIJobStatusQueued
	case "running":
		return models.CIJobStatusRunning
	case "completed", "success":
		return models.CIJobStatusSuccess
	case "failed":
		return models.CIJobStatusFailed
	case "cancelled":
		return models.CIJobStatusCancelled
	case "timed_out", "timedout", "timeout":
		return models.CIJobStatusTimedOut
	default:
		return models.CIJobStatusError
	}
}

func mapStepType(stepType string) models.CIStepType {
	switch stepType {
	case "pre":
		return models.CIStepTypePre
	case "post":
		return models.CIStepTypePost
	default:
		return models.CIStepTypeExec
	}
}

func mapStepStatus(exitCode int) models.CIJobStatus {
	if exitCode == 0 {
		return models.CIJobStatusSuccess
	}
	return models.CIJobStatusFailed
}
