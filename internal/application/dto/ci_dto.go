package dto

import (
	"time"

	"github.com/google/uuid"
)

// CIJobResponse represents a CI job in API responses
type CIJobResponse struct {
	ID              uuid.UUID            `json:"id"`
	RunID           string               `json:"run_id"`
	RepositoryID    uuid.UUID            `json:"repository_id"`
	CommitSHA       string               `json:"commit_sha"`
	RefName         string               `json:"ref_name"`
	RefType         string               `json:"ref_type"`
	TriggerType     string               `json:"trigger_type"`
	TriggerActor    string               `json:"trigger_actor"`
	Status          string               `json:"status"`
	ConfigPath      string               `json:"config_path"`
	CreatedAt       time.Time            `json:"created_at"`
	StartedAt       *time.Time           `json:"started_at,omitempty"`
	FinishedAt      *time.Time           `json:"finished_at,omitempty"`
	Error           string               `json:"error,omitempty"`
	DurationSeconds float64              `json:"duration_seconds,omitempty"`
	Steps           []CIStepResponse     `json:"steps,omitempty"`
	Artifacts       []CIArtifactResponse `json:"artifacts,omitempty"`
}

// CIStepResponse represents a step in a CI job
type CIStepResponse struct {
	Name         string     `json:"name"`
	StepType     string     `json:"step_type"`
	Status       string     `json:"status"`
	ExitCode     int        `json:"exit_code"`
	DurationSecs float64    `json:"duration_secs"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// CIArtifactResponse represents a build artifact
type CIArtifactResponse struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	URL      string `json:"url"`
}

// CIJobListResponse represents a list of CI jobs
type CIJobListResponse struct {
	Jobs       []CIJobResponse `json:"jobs"`
	Total      int64           `json:"total"`
	Pagination Pagination      `json:"pagination"`
}

// Pagination represents pagination info
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// CILogEntryResponse represents a single log line
type CILogEntryResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	StepName  string    `json:"step_name"`
	Message   string    `json:"message"`
	Sequence  int       `json:"sequence"`
}

// CILogsResponse represents the logs for a job
type CILogsResponse struct {
	JobID      uuid.UUID            `json:"job_id"`
	Logs       []CILogEntryResponse `json:"logs"`
	Total      int64                `json:"total"`
	Pagination Pagination           `json:"pagination"`
}

// CIArtifactListResponse represents a list of artifacts
type CIArtifactListResponse struct {
	Artifacts []CIArtifactResponse `json:"artifacts"`
	Total     int                  `json:"total"`
}

// CIJobTriggerResponse represents the response after triggering a job
type CIJobTriggerResponse struct {
	Message string    `json:"message"`
	JobID   uuid.UUID `json:"job_id"`
	RunID   string    `json:"run_id"`
	Status  string    `json:"status"`
}

// CIRunnerLogEntryRequest represents a log entry from the runner
type CIRunnerLogEntryRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	StepName  string    `json:"step_name"`
	Message   string    `json:"message"`
	Sequence  int       `json:"sequence"`
}

// CIJobCompleteRequest represents the job completion payload from the runner
type CIJobCompleteRequest struct {
	Status     string `json:"status"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

// CIWebhookJobUpdateRequest represents a generic job update webhook
type CIWebhookJobUpdateRequest struct {
	JobID  uuid.UUID `json:"job_id"`
	Status string    `json:"status"`
	Event  string    `json:"event"`
}
