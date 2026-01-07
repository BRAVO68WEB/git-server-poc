package router

import (
	"net/http"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

// sshKeyRouter sets up SSH key management routes
func (r *Router) sshKeyRouter() {
	v1 := r.server.Group("/api/v1")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)

	// Initialize handler
	sshKeyHandler := handler.NewSSHKeyHandler(r.Deps.SSHKeyService)

	// Register Docs
	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/ssh-keys", openapi.RouteDocs{
		Summary:     "List SSH keys",
		Description: "Returns all SSH keys for the authenticated user",
		Tags:        []string{"SSH Keys"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "List of SSH keys",
				Model:       dto.ListSSHKeysResponse{},
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("POST", "/api/v1/ssh-keys", openapi.RouteDocs{
		Summary:     "Add SSH key",
		Description: "Adds a new SSH public key for the authenticated user",
		Tags:        []string{"SSH Keys"},
		RequestBody: dto.AddSSHKeyRequest{},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusCreated: {
				Description: "SSH key added successfully",
				Model:       dto.AddSSHKeyResponse{},
			},
			http.StatusBadRequest: {
				Description: "Invalid SSH key format",
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
			http.StatusConflict: {
				Description: "SSH key already exists",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("GET", "/api/v1/ssh-keys/:id", openapi.RouteDocs{
		Summary:     "Get SSH key",
		Description: "Returns a specific SSH key by ID",
		Tags:        []string{"SSH Keys"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "SSH key information",
				Model:       dto.SSHKeyInfo{},
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
			http.StatusNotFound: {
				Description: "SSH key not found",
			},
		},
	})

	r.server.OpenAPIGenerator.RegisterDocs("DELETE", "/api/v1/ssh-keys/:id", openapi.RouteDocs{
		Summary:     "Delete SSH key",
		Description: "Deletes an SSH key by ID",
		Tags:        []string{"SSH Keys"},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "SSH key deleted successfully",
				Model:       map[string]string{"message": "SSH key deleted successfully"},
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
			http.StatusNotFound: {
				Description: "SSH key not found",
			},
		},
	})

	// SSH key routes (require authentication)
	sshKeyGroup := v1.Group("/ssh-keys")
	{
		sshKeyGroup.POST("", authMiddleware.RequireAuth(), sshKeyHandler.AddSSHKey)
		sshKeyGroup.GET("", authMiddleware.RequireAuth(), sshKeyHandler.ListSSHKeys)
		sshKeyGroup.GET("/:id", authMiddleware.RequireAuth(), sshKeyHandler.GetSSHKey)
		sshKeyGroup.DELETE("/:id", authMiddleware.RequireAuth(), sshKeyHandler.DeleteSSHKey)
	}
}
