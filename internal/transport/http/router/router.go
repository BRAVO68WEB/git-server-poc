package router

import (
	"github.com/bravo68web/githut/internal/server"
	"github.com/gin-contrib/cors"
)

type Router struct {
	server *server.Server
}

// NewRouter creates a new Router instance.
func NewRouter(s *server.Server) *Router {
	return &Router{
		server: s,
	}
}

// RegisterRoutes sets up the routes and middleware for the server.
func (r *Router) RegisterRoutes() {
	// Apply CORS middleware
	r.server.Use(cors.Default())

	r.healthRouter()
}
