package router

import (
	"github.com/bravo68web/githut/internal/injectable"
	"github.com/bravo68web/githut/internal/server"
	"github.com/gin-contrib/cors"
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

	// Apply CORS middleware with cookie support
	r.server.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Cache-Control",
			"Cookie",
			"X-Requested-With",
			"X-Auth-Token",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"Set-Cookie",
			"Authorization",
		},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours preflight cache
	}))

	r.docsRouter()

	r.healthRouter()
	r.authRouter()
	r.repoRouter()
	r.gitRouter()
	r.sshKeyRouter()
	r.tokenRouter()
}
