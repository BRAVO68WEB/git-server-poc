package router

import (
	"github.com/bravo68web/githut/internal/transport/http/handler"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
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
	// Pattern: /{owner}/{repo}.git/...

	// Git info/refs endpoint - used for capability advertisement
	// GET /{owner}/{repo}.git/info/refs?service=git-upload-pack|git-receive-pack
	r.server.GET("/:owner/:repo/info/refs", authMiddleware.Authenticate(), h.HandleInfoRefs)

	// Git upload-pack endpoint - handles fetch/clone operations (read)
	// POST /{owner}/{repo}.git/git-upload-pack
	r.server.POST("/:owner/:repo/git-upload-pack", authMiddleware.Authenticate(), h.HandleUploadPack)

	// Git receive-pack endpoint - handles push operations (write)
	// POST /{owner}/{repo}.git/git-receive-pack
	r.server.POST("/:owner/:repo/git-receive-pack", authMiddleware.Authenticate(), h.HandleReceivePack)

	// Dumb HTTP Protocol fallback routes
	// These are used by older git clients or when smart protocol is not available

	// GET /{owner}/{repo}/HEAD
	r.server.GET("/:owner/:repo/HEAD", authMiddleware.Authenticate(), h.HandleGetHEAD)

	// GET /{owner}/{repo}/objects/info/packs
	r.server.GET("/:owner/:repo/objects/info/packs", authMiddleware.Authenticate(), h.HandleGetInfoPacks)

	// GET /{owner}/{repo}/objects/* (loose objects and pack files)
	r.server.GET("/:owner/:repo/objects/*path", authMiddleware.Authenticate(), h.HandleGetObject)

	// GET /{owner}/{repo}/refs/* (ref files)
	r.server.GET("/:owner/:repo/refs/*path", authMiddleware.Authenticate(), h.HandleGetRefs)
}
