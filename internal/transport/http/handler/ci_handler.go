package handler

import (
	"encoding/json"
	"fmt"
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
	cloneURL := h.ciService.BuildCloneURL(owner, repoName)

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

	// Get jobs from CI server
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

	// Get repository (for validation)
	_, err = h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get job from CI server
	job, err := h.ciService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	response := h.formatJobResponse(job)
	response["steps"] = h.formatStepsResponse(job.Steps)
	response["artifacts"] = h.formatArtifactsResponse(job.Artifacts)

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

	// Get repository (for validation)
	_, err = h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 10000 {
		limit = 10000
	}

	// Get logs from CI server
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

// StreamLogs streams logs for a CI job via SSE
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

	// Get repository (for validation)
	_, err = h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
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

	ctx := c.Request.Context()
	w := c.Writer

	// Send initial job status
	job, err := h.ciService.GetJob(ctx, jobID)
	if err == nil {
		h.sendSSE(w, "status", h.formatJobResponse(job))
	}

	// Stream events
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			h.sendSSE(w, event.Type, event)
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

	// Cancel the job
	if err := h.ciService.CancelJob(c.Request.Context(), jobID); err != nil {
		h.log.Error("Failed to cancel CI job",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancelled successfully",
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

	// Retry the job
	newJob, err := h.ciService.RetryJob(c.Request.Context(), jobID, currentUser.Username)
	if err != nil {
		h.log.Error("Failed to retry CI job",
			logger.Error(err),
			logger.String("job_id", jobIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry job"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Job retry initiated",
		"original_job_id": jobID,
		"new_job_id":      newJob.ID,
		"run_id":          newJob.RunID,
		"status":          newJob.Status,
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

	// Get latest job from CI server
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

	// Get repository (for validation)
	_, err = h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get artifacts from CI server
	artifacts, err := h.ciService.GetJobArtifacts(c.Request.Context(), jobID)
	if err != nil {
		h.log.Error("Failed to get artifacts",
			logger.Error(err),
			logger.String("job_id", jobID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get artifacts"})
		return
	}

	artifactResponses := make([]gin.H, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactResponses = append(artifactResponses, gin.H{
			"name":     artifact.Name,
			"size":     artifact.Size,
			"checksum": artifact.Checksum,
			"url":      artifact.URL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifactResponses,
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

	// Get repository (for validation)
	_, err = h.repoRepo.FindByOwnerUsernameAndName(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
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

// Helper methods

func (h *CIHandler) formatJobResponse(job *service.CIJob) gin.H {
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

func (h *CIHandler) formatStepsResponse(steps []service.CIStep) []gin.H {
	result := make([]gin.H, 0, len(steps))
	for _, step := range steps {
		s := gin.H{
			"name":          step.Name,
			"step_type":     step.StepType,
			"status":        step.Status,
			"exit_code":     step.ExitCode,
			"duration_secs": step.DurationSecs,
			"started_at":    step.StartedAt,
			"finished_at":   step.FinishedAt,
		}
		result = append(result, s)
	}
	return result
}

func (h *CIHandler) formatArtifactsResponse(artifacts []service.CIArtifact) []gin.H {
	result := make([]gin.H, 0, len(artifacts))
	for _, artifact := range artifacts {
		result = append(result, gin.H{
			"name":     artifact.Name,
			"size":     artifact.Size,
			"checksum": artifact.Checksum,
			"url":      artifact.URL,
		})
	}
	return result
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

// ReceiveLogs receives logs from the CI runner (webhook endpoint)
// This is kept for backwards compatibility but logs are now fetched directly
// POST /api/v1/ci/jobs/:job_id/logs
func (h *CIHandler) ReceiveLogs(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// For backwards compatibility, we accept the logs but broadcast them via SSE
	var entries []service.CIRunnerLogEntry
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast log entries to SSE subscribers
	for _, entry := range entries {
		h.ciService.BroadcastLogEvent(jobID, &service.CILog{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			StepName:  entry.StepName,
			Message:   entry.Message,
			Sequence:  entry.Sequence,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logs received",
		"count":   len(entries),
	})
}

// CompleteJob handles job completion webhook from CI runner
// POST /api/v1/ci/jobs/:job_id/complete
func (h *CIHandler) CompleteJob(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Parse completion event
	var completion struct {
		Status     string `json:"status"`
		StartedAt  string `json:"started_at,omitempty"`
		FinishedAt string `json:"finished_at,omitempty"`
		Error      string `json:"error,omitempty"`
	}

	if err := c.ShouldBindJSON(&completion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("Received job completion event",
		logger.String("job_id", jobIDStr),
		logger.String("status", completion.Status),
	)

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

	// Broadcast completion event to SSE subscribers
	h.ciService.BroadcastStatusEvent(jobID, completion.Status, startedAt, finishedAt)

	c.JSON(http.StatusOK, gin.H{
		"message": "Completion event received",
		"job_id":  jobID,
	})
}

// WebhookJobUpdate handles generic job update webhook from CI runner
// POST /api/v1/ci/webhook
func (h *CIHandler) WebhookJobUpdate(c *gin.Context) {
	var update struct {
		JobID  uuid.UUID `json:"job_id"`
		Status string    `json:"status"`
		Event  string    `json:"event"`
	}

	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("Received CI webhook",
		logger.String("job_id", update.JobID.String()),
		logger.String("status", update.Status),
		logger.String("event", update.Event),
	)

	// Broadcast the update
	h.ciService.BroadcastStatusEvent(update.JobID, update.Status, nil, nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook received",
		"job_id":  update.JobID,
	})
}
