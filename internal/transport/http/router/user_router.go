package router

import (
	"net/http"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/transport/http/handler"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/openapi"
)

func (r *Router) userRouter() {
	// Register base route
	v1 := r.server.Group("/api/v1")
	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(r.Deps.AuthService)
	// Initialize handlers
	userHandler := handler.NewUserHandler(r.Deps.UserService)

	// Register Docs
	r.server.OpenAPIGenerator.RegisterDocs("PUT", "/api/v1/users/username", openapi.RouteDocs{
		Summary:     "Update username",
		Description: "Updates the current user's username",
		Tags:        []string{"Users"},
		RequestBody: dto.UpdateUserRequest{},
		Responses: map[int]openapi.ResponseDoc{
			http.StatusOK: {
				Description: "Username updated successfully",
				Model:       dto.UserInfo{},
			},
			http.StatusBadRequest: {
				Description: "Invalid request",
			},
			http.StatusUnauthorized: {
				Description: "Authentication required",
			},
			http.StatusConflict: {
				Description: "Username already taken",
			},
		},
	})

	// Register user routes
	userGroup := v1.Group("/users")
	{
		userGroup.Use(authMiddleware.RequireAuth())
		userGroup.PUT("/username", userHandler.UpdateCurrentUsername)
	}
}
