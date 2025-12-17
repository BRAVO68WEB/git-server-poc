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
	// Apply CORS middleware
	r.server.Use(cors.Default())

	r.docsRouter()

	r.healthRouter()
	r.authRouter()
	r.repoRouter()
	r.gitRouter()
}
