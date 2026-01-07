package router

import (
	"net/http"

	"github.com/bravo68web/stasis/pkg/openapi"
	"github.com/gin-gonic/gin"
)

func (r *Router) docsRouter() {

	// Register OpenAPI Docs for the spec file
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/docs/openapi.yaml", openapi.RouteDocs{
		Summary:     "Get OpenAPI Spec",
		Description: "Download the OpenAPI specification in YAML format",
		Tags:        []string{"Documentation"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "OpenAPI YAML file",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("HEAD", "/docs/openapi.yaml", openapi.RouteDocs{
		Summary:     "Check OpenAPI Spec",
		Description: "Check availability of the OpenAPI specification",
		Tags:        []string{"Documentation"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "OpenAPI YAML file is available",
			},
		},
	})

	// Serve the OpenAPI spec file
	r.server.Engine.StaticFile("/docs/openapi.yaml", "docs/openapi.yaml")

	// Register OpenAPI Docs for the documentation UI
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/docs", openapi.RouteDocs{
		Summary:     "API Documentation",
		Description: "View the interactive API documentation (Scalar)",
		Tags:        []string{"Documentation"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "HTML Documentation",
			},
		},
	})

	// Serve the HTML documentation
	r.server.Engine.GET("/docs", func(c *gin.Context) {
		html := `
<!doctype html>
<html>
  <head>
    <title>Stasis API Documentation</title>
    <meta charset="utf-8" />
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1" />
    <style>
      body {
        margin: 0;
      }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/docs/openapi.yaml"
      data-proxy-url="https://proxy.scalar.com"
      src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
	})
}
