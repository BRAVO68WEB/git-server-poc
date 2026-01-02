package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/pkg/logger"
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
	log         *logger.Logger
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		log:         logger.Get().WithFields(logger.Component("auth-middleware")),
	}
}

// Authenticate attempts to authenticate the request but doesn't require it
// This is useful for endpoints that work differently for authenticated vs anonymous users
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)
		if user != nil {
			m.log.Debug("User authenticated (optional auth)",
				logger.String("user_id", user.ID.String()),
				logger.String("username", user.Username),
				logger.Path(c.Request.URL.Path),
			)
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
			m.log.Warn("Authentication required but not provided",
				logger.Path(c.Request.URL.Path),
				logger.Method(c.Request.Method),
				logger.ClientIP(c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		m.log.Debug("User authenticated successfully",
			logger.String("user_id", user.ID.String()),
			logger.String("username", user.Username),
			logger.Path(c.Request.URL.Path),
		)
		m.setUserContext(c, user)
		c.Next()
	}
}

// RequireAdmin requires admin privileges
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.extractAndValidateUser(c)
		if user == nil {
			m.log.Warn("Admin access attempted without authentication",
				logger.Path(c.Request.URL.Path),
				logger.Method(c.Request.Method),
				logger.ClientIP(c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		if !user.IsAdmin {
			m.log.Warn("Non-admin user attempted to access admin endpoint",
				logger.String("user_id", user.ID.String()),
				logger.String("username", user.Username),
				logger.Path(c.Request.URL.Path),
				logger.Method(c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "admin privileges required",
			})
			return
		}

		m.log.Debug("Admin user authenticated",
			logger.String("user_id", user.ID.String()),
			logger.String("username", user.Username),
			logger.Path(c.Request.URL.Path),
		)
		m.setUserContext(c, user)
		c.Next()
	}
}

// extractAndValidateUser extracts and validates the user from the request
// Supports:
// - Bearer token (session JWT from OIDC or PAT)
// - Basic Auth (username:password where password is a PAT for git operations)
// - Query parameter access_token (for git operations)
func (m *AuthMiddleware) extractAndValidateUser(c *gin.Context) *models.User {
	ctx := c.Request.Context()

	authHeader := c.GetHeader("Authorization")

	println("authHeader", authHeader)

	// Try Bearer token first (Authorization header)
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Try session token authentication first (OIDC JWT)
		user, err := m.authService.AuthenticateSession(ctx, token)
		if err == nil && user != nil {
			m.log.Debug("User authenticated via session token",
				logger.String("user_id", user.ID.String()),
				logger.String("auth_method", "session_token"),
			)
			return user
		}

		// If session auth fails, try PAT (Personal Access Token) authentication
		user, err = m.authService.AuthenticateToken(ctx, token)
		if err == nil && user != nil {
			m.log.Debug("User authenticated via PAT",
				logger.String("user_id", user.ID.String()),
				logger.String("auth_method", "pat"),
			)
			return user
		}
	}

	// Try Basic Auth (for Git HTTP protocol)
	// Git sends credentials as Basic Auth with username and password/token
	if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
		user := m.authenticateBasic(ctx, authHeader)
		if user != nil {
			m.log.Debug("User authenticated via Basic Auth",
				logger.String("user_id", user.ID.String()),
				logger.String("auth_method", "basic_auth"),
			)
			return user
		}
	}

	// Try token from query parameter (for git operations)
	if token := c.Query("access_token"); token != "" {
		// Try session token authentication first
		user, err := m.authService.AuthenticateSession(ctx, token)
		if err == nil && user != nil {
			m.log.Debug("User authenticated via query param session token",
				logger.String("user_id", user.ID.String()),
				logger.String("auth_method", "query_session_token"),
			)
			return user
		}

		// Try PAT authentication
		user, err = m.authService.AuthenticateToken(ctx, token)
		if err == nil && user != nil {
			m.log.Debug("User authenticated via query param PAT",
				logger.String("user_id", user.ID.String()),
				logger.String("auth_method", "query_pat"),
			)
			return user
		}
	}

	return nil
}

// authenticateBasic handles Basic authentication for Git HTTP protocol
// The password field can be a Personal Access Token (PAT)
func (m *AuthMiddleware) authenticateBasic(ctx context.Context, authHeader string) *models.User {
	// Decode Basic auth header
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	// Split into username:password
	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return nil
	}

	password := credentials[1]

	// For Git operations, the "password" is typically a Personal Access Token
	// Try authenticating the password as a PAT
	user, err := m.authService.AuthenticateToken(ctx, password)
	if err == nil && user != nil {
		m.log.Debug("User authenticated via Basic Auth PAT",
			logger.String("user_id", user.ID.String()),
		)
		return user
	}

	// Also try as a session token (for OIDC users)
	user, err = m.authService.AuthenticateSession(ctx, password)
	if err == nil && user != nil {
		m.log.Debug("User authenticated via Basic Auth session token",
			logger.String("user_id", user.ID.String()),
		)
		return user
	}

	m.log.Debug("Basic auth failed - no valid credentials")
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
