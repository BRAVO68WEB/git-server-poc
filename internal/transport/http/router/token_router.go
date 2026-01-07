package router

import (
	"net/http"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

// tokenRouter sets up personal access token management routes
func (r *Router) tokenRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	tokenHandler := handler.NewTokenHandler(r.Deps.TokenService)

	// Register Docs
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/tokens", openapi.RouteDocs{
		Summary:     "List tokens",
		Description: "Returns all personal access tokens for the authenticated user",
		Tags:        []string{"Tokens"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "List of tokens",
				Model:       dto.ListTokensResponse{},
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/tokens", openapi.RouteDocs{
		Summary:     "Create token",
		Description: "Creates a new personal access token",
		Tags:        []string{"Tokens"},
		RequestBody: dto.CreateTokenRequest{},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusCreated: {
				Description: "Token created successfully",
			},
			http.StatusBadRequest: {
				Description: "Invalid request",
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("DELETE", "/api/v1/tokens/:id", openapi.RouteDocs{
		Summary:     "Delete token",
		Description: "Deletes a personal access token by ID",
		Tags:        []string{"Tokens"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "Token deleted successfully",
				Model:       map[string]string{"message": "Token deleted successfully"},
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
			http.StatusNotFound: {
				Description: "Token not found",
			},
		},
	})

	// Token routes (require authentication)
	tokenGroup := v1.Group("/tokens")
	{
		tokenGroup.POST("", authMiddleware.RequireAuth(), tokenHandler.CreateToken)
		tokenGroup.GET("", authMiddleware.RequireAuth(), tokenHandler.ListTokens)
		tokenGroup.DELETE("/:id", authMiddleware.RequireAuth(), tokenHandler.DeleteToken)
	}
}
