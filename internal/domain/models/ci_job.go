package models

import (
	"time"

	"github.com/google/uuid"
)

// CIJobStatus represents the status of a CI job
type CIJobStatus string

const (
	CIJobStatusPending   CIJobStatus = "pending"
	CIJobStatusQueued    CIJobStatus = "queued"
	CIJobStatusRunning   CIJobStatus = "running"
	CIJobStatusSuccess   CIJobStatus = "success"
	CIJobStatusFailed    CIJobStatus = "failed"
	CIJobStatusCancelled CIJobStatus = "cancelled"
	CIJobStatusTimedOut  CIJobStatus = "timed_out"
	CIJobStatusError     CIJobStatus = "error"
)

// CITriggerType represents the type of event that triggered the CI job
type CITriggerType string

const (
	CITriggerTypePush        CITriggerType = "push"
	CITriggerTypeTag         CITriggerType = "tag"
	CITriggerTypePullRequest CITriggerType = "pull_request"
	CITriggerTypeManual      CITriggerType = "manual"
)

// CIRefType represents the type of git reference
type CIRefType string

const (
	CIRefTypeBranch CIRefType = "branch"
	CIRefTypeTag    CIRefType = "tag"
)

// CIJob represents a CI/CD job in the database
type CIJob struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	RunID        uuid.UUID   `json:"run_id" gorm:"type:uuid;not null;index"` // CI runner's run ID
	RepositoryID uuid.UUID   `json:"repository_id" gorm:"type:uuid;not null;index"`
	Repository   *Repository `json:"-" gorm:"foreignKey:RepositoryID"`

	// Git reference information
	CommitSHA string    `json:"commit_sha" gorm:"size:40;not null;index"`
	RefName   string    `json:"ref_name" gorm:"size:255;not null"` // branch or tag name
	RefType   CIRefType `json:"ref_type" gorm:"size:20;not null"`

	// Trigger information
	TriggerType  CITriggerType `json:"trigger_type" gorm:"size:20;not null"`
	TriggerActor string        `json:"trigger_actor" gorm:"size:255;not null"` // username who triggered

	// Job status
	Status CIJobStatus `json:"status" gorm:"size:20;not null;default:'pending';index"`
	Error  *string     `json:"error,omitempty" gorm:"type:text"`

	// Timing
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Configuration
	ConfigPath string `json:"config_path" gorm:"size:255;default:'.stasis-ci.yaml'"`

	// Relations
	Steps     []CIJobStep  `json:"steps,omitempty" gorm:"foreignKey:JobID"`
	Logs      []CIJobLog   `json:"logs,omitempty" gorm:"foreignKey:JobID"`
	Artifacts []CIArtifact `json:"artifacts,omitempty" gorm:"foreignKey:JobID"`
}

// TableName specifies the table name for CIJob
func (CIJob) TableName() string {
	return "ci_jobs"
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

// IsFinished returns true if the job has completed (success, failed, cancelled, etc.)
func (j *CIJob) IsFinished() bool {
	switch j.Status {
	case CIJobStatusSuccess, CIJobStatusFailed, CIJobStatusCancelled, CIJobStatusTimedOut, CIJobStatusError:
		return true
	default:
		return false
	}
}

// CIStepType represents the type of a CI step
type CIStepType string

const (
	CIStepTypePre  CIStepType = "pre"
	CIStepTypeExec CIStepType = "exec"
	CIStepTypePost CIStepType = "post"
)

// CIJobStep represents a step within a CI job
type CIJobStep struct {
	ID         uuid.UUID   `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	JobID      uuid.UUID   `json:"job_id" gorm:"type:uuid;not null;index"`
	Name       string      `json:"name" gorm:"size:255;not null"`
	StepType   CIStepType  `json:"step_type" gorm:"size:20;not null"`
	Status     CIJobStatus `json:"status" gorm:"size:20;not null;default:'pending'"`
	ExitCode   *int        `json:"exit_code,omitempty"`
	StartedAt  *time.Time  `json:"started_at,omitempty"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	Order      int         `json:"order" gorm:"not null;default:0"` // Execution order
}

// TableName specifies the table name for CIJobStep
func (CIJobStep) TableName() string {
	return "ci_job_steps"
}

// Duration returns the duration of the step if it has finished
func (s *CIJobStep) Duration() *time.Duration {
	if s.StartedAt == nil {
		return nil
	}
	endTime := time.Now()
	if s.FinishedAt != nil {
		endTime = *s.FinishedAt
	}
	duration := endTime.Sub(*s.StartedAt)
	return &duration
}

// CILogLevel represents the log level
type CILogLevel string

const (
	CILogLevelDebug   CILogLevel = "debug"
	CILogLevelInfo    CILogLevel = "info"
	CILogLevelWarning CILogLevel = "warning"
	CILogLevelError   CILogLevel = "error"
)

// CIJobLog represents a log entry for a CI job
type CIJobLog struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	JobID     uuid.UUID  `json:"job_id" gorm:"type:uuid;not null;index:idx_job_sequence"`
	StepName  *string    `json:"step_name,omitempty" gorm:"size:255;index"`
	Level     CILogLevel `json:"level" gorm:"size:20;not null;default:'info'"`
	Message   string     `json:"message" gorm:"type:text;not null"`
	Sequence  uint64     `json:"sequence" gorm:"not null;index:idx_job_sequence"`
	Timestamp time.Time  `json:"timestamp" gorm:"not null;index"`
}

// TableName specifies the table name for CIJobLog
func (CIJobLog) TableName() string {
	return "ci_job_logs"
}

// CIArtifact represents an artifact produced by a CI job
type CIArtifact struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	JobID     uuid.UUID  `json:"job_id" gorm:"type:uuid;not null;index"`
	Name      string     `json:"name" gorm:"size:255;not null"`
	Path      string     `json:"path" gorm:"size:1024;not null"`
	Size      int64      `json:"size" gorm:"not null"`
	Checksum  string     `json:"checksum" gorm:"size:64;not null"` // SHA256
	URL       *string    `json:"url,omitempty" gorm:"size:2048"`   // External URL (S3, etc.)
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // When the artifact expires
}

// TableName specifies the table name for CIArtifact
func (CIArtifact) TableName() string {
	return "ci_artifacts"
}
