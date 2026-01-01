package router

import (
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
)

// ciRouter sets up CI-related routes
func (r *Router) ciRouter() {
	// Use the shared CI service from dependencies
	ciService := r.Deps.CIService

	// Initialize CI handler
	ciHandler := handler.NewCIHandler(ciService, r.Deps.RepoService.GetRepoRepository())

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// ========================================
	// Repository-scoped CI routes
	// ========================================
	// These routes are scoped to a specific repository
	repoGroup := r.server.Group("/api/v1/repos/:owner/:repo/ci")
	{
		// Public routes (can view job status) - optional auth to handle private repos
		repoGroup.GET("/latest", authMiddleware.Authenticate(), ciHandler.GetLatestJob)
		repoGroup.GET("/jobs", authMiddleware.Authenticate(), ciHandler.ListJobs)
		repoGroup.GET("/jobs/:job_id", authMiddleware.Authenticate(), ciHandler.GetJob)
		repoGroup.GET("/jobs/:job_id/logs", authMiddleware.Authenticate(), ciHandler.GetJobLogs)
		repoGroup.GET("/jobs/:job_id/stream", authMiddleware.Authenticate(), ciHandler.StreamLogs)
		repoGroup.GET("/jobs/:job_id/artifacts", authMiddleware.Authenticate(), ciHandler.ListArtifacts)
		repoGroup.GET("/jobs/:job_id/artifacts/:artifact_name", authMiddleware.Authenticate(), ciHandler.DownloadArtifact)

		// Protected routes (require authentication)
		repoGroup.POST("/jobs", authMiddleware.RequireAuth(), ciHandler.TriggerJob)
		repoGroup.POST("/jobs/:job_id/cancel", authMiddleware.RequireAuth(), ciHandler.CancelJob)
		repoGroup.POST("/jobs/:job_id/retry", authMiddleware.RequireAuth(), ciHandler.RetryJob)
	}

	// ========================================
	// Internal CI routes (called by CI runner)
	// ========================================
	// These routes are called by the CI runner to report status and logs
	ciInternalGroup := r.server.Group("/api/v1/ci")
	{
		// Receive logs from CI runner
		// The CI runner authenticates using Bearer token or API key
		ciInternalGroup.POST("/jobs/:job_id/logs", ciHandler.ReceiveLogs)

		// Receive job completion events from CI runner
		ciInternalGroup.POST("/jobs/:job_id/complete", ciHandler.CompleteJob)

		// Webhook for job status updates from CI runner
		ciInternalGroup.POST("/webhook/job-update", ciHandler.WebhookJobUpdate)
	}
}
