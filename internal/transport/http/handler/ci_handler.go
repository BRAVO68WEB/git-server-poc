package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CIHandler handles CI-related HTTP requests
type CIHandler struct {
	ciService *service.CIService
	repoRepo  repository.RepoRepository
	log       *logger.Logger
}

// NewCIHandler creates a new CI handler
func NewCIHandler(ciService *service.CIService, repoRepo repository.RepoRepository) *CIHandler {
	return &CIHandler{
		ciService: ciService,
		repoRepo:  repoRepo,
		log:       logger.Get(),
	}
}

// TriggerJobRequest represents the request body for triggering a CI job
type TriggerJobRequest struct {
	CommitSHA string `json:"commit_sha" binding:"required"`
	RefName   string `json:"ref_name" binding:"required"`
	RefType   string `json:"ref_type" binding:"required,oneof=branch tag"` // "branch" or "tag"
}

// TriggerJob triggers a new CI job for a repository
// POST /api/v1/repos/:owner/:repo/ci/jobs
func (h *CIHandler) TriggerJob(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Get authenticated user
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUser := user.(*models.User)

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Check permissions (owner or collaborator)
	if repo.OwnerID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req TriggerJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map ref type
	refType := models.CIRefTypeBranch
	if req.RefType == "tag" {
		refType = models.CIRefTypeTag
	}

	// Build clone URL
	cloneURL := fmt.Sprintf("%s/%s/%s.git", c.Request.Host, owner, repoName)

	// Trigger the job
	job, err := h.ciService.TriggerJob(c.Request.Context(), &service.TriggerJobRequest{
		RepositoryID: repo.ID,
		Owner:        owner,
		RepoName:     repoName,
		CloneURL:     cloneURL,
		CommitSHA:    req.CommitSHA,
		RefName:      req.RefName,
		RefType:      refType,
		TriggerType:  models.CITriggerTypeManual,
		TriggerActor: currentUser.Username,
		Metadata:     map[string]string{},
	})
	if err != nil {
		h.log.Error("Failed to trigger CI job",
			logger.Error(err),
			logger.String("owner", owner),
			logger.String("repo", repoName),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger job"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Job triggered successfully",
		"job_id":  job.ID,
		"run_id":  job.RunID,
		"status":  job.Status,
	})
}

