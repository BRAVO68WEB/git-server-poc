package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/models"
	domainservice "github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
	"github.com/bravo68web/stasis/pkg/logger"
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
	log         *logger.Logger
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
		log:         logger.Get().WithFields(logger.Component("auth-handler")),
	}
}

// OIDCLogin handles GET /api/v1/auth/oidc/login
// Initiates the OIDC authentication flow by redirecting to the identity provider
func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	h.log.Debug("OIDC login initiated",
		logger.ClientIP(c.ClientIP()),
	)

	if h.oidcService == nil || !h.oidcService.IsEnabled() {
		h.log.Warn("OIDC login attempted but OIDC is not enabled")
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": "OIDC authentication is not enabled",
		})
		return
	}

	if !h.oidcService.IsInitialized() {
		h.log.Warn("OIDC login attempted but OIDC service is not initialized")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "service_unavailable",
			"message": "OIDC service is not initialized",
		})
		return
	}

	// Generate authorization URL with state
	authURL, state, err := h.oidcService.GenerateAuthURL()
	if err != nil {
		h.log.Error("Failed to generate OIDC authorization URL",
			logger.Error(err),
		)
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

	h.log.Info("Redirecting user to OIDC provider",
		logger.ClientIP(c.ClientIP()),
	)

	// Redirect to the identity provider
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OIDCCallback handles GET /api/v1/auth/oidc/callback
// Processes the callback from the identity provider after user authentication
func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	h.log.Debug("OIDC callback received",
		logger.ClientIP(c.ClientIP()),
	)

	if h.oidcService == nil || !h.oidcService.IsEnabled() {
		h.log.Warn("OIDC callback received but OIDC is not enabled")
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": "OIDC authentication is not enabled",
		})
		return
	}

	// Check for errors from the identity provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		h.log.Warn("OIDC provider returned error",
			logger.String("error", errParam),
			logger.String("description", errDesc),
		)
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
		h.log.Warn("OIDC callback missing authorization code")
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
		h.log.Warn("OIDC callback missing or expired state cookie",
			logger.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Missing or expired state cookie",
		})
		return
	}

	// Clear the state cookie
	c.SetCookie(oidcStateCookie, "", -1, "/", "", false, true)

	h.log.Debug("Processing OIDC callback")

	// Handle the callback - this exchanges the code for tokens and creates/updates the user
	user, sessionToken, err := h.oidcService.HandleCallback(c.Request.Context(), code, state, expectedState)
	if err != nil {
		h.log.Error("OIDC callback handling failed",
			logger.Error(err),
		)
		h.handleError(c, err)
		return
	}

	h.log.Info("OIDC authentication successful",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
		logger.String("email", user.Email),
	)

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
			h.log.Error("Failed to marshal user info for redirect",
				logger.Error(err),
			)
			h.handleError(c, err)
			return
		}
		userBase64 := base64.URLEncoding.EncodeToString(userJSON)

		// Redirect to frontend callback page with token and user in URL fragment
		// Using fragment (#) so the token doesn't appear in server logs
		redirectURL := fmt.Sprintf("%s/auth/callback#token=%s&user=%s",
			frontendURL, sessionToken, userBase64)

		h.log.Debug("Redirecting user to frontend",
			logger.String("frontend_url", frontendURL),
			logger.String("user_id", user.ID.String()),
		)

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
	h.log.Debug("OIDC logout initiated",
		logger.ClientIP(c.ClientIP()),
	)

	// Get the post-logout redirect URI from query parameter
	postLogoutRedirectURI := c.Query("redirect_uri")

	// If OIDC is enabled and supports logout, get the logout URL
	var logoutURL string
	if h.oidcService != nil && h.oidcService.IsEnabled() && h.oidcService.IsInitialized() {
		var err error
		logoutURL, err = h.oidcService.GetLogoutURL("", postLogoutRedirectURI)
		if err != nil {
			h.log.Warn("Failed to get OIDC logout URL",
				logger.Error(err),
			)
			// Log error but continue - logout from our side should still work
		}
	}

	// If we have a provider logout URL, redirect to it
	if logoutURL != "" {
		h.log.Info("User logged out with provider logout URL")
		c.JSON(http.StatusOK, gin.H{
			"message":    "Logged out successfully",
			"logout_url": logoutURL,
		})
		return
	}

	h.log.Info("User logged out successfully")
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
		h.log.Debug("GetCurrentUser called without authentication")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	h.log.Debug("Returning current user info",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
	)

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

	h.log.Debug("Returning OIDC config",
		logger.Bool("enabled", enabled),
		logger.Bool("initialized", initialized),
	)

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
