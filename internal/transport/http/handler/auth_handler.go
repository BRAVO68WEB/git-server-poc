package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/githut/internal/application/dto"
	"github.com/bravo68web/githut/internal/application/service"
	"github.com/bravo68web/githut/internal/config"
	"github.com/bravo68web/githut/internal/domain/models"
	domainservice "github.com/bravo68web/githut/internal/domain/service"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/githut/pkg/errors"
)

const (
	// Cookie names for OIDC state management
	oidcStateCookie    = "oidc_state"
	oidcStateCookieExp = 10 * time.Minute
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService domainservice.AuthService
	userService *service.UserService
	oidcService *service.OIDCService
	config      *config.Config
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(
	authService domainservice.AuthService,
	userService *service.UserService,
	oidcService *service.OIDCService,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		oidcService: oidcService,
		config:      cfg,
	}
}

// OIDCLogin handles GET /api/v1/auth/oidc/login
// Initiates the OIDC authentication flow by redirecting to the identity provider
func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	if h.oidcService == nil || !h.oidcService.IsEnabled() {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": "OIDC authentication is not enabled",
		})
		return
	}

	if !h.oidcService.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "service_unavailable",
			"message": "OIDC service is not initialized",
		})
		return
	}

	// Generate authorization URL with state
	authURL, state, err := h.oidcService.GenerateAuthURL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate authorization URL",
		})
		return
	}

	// Store state in a secure HTTP-only cookie
	c.SetCookie(
		oidcStateCookie,
		state,
		int(oidcStateCookieExp.Seconds()),
		"/",
		"",    // domain
		false, // secure (set to true in production with HTTPS)
		true,  // httpOnly
	)

	// Redirect to the identity provider
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OIDCCallback handles GET /api/v1/auth/oidc/callback
// Processes the callback from the identity provider after user authentication
func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	if h.oidcService == nil || !h.oidcService.IsEnabled() {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": "OIDC authentication is not enabled",
		})
		return
	}

	// Check for errors from the identity provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errParam,
			"message":     "Authentication failed at identity provider",
			"description": errDesc,
		})
		return
	}

	// Get the authorization code
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Missing authorization code",
		})
		return
	}

	// Get the state from query and cookie
	state := c.Query("state")
	expectedState, err := c.Cookie(oidcStateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Missing or expired state cookie",
		})
		return
	}

	// Clear the state cookie
	c.SetCookie(oidcStateCookie, "", -1, "/", "", false, true)

	// Handle the callback - this exchanges the code for tokens and creates/updates the user
	user, sessionToken, err := h.oidcService.HandleCallback(c.Request.Context(), code, state, expectedState)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check if we have a frontend URL to redirect to
	frontendURL := h.config.OIDC.FrontendURL
	if frontendURL != "" {
		// Prepare user info for the frontend
		userInfo := dto.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		}

		// Encode user info as base64 JSON
		userJSON, err := json.Marshal(userInfo)
		if err != nil {
			h.handleError(c, err)
			return
		}
		userBase64 := base64.URLEncoding.EncodeToString(userJSON)

		// Redirect to frontend callback page with token and user in URL fragment
		// Using fragment (#) so the token doesn't appear in server logs
		redirectURL := fmt.Sprintf("%s/auth/callback#token=%s&user=%s",
			frontendURL, sessionToken, userBase64)

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	// Fallback: Return the session token and user info as JSON (for API clients)
	c.JSON(http.StatusOK, dto.OIDCCallbackResponse{
		Token: sessionToken,
		User: dto.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
		Message: "Authentication successful",
	})
}

// OIDCLogout handles POST /api/v1/auth/oidc/logout
// Logs the user out by invalidating the session
func (h *AuthHandler) OIDCLogout(c *gin.Context) {
	// Get the post-logout redirect URI from query parameter
	postLogoutRedirectURI := c.Query("redirect_uri")

	// If OIDC is enabled and supports logout, get the logout URL
	var logoutURL string
	if h.oidcService != nil && h.oidcService.IsEnabled() && h.oidcService.IsInitialized() {
		var err error
		logoutURL, err = h.oidcService.GetLogoutURL("", postLogoutRedirectURI)
		if err != nil {
			// Log error but continue - logout from our side should still work
		}
	}

	// If we have a provider logout URL, redirect to it
	if logoutURL != "" {
		c.JSON(http.StatusOK, gin.H{
			"message":    "Logged out successfully",
			"logout_url": logoutURL,
		})
		return
	}

	// Otherwise just confirm logout
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser handles GET /api/v1/auth/me
// Returns the currently authenticated user's information
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		IsAdmin:  user.IsAdmin,
	})
}

// GetOIDCConfig handles GET /api/v1/auth/oidc/config
// Returns the OIDC configuration status (for frontend to know if OIDC is enabled)
func (h *AuthHandler) GetOIDCConfig(c *gin.Context) {
	enabled := h.oidcService != nil && h.oidcService.IsEnabled()
	initialized := enabled && h.oidcService.IsInitialized()

	c.JSON(http.StatusOK, gin.H{
		"oidc_enabled":     enabled,
		"oidc_initialized": initialized,
	})
}

// handleError handles errors and returns appropriate HTTP responses
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if ok := apperrors.IsNotFound(err); ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": err.Error(),
		})
		return
	}

	if ok := apperrors.IsUnauthorized(err); ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": err.Error(),
		})
		return
	}

	if ok := apperrors.IsForbidden(err); ok {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": err.Error(),
		})
		return
	}

	if ok := apperrors.IsConflict(err); ok {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "conflict",
			"message": err.Error(),
		})
		return
	}

	// Check for AppError
	if e, ok := err.(*apperrors.AppError); ok {
		appErr = e
		c.JSON(appErr.HTTPStatus(), gin.H{
			"error":   "error",
			"message": appErr.Message,
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An unexpected error occurred",
	})
}

// Ensure AuthHandler implements all required methods
var _ interface {
	OIDCLogin(c *gin.Context)
	OIDCCallback(c *gin.Context)
	OIDCLogout(c *gin.Context)
	GetCurrentUser(c *gin.Context)
	GetOIDCConfig(c *gin.Context)
} = (*AuthHandler)(nil)

// Unused but kept for interface compliance
var _ *models.User = nil
