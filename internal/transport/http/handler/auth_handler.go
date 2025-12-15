package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/githut/internal/application/dto"
	"github.com/bravo68web/githut/internal/application/service"
	"github.com/bravo68web/githut/internal/domain/models"
	domainservice "github.com/bravo68web/githut/internal/domain/service"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/githut/pkg/errors"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService domainservice.AuthService
	userService *service.UserService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(
	authService domainservice.AuthService,
	userService *service.UserService,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Create user
	user, err := h.userService.CreateUser(c.Request.Context(), service.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsAdmin:  false,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.RegisterResponse{
		User: dto.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
		Message: "User registered successfully",
	})
}

// GetCurrentUser handles GET /api/auth/me
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

// ChangePassword handles POST /api/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Change password
	if err := h.userService.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
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
	Register(c *gin.Context)
	GetCurrentUser(c *gin.Context)
	ChangePassword(c *gin.Context)
} = (*AuthHandler)(nil)

// Unused but kept for interface compliance
var _ *models.User = nil
