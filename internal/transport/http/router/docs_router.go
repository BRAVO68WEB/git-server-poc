package router

import (
	"github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/bravo68web/stasis/docs"
)

func (r *Router) docsRouter() {
	// Read the embedded OpenAPI spec (JSON format)
	specContent, err := docs.OpenAPISpec.ReadFile("openapi.json")
	if err != nil {
		// If we can't read the spec, log and return without setting up docs
		return
	}

	// Configure Scalar with the OpenAPI spec
	cfg := openapiui.Config{
		Title:   "Git server API Documentation",
		Theme:   "deepSpace",
		SpecURL: "/docs/openapi.json",
		SpecProvider: func() ([]byte, error) {
			return specContent, nil
		},
	}

	// Register the docs route
	r.server.GET("/docs/*any", openapiui.WrapHandler(cfg))
}
