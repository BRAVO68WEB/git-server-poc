package router

import (
	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

func (r *Router) authRouter() {
	v1 := r.server.Group("/api/v1")

	// Register OpenAPI Docs
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/auth/oidc/config", openapi.RouteDocs{
		Summary:     "Get OIDC configuration",
		Description: "Get the current OIDC configuration status",
		Tags:        []string{"Authentication"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.OIDCConfigResponse{},
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/auth/oidc/login", openapi.RouteDocs{
		Summary:     "Initiate OIDC login",
		Description: "Initiates the OIDC login flow by redirecting to the identity provider",
		Tags:        []string{"Authentication"},
		Responses: map[int]openapi.ResponseDoc{
			302: {
				Description: "Redirect to Identity Provider",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/auth/oidc/callback", openapi.RouteDocs{
		Summary:     "OIDC Callback",
		Description: "Handle the callback from the identity provider",
		Tags:        []string{"Authentication"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Authentication successful",
				Model:       dto.OIDCCallbackResponse{},
			},
			401: {
				Description: "Authentication failed",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/auth/oidc/logout", openapi.RouteDocs{
		Summary:     "Logout",
		Description: "Invalidate the current session",
		Tags:        []string{"Authentication"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Logged out successfully",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/auth/me", openapi.RouteDocs{
		Summary:     "Get current user",
		Description: "Get information about the currently authenticated user",
		Tags:        []string{"Authentication"},
		Responses: map[int]openapi.ResponseDoc{
			200: {
				Description: "Successful response",
				Model:       dto.UserInfo{},
			},
			401: {
				Description: "Unauthorized",
			},
		},
	})

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handlers
	h := handler.NewAuthHandler(r.Deps.AuthService, r.Deps.UserService, r.Deps.OIDCService, r.server.Config)

	auth := v1.Group("/auth")
	{
		// OIDC authentication routes (public)
		oidc := auth.Group("/oidc")
		{
			// Get OIDC configuration status
			oidc.GET("/config", h.GetOIDCConfig)

			// Initiate OIDC login flow - redirects to identity provider
			oidc.GET("/login", h.OIDCLogin)

			// OIDC callback - handles response from identity provider
			oidc.GET("/callback", h.OIDCCallback)

			// Logout - invalidates session and optionally redirects to provider logout
			oidc.POST("/logout", h.OIDCLogout)
		}

		// Protected auth routes
		auth.GET("/me", authMiddleware.RequireAuth(), h.GetCurrentUser)
	}
}
