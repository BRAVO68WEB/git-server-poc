package router

import (
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
)

// tokenRouter sets up personal access token management routes
func (r *Router) tokenRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	tokenHandler := handler.NewTokenHandler(r.Deps.TokenService)

	// Token routes (require authentication)
	tokenGroup := v1.Group("/tokens")
	{
		tokenGroup.POST("", authMiddleware.RequireAuth(), tokenHandler.CreateToken)
		tokenGroup.GET("", authMiddleware.RequireAuth(), tokenHandler.ListTokens)
		tokenGroup.DELETE("/:id", authMiddleware.RequireAuth(), tokenHandler.DeleteToken)
	}
}
