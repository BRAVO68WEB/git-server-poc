package router

import (
	"net/http"

	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/pkg/openapi"
)

func (r *Router) healthRouter() {
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/", openapi.RouteDocs{
		Summary:     "Health check",
		Description: "Returns the health status of the API",
		Tags:        []string{"Health"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "API is healthy",
			},
		},
	})

	r.server.GET("/", handler.HealthHandler())
}
