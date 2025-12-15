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
	h := handler.NewAuthHandler(r.Deps.AuthService, r.Deps.UserService)

	auth := v1.Group("/auth")
	{
		// Public auth routes
		auth.POST("/register", h.Register)

		// Protected auth routes
		auth.GET("/me", authMiddleware.RequireAuth(), h.GetCurrentUser)
		auth.POST("/change-password", authMiddleware.RequireAuth(), h.ChangePassword)
	}
}
