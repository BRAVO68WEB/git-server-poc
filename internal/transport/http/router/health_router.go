package router

import (
	"github.com/bravo68web/githut/internal/transport/http/handler"
)

func (r *Router) healthRouter() {
	r.server.GET("/", handler.HealthHandler())
}
