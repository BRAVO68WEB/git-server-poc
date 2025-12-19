package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/service"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// UserContextKey is the key for storing user in context
	UserContextKey ContextKey = "user"

	// TokenContextKey is the key for storing token info in context
	TokenContextKey ContextKey = "token"

	// IsAuthenticatedKey is the key for storing authentication status
	IsAuthenticatedKey ContextKey = "is_authenticated"
)

// AuthMiddleware handles authentication for HTTP requests
type AuthMiddleware struct {
	authService service.AuthService
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate attempts to authenticate the request but doesn't require it
// This is useful for endpoints that work differently for authenticated vs anonymous users
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)
		if user != nil {
			m.setUserContext(c, user)
		}
		c.Next()
	}
}

// RequireAuth requires authentication for the endpoint
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		m.setUserContext(c, user)
		c.Next()
	}
}

// RequireAdmin requires admin privileges
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "admin privileges required",
			})
			return
		}

		m.setUserContext(c, user)
		c.Next()
	}
}

// extractAndValidateUser extracts and validates the user from the request
// Supports:
// - Bearer token (session JWT from OIDC or PAT)
// - Query parameter access_token (for git operations)
func (m *AuthMiddleware) extractAndValidateUser(c *gin.Context) *models.User {
	ctx := c.Request.Context()

	// Try Bearer token first (Authorization header)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Try session token authentication first (OIDC JWT)
		user, err := m.authService.AuthenticateSession(ctx, token)
		if err == nil && user != nil {
			return user
		}

		// If session auth fails, try PAT (Personal Access Token) authentication
		user, err = m.authService.AuthenticateToken(ctx, token)
		if err == nil && user != nil {
			return user
		}
	}

	// Try token from query parameter (for git operations)
	if token := c.Query("access_token"); token != "" {
		// Try session token authentication first
		user, err := m.authService.AuthenticateSession(ctx, token)
		if err == nil && user != nil {
			return user
		}

		// Try PAT authentication
		user, err = m.authService.AuthenticateToken(ctx, token)
		if err == nil && user != nil {
			return user
		}
	}

	return nil
}

// setUserContext sets the user in the gin context
func (m *AuthMiddleware) setUserContext(c *gin.Context, user *models.User) {
	c.Set(string(UserContextKey), user)
	c.Set(string(IsAuthenticatedKey), true)

	// Also set in request context for downstream handlers
	ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
	ctx = context.WithValue(ctx, IsAuthenticatedKey, true)
	c.Request = c.Request.WithContext(ctx)
}

// GetUserFromContext retrieves the authenticated user from the context
func GetUserFromContext(c *gin.Context) *models.User {
	if user, exists := c.Get(string(UserContextKey)); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	if authenticated, exists := c.Get(string(IsAuthenticatedKey)); exists {
		if auth, ok := authenticated.(bool); ok {
			return auth
		}
	}
	return false
}

// GetUserFromRequestContext retrieves the user from the request context
func GetUserFromRequestContext(ctx context.Context) *models.User {
	if user := ctx.Value(UserContextKey); user != nil {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// OptionalAuth is an alias for Authenticate for readability
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return m.Authenticate()
}

// AuthForGit handles authentication for Git HTTP protocol
// It considers public repositories for read operations
func (m *AuthMiddleware) AuthForGit(isWrite bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)

		// For write operations, always require auth
		if isWrite && user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required for push",
			})
			return
		}

		// Set user if authenticated
		if user != nil {
			m.setUserContext(c, user)
		}

		c.Next()
	}
}
