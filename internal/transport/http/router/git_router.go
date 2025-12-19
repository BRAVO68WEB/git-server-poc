package router

import (
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
)

func (r *Router) gitRouter() {
	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize git handler
	h := handler.NewGitHandler(
		r.Deps.GitService,
		r.Deps.RepoService,
		r.Deps.AuthService,
		r.Deps.Storage,
	)

	// Git Smart HTTP Protocol routes
	// These routes handle git clone, fetch, and push operations
	// We use a group to avoid route conflicts with the repo API routes

	// Create a group for git operations
	// Pattern: /:owner/:repo.git/... (repos accessed with .git suffix for git operations)
	gitGroup := r.server.Group("/:owner/:repo")
	gitGroup.Use(authMiddleware.Authenticate())
	{
		// Git info/refs endpoint - used for capability advertisement
		// GET /:owner/:repo/info/refs?service=git-upload-pack|git-receive-pack
		gitGroup.GET("/info/refs", h.HandleInfoRefs)

		// Git upload-pack endpoint - handles fetch/clone operations (read)
		// POST /:owner/:repo/git-upload-pack
		gitGroup.POST("/git-upload-pack", h.HandleUploadPack)

		// Git receive-pack endpoint - handles push operations (write)
		// POST /:owner/:repo/git-receive-pack
		gitGroup.POST("/git-receive-pack", h.HandleReceivePack)

		// Dumb HTTP Protocol fallback routes
		// These are used by older git clients or when smart protocol is not available

		// GET /:owner/:repo/HEAD
		gitGroup.GET("/HEAD", h.HandleGetHEAD)

		// Objects routes - need to be ordered from most specific to least specific
		// to avoid Gin router conflicts

		// GET /:owner/:repo/objects/info/packs (must be before wildcard route)
		gitGroup.GET("/objects/info/packs", h.HandleGetInfoPacks)

		// GET /:owner/:repo/objects/info/alternates
		gitGroup.GET("/objects/info/alternates", h.HandleGetInfoPacks)

		// GET /:owner/:repo/objects/pack/:packfile (for pack files)
		gitGroup.GET("/objects/pack/:packfile", h.HandleGetObject)

		// GET /:owner/:repo/objects/:dir/:file (for loose objects - 2 char dir + 38 char filename)
		gitGroup.GET("/objects/:dir/:file", h.HandleGetObject)

		// Refs routes
		// GET /:owner/:repo/refs/heads/:branch
		gitGroup.GET("/refs/heads/:branch", h.HandleGetRefs)

		// GET /:owner/:repo/refs/tags/:tag
		gitGroup.GET("/refs/tags/:tag", h.HandleGetRefs)
	}
}
