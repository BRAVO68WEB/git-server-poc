package router

import (
	"github.com/bravo68web/stasis/internal/transport/http/handler"
)

func (r *Router) healthRouter() {
	r.server.GET("/", handler.HealthHandler())
}
