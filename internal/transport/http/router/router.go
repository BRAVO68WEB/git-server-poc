package router

import (
	"github.com/bravo68web/stasis/internal/injectable"
	"github.com/bravo68web/stasis/internal/server"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
)

type Router struct {
	server *server.Server
	Deps   *injectable.Dependencies
}

// NewRouter creates a new Router instance.
func NewRouter(s *server.Server) *Router {
	deps := injectable.LoadDependencies(s.Config, s.DB)

	return &Router{
		server: s,
		Deps:   &deps,
	}
}

// RegisterRoutes sets up the routes and middleware for the server.
func (r *Router) RegisterRoutes() {
	// Get allowed origins from config, default to localhost for development
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}

	// Add configured frontend URL if available (from OIDC config)
	if r.server.Config.OIDC.FrontendURL != "" {
		allowedOrigins = append(allowedOrigins, r.server.Config.OIDC.FrontendURL)
	}

	// Apply CORS middleware
	r.server.Use(middleware.CORSMiddleware(allowedOrigins))

	r.docsRouter()

	r.healthRouter()
	r.authRouter()
	r.repoRouter()
	r.gitRouter()
	r.sshKeyRouter()
	r.tokenRouter()
}
