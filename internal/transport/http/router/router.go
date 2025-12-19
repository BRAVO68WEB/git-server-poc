package router

import (
	"github.com/bravo68web/stasis/internal/injectable"
	"github.com/bravo68web/stasis/internal/server"
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
	// Apply CORS middleware
	r.server.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.docsRouter()

	r.healthRouter()
	r.authRouter()
	r.repoRouter()
	r.gitRouter()
	r.sshKeyRouter()
}
