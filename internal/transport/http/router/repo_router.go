package router

import (
	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

func (r *Router) repoRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	h := handler.NewRepoHandler(
		r.Deps.RepoService,
		r.Deps.MirrorSyncService,
		r.server.Config.Server.Host,
		r.server.Config.SSH.Host,
		r.server.Config.SSH.Port,
	)

	// Register OpenAPI Docs
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/public", openapi.RouteDocs{
		Summary:     "List public repositories",
		Description: "Get a list of public repositories with pagination",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.RepoListResponse{},
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos", openapi.RouteDocs{
		Summary:     "Create repository",
		Description: "Create a new repository for the authenticated user",
		Tags:        []string{"Repositories"},
		RequestBody: dto.CreateRepoRequest{},
		Responses: map[int]openapi.ResponseDoc{
			201: {
				Description: "Repository created successfully",
				Model:       dto.RepoResponse{},
			},
			400: {
				Description: "Invalid request",
			},
			401: {
				Description: "Unauthorized",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/import", openapi.RouteDocs{
		Summary:     "Import repository",
		Description: "Import a repository from an external Git source",
		Tags:        []string{"Repositories"},
		RequestBody: dto.ImportRepoRequest{},
		Responses: map[int]openapi.ResponseDoc{
			201: {
				Description: "Repository imported successfully",
				Model:       dto.RepoResponse{},
			},
			400: {
				Description: "Invalid request",
			},
			401: {
				Description: "Unauthorized",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos", openapi.RouteDocs{
		Summary:     "List user repositories",
		Description: "Get a list of repositories for the authenticated user",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.RepoListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo", openapi.RouteDocs{
		Summary:     "Get repository",
		Description: "Get detailed information about a specific repository",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.RepoResponse{},
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("PATCH", "/api/v1/repos/:owner/:repo", openapi.RouteDocs{
		Summary:     "Update repository",
		Description: "Update repository details",
		Tags:        []string{"Repositories"},
		RequestBody: dto.UpdateRepoRequest{},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Repository updated successfully",
				Model:       dto.RepoResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("DELETE", "/api/v1/repos/:owner/:repo", openapi.RouteDocs{
		Summary:     "Delete repository",
		Description: "Delete a repository permanently",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Repository deleted successfully",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/stats", openapi.RouteDocs{
		Summary:     "Get repository stats",
		Description: "Get statistics for a repository",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.RepoStatsResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/branches", openapi.RouteDocs{
		Summary:     "List branches",
		Description: "List all branches in the repository",
		Tags:        []string{"Branches"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.BranchListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/branches", openapi.RouteDocs{
		Summary:     "Create branch",
		Description: "Create a new branch",
		Tags:        []string{"Branches"},
		RequestBody: dto.BranchRequest{},
		Responses: map[int]openapi.ResponseDoc{
			201: {
				Description: "Branch created successfully",
				Model:       dto.BranchResponse{},
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

	r.server.OpenAPIGenerator.RegisterDocs("DELETE", "/api/v1/repos/:owner/:repo/branches/:branch", openapi.RouteDocs{
		Summary:     "Delete branch",
		Description: "Delete a branch",
		Tags:        []string{"Branches"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Branch deleted successfully",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/tags", openapi.RouteDocs{
		Summary:     "List tags",
		Description: "List all tags in the repository",
		Tags:        []string{"Tags"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.TagListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/tags", openapi.RouteDocs{
		Summary:     "Create tag",
		Description: "Create a new tag",
		Tags:        []string{"Tags"},
		RequestBody: dto.TagRequest{},
		Responses: map[int]openapi.ResponseDoc{
			201: {
				Description: "Tag created successfully",
				Model:       dto.TagResponse{},
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

	r.server.OpenAPIGenerator.RegisterDocs("DELETE", "/api/v1/repos/:owner/:repo/tags/:tag", openapi.RouteDocs{
		Summary:     "Delete tag",
		Description: "Delete a tag",
		Tags:        []string{"Tags"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Tag deleted successfully",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/commits", openapi.RouteDocs{
		Summary:     "List commits",
		Description: "List commits in the repository",
		Tags:        []string{"Commits"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CommitListResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/commits/:sha", openapi.RouteDocs{
		Summary:     "Get commit",
		Description: "Get details of a specific commit",
		Tags:        []string{"Commits"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.CommitResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or commit not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/diff/:hash", openapi.RouteDocs{
		Summary:     "Get diff",
		Description: "Get diff for a commit",
		Tags:        []string{"Commits"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.DiffResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or commit not found",
			},
		},
	})
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/compare/:range", openapi.RouteDocs{
		Summary:     "Compare diff",
		Description: "Compare changes between two commits",
		Tags:        []string{"Commits"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.DiffResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or commit not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/tree/:ref", openapi.RouteDocs{
		Summary:     "Get tree",
		Description: "Get file tree for a reference",
		Tags:        []string{"Code"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.TreeResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or ref not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/tree/:ref/*path", openapi.RouteDocs{
		Summary:     "Get tree with path",
		Description: "Get file tree for a reference and path",
		Tags:        []string{"Code"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.TreeResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or ref not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/blob/:ref/*path", openapi.RouteDocs{
		Summary:     "Get file content",
		Description: "Get content of a specific file",
		Tags:        []string{"Code"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.FileContentResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or file not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/blame/:ref/*path", openapi.RouteDocs{
		Summary:     "Get blame",
		Description: "Get blame information for a file",
		Tags:        []string{"Code"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.BlameResponse{},
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository or file not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/repos/:owner/:repo/sync", openapi.RouteDocs{
		Summary:     "Sync mirror repository",
		Description: "Trigger a sync for a mirror repository",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			202: {
				Description: "Sync started",
			},
			400: {
				Description: "Not a mirror repository",
			},
			401: {
				Description: "Unauthorized",
			},
			403: {
				Description: "Forbidden",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/repos/:owner/:repo/mirror/status", openapi.RouteDocs{
		Summary:     "Get mirror status",
		Description: "Get sync status of a mirror repository",
		Tags:        []string{"Repositories"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
			},
			400: {
				Description: "Not a mirror repository",
			},
			401: {
				Description: "Unauthorized",
			},
			404: {
				Description: "Repository not found",
			},
		},
	})

	// Repository routes
	repos := v1.Group("/repos")
	{
		// List public repositories (no auth required)
		repos.GET("/public", h.ListPublicRepositories)

		// Protected repository routes
		repos.POST("", authMiddleware.RequireAuth(), h.CreateRepository)
		repos.POST("/import", authMiddleware.RequireAuth(), h.ImportRepository)
		repos.GET("", authMiddleware.RequireAuth(), h.ListRepositories)

		// Repository-specific routes
		repoRoutes := repos.Group("/:owner/:repo")
		{
			repoRoutes.GET("", authMiddleware.Authenticate(), h.GetRepository)
			repoRoutes.PATCH("", authMiddleware.RequireAuth(), h.UpdateRepository)
			repoRoutes.DELETE("", authMiddleware.RequireAuth(), h.DeleteRepository)
			repoRoutes.GET("/stats", authMiddleware.Authenticate(), h.GetRepositoryStats)

			// Branch routes
			repoRoutes.GET("/branches", authMiddleware.Authenticate(), h.ListBranches)
			repoRoutes.POST("/branches", authMiddleware.RequireAuth(), h.CreateBranch)
			repoRoutes.DELETE("/branches/:branch", authMiddleware.RequireAuth(), h.DeleteBranch)

			// Tag routes
			repoRoutes.GET("/tags", authMiddleware.Authenticate(), h.ListTags)
			repoRoutes.POST("/tags", authMiddleware.RequireAuth(), h.CreateTag)
			repoRoutes.DELETE("/tags/:tag", authMiddleware.RequireAuth(), h.DeleteTag)

			// Commit routes
			repoRoutes.GET("/commits", authMiddleware.Authenticate(), h.ListCommits)
			repoRoutes.GET("/commits/:sha", authMiddleware.Authenticate(), h.GetCommit)
			repoRoutes.GET("/diff/:hash", authMiddleware.Authenticate(), h.GetDiff)
			repoRoutes.GET("/compare/:range", authMiddleware.Authenticate(), h.GetCompareDiff)

			// Tree/code structure routes
			repoRoutes.GET("/tree/:ref", authMiddleware.Authenticate(), h.GetTree)
			repoRoutes.GET("/tree/:ref/*path", authMiddleware.Authenticate(), h.GetTree)

			// File content routes
			repoRoutes.GET("/blob/:ref/*path", authMiddleware.Authenticate(), h.GetFileContent)

			// Blame routes
			repoRoutes.GET("/blame/:ref/*path", authMiddleware.Authenticate(), h.GetBlame)

			// Mirror sync routes
			repoRoutes.POST("/sync", authMiddleware.RequireAuth(), h.SyncMirror)
			repoRoutes.GET("/mirror/status", authMiddleware.Authenticate(), h.GetMirrorStatus)

			// Mirror settings routes
			repoRoutes.GET("/mirror", authMiddleware.Authenticate(), h.GetMirrorSettings)
			repoRoutes.PATCH("/mirror", authMiddleware.RequireAuth(), h.UpdateMirrorSettings)
		}
	}
}
