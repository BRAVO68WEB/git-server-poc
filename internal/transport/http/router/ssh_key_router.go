package router

import (
	"github.com/bravo68web/githut/internal/transport/http/handler"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
)

// sshKeyRouter sets up SSH key management routes
func (r *Router) sshKeyRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	sshKeyHandler := handler.NewSSHKeyHandler(r.Deps.SSHKeyService)

	// SSH key routes (require authentication)
	sshKeyGroup := v1.Group("/ssh-keys")
	{
		sshKeyGroup.POST("", authMiddleware.RequireAuth(), sshKeyHandler.AddSSHKey)
		sshKeyGroup.GET("", authMiddleware.RequireAuth(), sshKeyHandler.ListSSHKeys)
		sshKeyGroup.GET("/:id", authMiddleware.RequireAuth(), sshKeyHandler.GetSSHKey)
		sshKeyGroup.DELETE("/:id", authMiddleware.RequireAuth(), sshKeyHandler.DeleteSSHKey)
	}
}
