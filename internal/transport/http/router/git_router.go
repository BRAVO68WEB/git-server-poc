package router

import (
	"net/http"

	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

func (r *Router) gitRouter() {
	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize git handler with CI service for triggering CI on push
	h := handler.NewGitHandler(
		r.Deps.GitService,
		r.Deps.RepoService,
		r.Deps.AuthService,
		r.Deps.Storage,
		r.Deps.CIService,
	)

	// Register Docs
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/info/refs", openapi.RouteDocs{
		Summary:     "Git info/refs",
		Description: "Advertises capabilities and refs for git-discovery",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Git references advertisement"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/:owner/:repo/git-upload-pack", openapi.RouteDocs{
		Summary:     "Git upload-pack",
		Description: "Handles git fetch/clone operations",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Pack data"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/:owner/:repo/git-receive-pack", openapi.RouteDocs{
		Summary:     "Git receive-pack",
		Description: "Handles git push operations",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Push status"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/HEAD", openapi.RouteDocs{
		Summary:     "Git HEAD",
		Description: "Get repository HEAD",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "HEAD content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/objects/info/packs", openapi.RouteDocs{
		Summary:     "Get packs info",
		Description: "Discover available packs",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Packs info content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/objects/info/alternates", openapi.RouteDocs{
		Summary:     "Get alternates info",
		Description: "Discover alternate object stores",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Alternates info content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/objects/pack/:packfile", openapi.RouteDocs{
		Summary:     "Get pack file",
		Description: "Download a pack file",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Pack file content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/objects/:dir/:file", openapi.RouteDocs{
		Summary:     "Get loose object",
		Description: "Download a loose object",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Object content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/refs/heads/:branch", openapi.RouteDocs{
		Summary:     "Get branch ref",
		Description: "Get specific branch reference",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Branch reference content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/:owner/:repo/refs/tags/:tag", openapi.RouteDocs{
		Summary:     "Get tag ref",
		Description: "Get specific tag reference",
		Tags:        []string{"Git Protocol"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK:           {Description: "Tag reference content"},
			http.StatusUnauthorized: {Description: "Authentication required"},
		},
	})

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
