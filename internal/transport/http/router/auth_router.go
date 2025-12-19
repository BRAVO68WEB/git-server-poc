package router

import (
	"github.com/bravo68web/githut/internal/transport/http/handler"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
)

func (r *Router) authRouter() {
	v1 := r.server.Group("/api/v1")

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
