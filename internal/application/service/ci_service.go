package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

// CIService handles CI/CD integration with the CI runner
// All data is fetched directly from the CI server - no local database storage
type CIService struct {
	config   *config.CIConfig
	client   *resty.Client
	repoRepo repository.RepoRepository
	log      *logger.Logger

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
	Artifacts []CIRunnerArtifact `json:"artifacts,omitempty"`
}

// CIRunnerArtifact represents an artifact from the CI runner
type CIRunnerArtifact struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Checksum string  `json:"checksum"`
	URL      *string `json:"url,omitempty"`
}

// CIRunnerLogEntry represents a log entry from the CI runner
type CIRunnerLogEntry struct {
	JobID     uuid.UUID `json:"job_id"`
	RunID     uuid.UUID `json:"run_id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	StepName  *string   `json:"step_name,omitempty"`
	Message   string    `json:"message"`
	Sequence  uint64    `json:"sequence"`
}

// CIRunnerLogsResponse represents the logs response from CI runner
type CIRunnerLogsResponse struct {
	JobID string             `json:"job_id"`
	Logs  []CIRunnerLogEntry `json:"logs"`
	Total int64              `json:"total"`
}

// CIRunnerJobsListResponse represents the jobs list response from CI runner
type CIRunnerJobsListResponse struct {
	Jobs  []CIRunnerJobResponse `json:"jobs"`
	Total int64                 `json:"total"`
}

// NewCIService creates a new CI service instance
func NewCIService(
	cfg *config.CIConfig,
	repoRepo repository.RepoRepository,
) *CIService {
	client := resty.New().
		SetTimeout(cfg.Timeout()).
		SetRetryCount(3).
		SetRetryWaitTime(100 * time.Millisecond).
		SetRetryMaxWaitTime(2 * time.Second)

	// Set API key header if configured
	if cfg.APIKey != "" {
		client.SetHeader("X-API-Key", cfg.APIKey)
	}

	return &CIService{
		config:      cfg,
		client:      client,
		repoRepo:    repoRepo,
		log:         logger.Get(),
		subscribers: make(map[uuid.UUID][]chan *JobEvent),
	}
}

// IsEnabled returns true if CI integration is enabled
func (s *CIService) IsEnabled() bool {
	return s.config.IsConfigured()
}

// TriggerJob creates a new CI job and submits it to the CI runner
func (s *CIService) TriggerJob(ctx context.Context, req *TriggerJobRequest) (*CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	jobID := uuid.New()
	runID := uuid.New()

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
		JobID: jobID,
		RunID: runID,
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

	url := fmt.Sprintf("%s/api/v1/jobs", s.config.ServerURL)

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(submitReq).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to submit job: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 202 {
		return nil, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	s.log.Info("Job submitted to CI runner",
		logger.String("job_id", jobID.String()),
		logger.String("run_id", runID.String()),
	)

	// Return a minimal job response
	return &CIJob{
		ID:           jobID,
		RunID:        runID,
		RepositoryID: req.RepositoryID,
		CommitSHA:    req.CommitSHA,
		RefName:      req.RefName,
		RefType:      string(req.RefType),
		TriggerType:  string(req.TriggerType),
		TriggerActor: req.TriggerActor,
		Status:       "queued",
		ConfigPath:   s.config.GetConfigPath(),
		CreatedAt:    time.Now(),
	}, nil
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

// CIJob represents a CI job (fetched from CI server, not stored locally)
type CIJob struct {
	ID           uuid.UUID    `json:"id"`
	RunID        uuid.UUID    `json:"run_id"`
	RepositoryID uuid.UUID    `json:"repository_id"`
	CommitSHA    string       `json:"commit_sha"`
	RefName      string       `json:"ref_name"`
	RefType      string       `json:"ref_type"`
	TriggerType  string       `json:"trigger_type"`
	TriggerActor string       `json:"trigger_actor"`
	Status       string       `json:"status"`
	Error        *string      `json:"error,omitempty"`
	ConfigPath   string       `json:"config_path"`
	CreatedAt    time.Time    `json:"created_at"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	FinishedAt   *time.Time   `json:"finished_at,omitempty"`
	Steps        []CIStep     `json:"steps,omitempty"`
	Artifacts    []CIArtifact `json:"artifacts,omitempty"`
}

