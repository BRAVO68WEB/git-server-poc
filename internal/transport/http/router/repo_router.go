package router

import (
	"github.com/bravo68web/githut/internal/transport/http/handler"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
)

func (r *Router) repoRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	h := handler.NewRepoHandler(
		r.Deps.RepoService,
		r.server.Config.Server.Host,
		r.server.Config.SSH.Host,
		r.server.Config.SSH.Port,
	)

	// Repository routes
	repos := v1.Group("/repos")
	{
		// List public repositories (no auth required)
		repos.GET("/public", h.ListPublicRepositories)

		// Protected repository routes
		repos.POST("", authMiddleware.RequireAuth(), h.CreateRepository)
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

			// Tree/code structure routes
			repoRoutes.GET("/tree/:ref", authMiddleware.Authenticate(), h.GetTree)
			repoRoutes.GET("/tree/:ref/*path", authMiddleware.Authenticate(), h.GetTree)

			// File content routes
			repoRoutes.GET("/blob/:ref/*path", authMiddleware.Authenticate(), h.GetFileContent)

			// Blame routes
			repoRoutes.GET("/blame/:ref/*path", authMiddleware.Authenticate(), h.GetBlame)
		}
	}
}
