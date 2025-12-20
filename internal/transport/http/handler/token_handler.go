package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
)

// TokenHandler handles personal access token HTTP requests
type TokenHandler struct {
	tokenService *service.TokenService
}

// NewTokenHandler creates a new TokenHandler instance
func NewTokenHandler(tokenService *service.TokenService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
	}
}

// CreateToken handles POST /api/v1/tokens
func (h *TokenHandler) CreateToken(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var req dto.CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Calculate expiration if provided
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &t
	}

	resp, err := h.tokenService.CreateToken(c.Request.Context(), service.CreateTokenRequest{
		UserID:    user.ID,
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return the raw token (only shown once)
	c.JSON(http.StatusCreated, gin.H{
		"token":   resp.RawToken,
		"message": "Token created successfully",
		"token_info": dto.TokenInfo{
			ID:        resp.Token.ID,
			Name:      resp.Token.Name,
			Scopes:    resp.Token.Scope,
			ExpiresAt: resp.Token.ExpiresAt,
			CreatedAt: resp.Token.CreatedAt,
		},
	})
}

// ListTokens handles GET /api/v1/tokens
func (h *TokenHandler) ListTokens(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	tokens, err := h.tokenService.ListTokens(c.Request.Context(), user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	tokenInfoList := dto.TokenInfoFromModels(tokens)

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokenInfoList,
		"total":  len(tokenInfoList),
	})
}

// DeleteToken handles DELETE /api/v1/tokens/:id
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid token ID",
		})
		return
	}

	if err := h.tokenService.DeleteToken(c.Request.Context(), user.ID, tokenID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token deleted successfully",
	})
}

// handleError handles errors and returns appropriate HTTP responses
func (h *TokenHandler) handleError(c *gin.Context, err error) {
	if apperrors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": err.Error(),
		})
		return
	}
	if apperrors.IsBadRequest(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": err.Error(),
		})
		return
	}
	if apperrors.IsConflict(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "conflict",
			"message": err.Error(),
		})
		return
	}
	if apperrors.IsForbidden(err) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An unexpected error occurred",
	})
}