// CIStep represents a CI job step
type CIStep struct {
	Name         string     `json:"name"`
	StepType     string     `json:"step_type"`
	Status       string     `json:"status"`
	ExitCode     int        `json:"exit_code"`
	DurationSecs float64    `json:"duration_secs"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// CIArtifact represents a CI artifact
type CIArtifact struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Checksum string  `json:"checksum"`
	URL      *string `json:"url,omitempty"`
}

// CILog represents a CI log entry
type CILog struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	StepName  *string   `json:"step_name,omitempty"`
	Message   string    `json:"message"`
	Sequence  uint64    `json:"sequence"`
}

// Duration returns the duration of the job if it has finished
func (j *CIJob) Duration() *time.Duration {
	if j.StartedAt == nil {
		return nil
	}
	endTime := time.Now()
	if j.FinishedAt != nil {
		endTime = *j.FinishedAt
	}
	duration := endTime.Sub(*j.StartedAt)
	return &duration
}

// IsFinished returns true if the job has completed
func (j *CIJob) IsFinished() bool {
	switch j.Status {
	case "success", "failed", "cancelled", "timed_out", "error":
		return true
	default:
		return false
	}
}

// GetJob retrieves a CI job by ID from the CI server
func (s *CIService) GetJob(ctx context.Context, jobID uuid.UUID) (*CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s", s.config.ServerURL, jobID)

	var runnerResp CIRunnerJobResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&runnerResp).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch job: %w", err)
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("job not found")
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return s.mapRunnerResponseToJob(&runnerResp), nil
}

// GetJobByRunID retrieves a CI job by run ID from the CI server
func (s *CIService) GetJobByRunID(ctx context.Context, runID uuid.UUID) (*CIJob, error) {
	// CI runner uses job_id as the primary identifier
	// This method might need to query with a different endpoint if supported
	return s.GetJob(ctx, runID)
}

// ListJobsByRepository lists CI jobs for a repository from the CI server
func (s *CIService) ListJobsByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*CIJob, int64, error) {
	if !s.IsEnabled() {
		return nil, 0, fmt.Errorf("CI integration is not enabled")
	}

	// Get repository info to build the query
	repo, err := s.repoRepo.FindByID(ctx, repoID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find repository: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/jobs", s.config.ServerURL)

	var listResp CIRunnerJobsListResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"owner":  repo.Owner.Username,
			"repo":   repo.Name,
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		}).
		SetResult(&listResp).
		Get(url)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list jobs: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, 0, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	jobs := make([]*CIJob, 0, len(listResp.Jobs))
	for i := range listResp.Jobs {
		job := s.mapRunnerResponseToJob(&listResp.Jobs[i])
		job.RepositoryID = repoID
		jobs = append(jobs, job)
	}

	return jobs, listResp.Total, nil
}

// ListJobsByRef lists CI jobs for a specific ref from the CI server
func (s *CIService) ListJobsByRef(ctx context.Context, repoID uuid.UUID, refName string, limit, offset int) ([]*CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	repo, err := s.repoRepo.FindByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/jobs", s.config.ServerURL)

	var listResp CIRunnerJobsListResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"owner":    repo.Owner.Username,
			"repo":     repo.Name,
			"ref_name": refName,
			"limit":    fmt.Sprintf("%d", limit),
			"offset":   fmt.Sprintf("%d", offset),
		}).
		SetResult(&listResp).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by ref: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	jobs := make([]*CIJob, 0, len(listResp.Jobs))
	for i := range listResp.Jobs {
		job := s.mapRunnerResponseToJob(&listResp.Jobs[i])
		job.RepositoryID = repoID
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetJobLogs retrieves logs for a CI job from the CI server
func (s *CIService) GetJobLogs(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*CILog, int64, error) {
	if !s.IsEnabled() {
		return nil, 0, fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/logs", s.config.ServerURL, jobID)

	var logsResp CIRunnerLogsResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		}).
		SetResult(&logsResp).
		Get(url)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch logs: %w", err)
	}

	if resp.StatusCode() == 404 {
		return nil, 0, fmt.Errorf("job not found")
	}

	if resp.StatusCode() != 200 {
		return nil, 0, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	logs := make([]*CILog, 0, len(logsResp.Logs))
	for _, entry := range logsResp.Logs {
		logs = append(logs, &CILog{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			StepName:  entry.StepName,
			Message:   entry.Message,
			Sequence:  entry.Sequence,
		})
	}

	return logs, logsResp.Total, nil
}

// GetJobLogsAfterSequence retrieves logs after a specific sequence number
func (s *CIService) GetJobLogsAfterSequence(ctx context.Context, jobID uuid.UUID, afterSequence uint64, limit int) ([]*CILog, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/logs", s.config.ServerURL, jobID)

	var logsResp CIRunnerLogsResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"after_sequence": fmt.Sprintf("%d", afterSequence),
			"limit":          fmt.Sprintf("%d", limit),
		}).
		SetResult(&logsResp).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	logs := make([]*CILog, 0, len(logsResp.Logs))
	for _, entry := range logsResp.Logs {
		logs = append(logs, &CILog{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			StepName:  entry.StepName,
			Message:   entry.Message,
			Sequence:  entry.Sequence,
		})
	}

	return logs, nil
}

// CancelJob cancels a running CI job
func (s *CIService) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	if !s.IsEnabled() {
		return fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/cancel", s.config.ServerURL, jobID)

	resp, err := s.client.R().
		SetContext(ctx).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// RetryJob retries a failed CI job
func (s *CIService) RetryJob(ctx context.Context, jobID uuid.UUID, actor string) (*CIJob, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	// Get the original job to get its details
	originalJob, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// Get repository info
	repo, err := s.repoRepo.FindByID(ctx, originalJob.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	// Create a new trigger request based on the original job
	refType := models.CIRefTypeBranch
	if originalJob.RefType == "tag" {
		refType = models.CIRefTypeTag
	}

	req := &TriggerJobRequest{
		RepositoryID: originalJob.RepositoryID,
		Owner:        repo.Owner.Username,
		RepoName:     repo.Name,
		CloneURL:     s.buildCloneURL(repo),
		CommitSHA:    originalJob.CommitSHA,
		RefName:      originalJob.RefName,
		RefType:      refType,
		TriggerType:  models.CITriggerTypeManual,
		TriggerActor: actor,
		Metadata: map[string]string{
			"retry_of": jobID.String(),
		},
	}

	return s.TriggerJob(ctx, req)
}

// GetJobArtifacts retrieves artifacts for a CI job from the CI server
func (s *CIService) GetJobArtifacts(ctx context.Context, jobID uuid.UUID) ([]*CIArtifact, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	// Get job which includes artifacts
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	artifacts := make([]*CIArtifact, 0, len(job.Artifacts))
	for i := range job.Artifacts {
		artifacts = append(artifacts, &job.Artifacts[i])
	}

	return artifacts, nil
}

// DownloadArtifact downloads an artifact from the CI runner
func (s *CIService) DownloadArtifact(ctx context.Context, jobID uuid.UUID, artifactName string) ([]byte, string, error) {
	if !s.IsEnabled() {
		return nil, "", fmt.Errorf("CI integration is not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/artifacts/%s", s.config.ServerURL, jobID, artifactName)

	resp, err := s.client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(url)

	if err != nil {
		return nil, "", fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() == 404 {
		return nil, "", fmt.Errorf("artifact not found")
	}

	if resp.StatusCode() != 200 {
		body, _ := io.ReadAll(resp.RawBody())
		return nil, "", fmt.Errorf("CI runner returned status %d: %s", resp.StatusCode(), string(body))
	}

	data, err := io.ReadAll(resp.RawBody())
	if err != nil {
		return nil, "", fmt.Errorf("failed to read artifact data: %w", err)
	}

	contentType := resp.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}

// GetJobSteps retrieves steps for a CI job from the CI server
func (s *CIService) GetJobSteps(ctx context.Context, jobID uuid.UUID) ([]*CIStep, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("CI integration is not enabled")
	}

	// Get job which includes steps
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	steps := make([]*CIStep, 0, len(job.Steps))
	for i := range job.Steps {
		steps = append(steps, &job.Steps[i])
	}

	return steps, nil
}

// GetLatestJobByRepository gets the latest job for a repository from the CI server
func (s *CIService) GetLatestJobByRepository(ctx context.Context, repoID uuid.UUID) (*CIJob, error) {
	jobs, _, err := s.ListJobsByRepository(ctx, repoID, 1, 0)
	if err != nil {
		return nil, err
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return jobs[0], nil
}

// GetLatestJobByRef gets the latest job for a specific ref from the CI server
func (s *CIService) GetLatestJobByRef(ctx context.Context, repoID uuid.UUID, refName string) (*CIJob, error) {
	jobs, err := s.ListJobsByRef(ctx, repoID, refName, 1, 0)
	if err != nil {
		return nil, err
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return jobs[0], nil
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

// BroadcastLogEvent broadcasts a log event to all subscribers
func (s *CIService) BroadcastLogEvent(jobID uuid.UUID, log *CILog) {
	s.broadcastEvent(jobID, &JobEvent{
		Type:      "log",
		JobID:     jobID,
		Timestamp: log.Timestamp,
		Data:      s.mustMarshal(log),
	})
}

// BroadcastStatusEvent broadcasts a status update event to all subscribers
func (s *CIService) BroadcastStatusEvent(jobID uuid.UUID, status string, startedAt, finishedAt *time.Time) {
	s.broadcastEvent(jobID, &JobEvent{
		Type:      "status",
		JobID:     jobID,
		Timestamp: time.Now(),
		Data: s.mustMarshal(map[string]interface{}{
			"status":      status,
			"started_at":  startedAt,
			"finished_at": finishedAt,
		}),
	})
}

// GetConfigPath returns the path to the CI config file in repositories
func (s *CIService) GetConfigPath() string {
	return s.config.GetConfigPath()
}

// BuildCloneURL constructs the clone URL for the CI runner to use
func (s *CIService) BuildCloneURL(owner, repoName string) string {
	baseURL := s.config.GetGitServerURLWithAPIToken()
	return baseURL + "/" + owner + "/" + repoName + ".git"
}

// Helper functions

func (s *CIService) buildCloneURL(repo *models.Repository) string {
	if s.config.GitServerURL != "" {
		return fmt.Sprintf("%s/%s/%s.git", s.config.GitServerURL, repo.Owner.Username, repo.Name)
	}
	return fmt.Sprintf("http://localhost:8080/%s/%s.git", repo.Owner.Username, repo.Name)
}

func (s *CIService) mapRunnerResponseToJob(resp *CIRunnerJobResponse) *CIJob {
	job := &CIJob{
		ID:           resp.JobID,
		RunID:        resp.RunID,
		CommitSHA:    resp.Repository.CommitSHA,
		RefName:      resp.Repository.RefName,
		TriggerType:  resp.Trigger.EventType,
		TriggerActor: resp.Trigger.Actor,
		Status:       resp.Status,
		Error:        resp.Error,
	}

	// Parse timestamps
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

	// Map steps
	if resp.Result != nil {
		for _, stepData := range resp.Result.Steps {
			step := CIStep{
				Name:         stepData.Name,
				StepType:     stepData.StepType,
				ExitCode:     stepData.ExitCode,
				DurationSecs: stepData.DurationSecs,
			}
			if stepData.ExitCode == 0 {
				step.Status = "success"
			} else {
				step.Status = "failed"
			}
			if t, err := time.Parse(time.RFC3339, stepData.StartedAt); err == nil {
				step.StartedAt = &t
			}
			if t, err := time.Parse(time.RFC3339, stepData.FinishedAt); err == nil {
				step.FinishedAt = &t
			}
			job.Steps = append(job.Steps, step)
		}
	}

	// Map artifacts
	for _, artifactData := range resp.Artifacts {
		job.Artifacts = append(job.Artifacts, CIArtifact{
			Name:     artifactData.Name,
			Size:     artifactData.Size,
			Checksum: artifactData.Checksum,
			URL:      artifactData.URL,
		})
	}

	return job
}

func (s *CIService) mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}
