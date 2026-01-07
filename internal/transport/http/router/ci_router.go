package router

import (
	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

// ciRouter sets up CI-related routes
func (r *Router) ciRouter() {
	// Use the shared CI service from dependencies
	ciService := r.Deps.CIService

	// Initialize CI handler
	ciHandler := handler.NewCIHandler(ciService, r.Deps.RepoService.GetRepoRepository())

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// CI Routes Documentation
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/latest", openapi.RouteDocs{
		Summary:     "Get latest job",
		Description: "Get the latest CI job for a repository",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CIJobResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs", openapi.RouteDocs{
		Summary:     "List jobs",
		Description: "List CI jobs for a repository",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CIJobListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id", openapi.RouteDocs{
		Summary:     "Get job",
		Description: "Get details of a specific CI job",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CIJobResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/logs", openapi.RouteDocs{
		Summary:     "Get job logs",
		Description: "Get logs for a specific CI job",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CILogsResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/artifacts", openapi.RouteDocs{
		Summary:     "List artifacts",
		Description: "List artifacts for a CI job",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CIArtifactListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/ci/jobs", openapi.RouteDocs{
		Summary:     "Trigger job",
		Description: "Trigger a new CI job",
		Tags:        []string{"CI"},
		RequestBody: handler.TriggerJobRequest{},
		Responses: map[int]openapi.ResponseDoc{
			202: {
				Description: "Job triggered successfully",
				Model:       dto.CIJobTriggerResponse{},
			},
			400: {
				Description: "Invalid request",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/cancel", openapi.RouteDocs{
		Summary:     "Cancel job",
		Description: "Cancel a running CI job",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Job cancelled successfully",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/retry", openapi.RouteDocs{
		Summary:     "Retry job",
		Description: "Retry a failed CI job",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			202: {
				Description: "Job retry initiated",
				Model:       dto.CIJobTriggerResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/stream", openapi.RouteDocs{
		Summary:     "Stream logs",
		Description: "Stream logs for a CI job via SSE",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful stream",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or job not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/ci/jobs/:job_id/artifacts/:artifact_name", openapi.RouteDocs{
		Summary:     "Download artifact",
		Description: "Download a specific artifact",
		Tags:        []string{"CI"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful download",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository, job or artifact not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/ci/jobs/:job_id/logs", openapi.RouteDocs{
		Summary:     "Receive logs",
		Description: "Receive logs from CI runner",
		Tags:        []string{"CI Internal"},
		RequestBody: []dto.CIRunnerLogEntryRequest{},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Logs received",
			},
			400: {
				Description: "Invalid request",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/ci/jobs/:job_id/complete", openapi.RouteDocs{
		Summary:     "Complete job",
		Description: "Mark job as complete",
		Tags:        []string{"CI Internal"},
		RequestBody: dto.CIJobCompleteRequest{},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Job completed",
			},
			400: {
				Description: "Invalid request",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/ci/webhook/job-update", openapi.RouteDocs{
		Summary:     "Job update webhook",
		Description: "Receive generic job updates",
		Tags:        []string{"CI Internal"},
		RequestBody: dto.CIWebhookJobUpdateRequest{},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Update received",
			},
			400: {
				Description: "Invalid request",
			},
		},
	})

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