// ListJobs lists CI jobs for a repository
// GET /api/v1/repos/:owner/:repo/ci/jobs
func (h *CIHandler) ListJobs(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	// Get jobs
	jobs, total, err := h.ciService.ListJobsByRepository(c.Request.Context(), repo.ID, limit, offset)
	if err != nil {
		h.log.Error("Failed to list CI jobs",
			logger.Error(err),
			logger.String("owner", owner),
			logger.String("repo", repoName),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	// Build response
	jobResponses := make([]gin.H, 0, len(jobs))
	for _, job := range jobs {
		jobResponses = append(jobResponses, h.formatJobResponse(job))
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobResponses,
		"total": total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetJob gets a specific CI job
// GET /api/v1/repos/:owner/:repo/ci/jobs/:job_id
func (h *CIHandler) GetJob(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get job
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to this repository
	if job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Get steps
	steps, _ := h.ciService.GetJobSteps(c.Request.Context(), jobID)

	// Get artifacts
	artifacts, _ := h.ciService.GetJobArtifacts(c.Request.Context(), jobID)

	response := h.formatJobResponse(job)
	response["steps"] = h.formatStepsResponse(steps)
	response["artifacts"] = h.formatArtifactsResponse(artifacts)

	c.JSON(http.StatusOK, response)
}

// GetJobLogs gets logs for a CI job
// GET /api/v1/repos/:owner/:repo/ci/jobs/:job_id/logs
func (h *CIHandler) GetJobLogs(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Verify job belongs to this repository
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil || job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 10000 {
		limit = 10000
	}

	// Get logs
	logs, total, err := h.ciService.GetJobLogs(c.Request.Context(), jobID, limit, offset)
	if err != nil {
		h.log.Error("Failed to get job logs",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get logs"})
		return
	}

	// Format logs
	logResponses := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		logResponses = append(logResponses, gin.H{
			"timestamp": log.Timestamp,
			"level":     log.Level,
			"step_name": log.StepName,
			"message":   log.Message,
			"sequence":  log.Sequence,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"logs":   logResponses,
		"total":  total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ReceiveLogs receives logs from the CI runner
// POST /api/v1/ci/jobs/:job_id/logs
func (h *CIHandler) ReceiveLogs(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Parse log entries
	var entries []service.LogEntry
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store logs
	if err := h.ciService.ReceiveLogs(c.Request.Context(), jobID, entries); err != nil {
		h.log.Error("Failed to receive logs",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logs received",
		"count":   len(entries),
	})
}

// StreamLogs streams job logs via Server-Sent Events
// GET /api/v1/repos/:owner/:repo/ci/jobs/:job_id/stream
func (h *CIHandler) StreamLogs(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Verify job belongs to this repository
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil || job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe to job events
	eventCh := h.ciService.Subscribe(jobID)
	defer h.ciService.Unsubscribe(jobID, eventCh)

	// Send initial connection message
	h.sendSSE(c.Writer, "connected", gin.H{
		"job_id":  jobID,
		"status":  job.Status,
		"message": "Connected to job stream",
	})

	// Send existing logs
	logs, _, _ := h.ciService.GetJobLogs(c.Request.Context(), jobID, 1000, 0)
	for _, log := range logs {
		h.sendSSE(c.Writer, "log", gin.H{
			"timestamp": log.Timestamp,
			"level":     log.Level,
			"step_name": log.StepName,
			"message":   log.Message,
			"sequence":  log.Sequence,
		})
	}

	// Create ticker for heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Stream events
	clientGone := c.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			h.sendSSE(c.Writer, event.Type, event.Data)
		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

// CancelJob cancels a running CI job
// POST /api/v1/repos/:owner/:repo/ci/jobs/:job_id/cancel
func (h *CIHandler) CancelJob(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get authenticated user
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUser := user.(*models.User)

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Check permissions
	if repo.OwnerID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Verify job belongs to this repository
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil || job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Cancel the job
	if err := h.ciService.CancelJob(c.Request.Context(), jobID); err != nil {
		h.log.Error("Failed to cancel job",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancelled",
		"job_id":  jobID,
	})
}

// RetryJob retries a failed CI job
// POST /api/v1/repos/:owner/:repo/ci/jobs/:job_id/retry
func (h *CIHandler) RetryJob(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get authenticated user
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUser := user.(*models.User)

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Check permissions
	if repo.OwnerID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Verify job belongs to this repository
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil || job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Retry the job
	newJob, err := h.ciService.RetryJob(c.Request.Context(), jobID, currentUser.Username)
	if err != nil {
		h.log.Error("Failed to retry job",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry job"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Job retry triggered",
		"new_job_id":      newJob.ID,
		"new_run_id":      newJob.RunID,
		"original_job_id": jobID,
	})
}

// CompleteJob receives job completion events from the CI runner
// POST /api/v1/ci/jobs/:job_id/complete
func (h *CIHandler) CompleteJob(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Read raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	h.log.Info("Received job completion event",
		logger.String("job_id", jobID.String()),
	)

	// RustDuration matches Rust's std::time::Duration JSON format
	type RustDuration struct {
		Secs  uint64 `json:"secs"`
		Nanos uint32 `json:"nanos"`
	}

	// Parse the completion event
	var completion struct {
		JobID      uuid.UUID    `json:"job_id"`
		RunID      uuid.UUID    `json:"run_id"`
		Status     string       `json:"status"`
		StartedAt  string       `json:"started_at,omitempty"`
		FinishedAt string       `json:"finished_at,omitempty"`
		Duration   RustDuration `json:"duration,omitempty"`
		ExitCode   int          `json:"exit_code"`
		Steps      []struct {
			Name     string       `json:"name"`
			Status   string       `json:"status"`
			ExitCode int          `json:"exit_code"`
			Duration RustDuration `json:"duration,omitempty"`
		} `json:"steps,omitempty"`
		Artifacts []struct {
			Name     string  `json:"name"`
			Size     int64   `json:"size"`
			Checksum string  `json:"checksum"`
			URL      *string `json:"url,omitempty"`
		} `json:"artifacts,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(body, &completion); err != nil {
		h.log.Error("Failed to parse completion event",
			logger.Error(err),
			logger.String("job_id", jobID.String()),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Map status string to models.CIJobStatus
	var status models.CIJobStatus
	switch completion.Status {
	case "Success", "success":
		status = models.CIJobStatusSuccess
	case "Failed", "failed":
		status = models.CIJobStatusFailed
	case "Cancelled", "cancelled":
		status = models.CIJobStatusCancelled
	case "TimedOut", "timed_out":
		status = models.CIJobStatusTimedOut
	default:
		status = models.CIJobStatusError
	}

	// Parse timestamps
	var startedAt, finishedAt *time.Time
	if completion.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, completion.StartedAt); err == nil {
			startedAt = &t
		}
	}
	if completion.FinishedAt != "" {
		if t, err := time.Parse(time.RFC3339, completion.FinishedAt); err == nil {
			finishedAt = &t
		}
	}

	// Update job with completion details including timestamps
	var errorMsg *string
	if status == models.CIJobStatusFailed || status == models.CIJobStatusError {
		msg := fmt.Sprintf("Job finished with status: %s, exit code: %d", completion.Status, completion.ExitCode)
		errorMsg = &msg
	}

	if err := h.ciService.UpdateJobCompletion(c.Request.Context(), jobID, status, startedAt, finishedAt, errorMsg); err != nil {
		h.log.Error("Failed to update job completion",
			logger.Error(err),
			logger.String("job_id", jobID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job"})
		return
	}

	// Save artifacts from completion event
	if len(completion.Artifacts) > 0 {
		for _, artifactData := range completion.Artifacts {
			artifact := &models.CIArtifact{
				JobID:    jobID,
				Name:     artifactData.Name,
				Path:     artifactData.Name, // Use name as path since CI runner doesn't always provide path
				Size:     artifactData.Size,
				Checksum: artifactData.Checksum,
				URL:      artifactData.URL,
			}
			if err := h.ciService.SaveArtifact(c.Request.Context(), artifact); err != nil {
				h.log.Warn("Failed to save artifact from completion",
					logger.Error(err),
					logger.String("job_id", jobID.String()),
					logger.String("artifact", artifactData.Name),
				)
			} else {
				h.log.Info("Artifact saved",
					logger.String("job_id", jobID.String()),
					logger.String("artifact", artifactData.Name),
				)
			}
		}
	}

	h.log.Info("Job completion processed",
		logger.String("job_id", jobID.String()),
		logger.String("status", string(status)),
		logger.Int("artifacts", len(completion.Artifacts)),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Job completion received",
		"job_id":  jobID,
		"status":  status,
	})
}

// WebhookJobUpdate receives job status updates from the CI runner
// POST /api/v1/ci/webhook/job-update
func (h *CIHandler) WebhookJobUpdate(c *gin.Context) {
	// Read raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Parse the update
	var update service.CIRunnerJobResponse
	if err := json.Unmarshal(body, &update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Update job status
	if err := h.ciService.UpdateJobFromRunner(c.Request.Context(), update.JobID, &update); err != nil {
		h.log.Error("Failed to update job from webhook",
			logger.Error(err),
			logger.String("job_id", update.JobID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job updated",
		"job_id":  update.JobID,
	})
}

// GetLatestJob gets the latest CI job for a repository
// GET /api/v1/repos/:owner/:repo/ci/latest
func (h *CIHandler) GetLatestJob(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get latest job
	job, err := h.ciService.GetLatestJobByRepository(c.Request.Context(), repo.ID)
	if err != nil {
		h.log.Error("Failed to get latest job",
			logger.Error(err),
			logger.String("owner", owner),
			logger.String("repo", repoName),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get latest job"})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no jobs found"})
		return
	}

	c.JSON(http.StatusOK, h.formatJobResponse(job))
}

// Helper methods

func (h *CIHandler) formatJobResponse(job *models.CIJob) gin.H {
	response := gin.H{
		"id":            job.ID,
		"run_id":        job.RunID,
		"repository_id": job.RepositoryID,
		"commit_sha":    job.CommitSHA,
		"ref_name":      job.RefName,
		"ref_type":      job.RefType,
		"trigger_type":  job.TriggerType,
		"trigger_actor": job.TriggerActor,
		"status":        job.Status,
		"config_path":   job.ConfigPath,
		"created_at":    job.CreatedAt,
		"started_at":    job.StartedAt,
		"finished_at":   job.FinishedAt,
		"error":         job.Error,
	}

	if duration := job.Duration(); duration != nil {
		response["duration_seconds"] = duration.Seconds()
	}

	return response
}

func (h *CIHandler) formatStepsResponse(steps []*models.CIJobStep) []gin.H {
	result := make([]gin.H, 0, len(steps))
	for _, step := range steps {
		s := gin.H{
			"id":          step.ID,
			"name":        step.Name,
			"step_type":   step.StepType,
			"status":      step.Status,
			"exit_code":   step.ExitCode,
			"order":       step.Order,
			"started_at":  step.StartedAt,
			"finished_at": step.FinishedAt,
		}
		if duration := step.Duration(); duration != nil {
			s["duration_seconds"] = duration.Seconds()
		}
		result = append(result, s)
	}
	return result
}

func (h *CIHandler) formatArtifactsResponse(artifacts []*models.CIArtifact) []gin.H {
	result := make([]gin.H, 0, len(artifacts))
	for _, artifact := range artifacts {
		result = append(result, gin.H{
			"id":         artifact.ID,
			"name":       artifact.Name,
			"path":       artifact.Path,
			"size":       artifact.Size,
			"checksum":   artifact.Checksum,
			"url":        artifact.URL,
			"created_at": artifact.CreatedAt,
			"expires_at": artifact.ExpiresAt,
		})
	}
	return result
}

// ListArtifacts lists all artifacts for a CI job
// GET /api/v1/repos/:owner/:repo/ci/jobs/:job_id/artifacts
func (h *CIHandler) ListArtifacts(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get job
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to this repository
	if job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Get artifacts
	artifacts, err := h.ciService.GetJobArtifacts(c.Request.Context(), jobID)
	if err != nil {
		h.log.Error("Failed to get artifacts",
			logger.Error(err),
			logger.String("job_id", jobID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get artifacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": h.formatArtifactsResponse(artifacts),
		"total":     len(artifacts),
	})
}

// DownloadArtifact proxies artifact download from CI runner
// GET /api/v1/repos/:owner/:repo/ci/jobs/:job_id/artifacts/:artifact_name
func (h *CIHandler) DownloadArtifact(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	jobIDStr := c.Param("job_id")
	artifactName := c.Param("artifact_name")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Get repository
	repo, err := h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get job
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to this repository
	if job.RepositoryID != repo.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Download artifact from CI runner
	data, contentType, err := h.ciService.DownloadArtifact(c.Request.Context(), jobID, artifactName)
	if err != nil {
		h.log.Error("Failed to download artifact",
			logger.Error(err),
			logger.String("job_id", jobID.String()),
			logger.String("artifact", artifactName),
		)
		if err.Error() == "artifact not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download artifact"})
		return
	}

	// Set response headers
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", artifactName))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}

func (h *CIHandler) sendSSE(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
