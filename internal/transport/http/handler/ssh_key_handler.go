package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bravo68web/githut/internal/application/dto"
	"github.com/bravo68web/githut/internal/application/service"
	"github.com/bravo68web/githut/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/githut/pkg/errors"
)

// SSHKeyHandler handles SSH key-related HTTP requests
type SSHKeyHandler struct {
	sshKeyService *service.SSHKeyService
}

// NewSSHKeyHandler creates a new SSHKeyHandler instance
func NewSSHKeyHandler(sshKeyService *service.SSHKeyService) *SSHKeyHandler {
	return &SSHKeyHandler{
		sshKeyService: sshKeyService,
	}
}

// AddSSHKey handles POST /api/v1/user/ssh-keys
func (h *SSHKeyHandler) AddSSHKey(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var req dto.AddSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.sshKeyService.AddSSHKey(c.Request.Context(), service.AddSSHKeyRequest{
		UserID:    user.ID,
		Title:     req.Title,
		PublicKey: req.Key,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AddSSHKeyResponse{
		Key: dto.SSHKeyInfo{
			ID:          resp.Key.ID,
			Title:       resp.Key.Title,
			Fingerprint: resp.Key.Fingerprint,
			CreatedAt:   resp.Key.CreatedAt,
		},
		Message: "SSH key added successfully",
	})
}

// ListSSHKeys handles GET /api/v1/user/ssh-keys
func (h *SSHKeyHandler) ListSSHKeys(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	keys, err := h.sshKeyService.ListSSHKeys(c.Request.Context(), user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	keyInfos := make([]dto.SSHKeyInfo, len(keys))
	for i, key := range keys {
		keyInfos[i] = dto.SSHKeyInfo{
			ID:          key.ID,
			Title:       key.Title,
			Fingerprint: key.Fingerprint,
			CreatedAt:   key.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, dto.ListSSHKeysResponse{
		Keys:  keyInfos,
		Total: len(keys),
	})
}

// GetSSHKey handles GET /api/v1/user/ssh-keys/:id
func (h *SSHKeyHandler) GetSSHKey(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid SSH key ID",
		})
		return
	}

	key, err := h.sshKeyService.GetSSHKey(c.Request.Context(), keyID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify ownership
	if key.UserID != user.ID && !user.IsAdmin {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "SSH key not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.SSHKeyInfo{
		ID:          key.ID,
		Title:       key.Title,
		Fingerprint: key.Fingerprint,
		CreatedAt:   key.CreatedAt,
	})
}

// DeleteSSHKey handles DELETE /api/v1/user/ssh-keys/:id
func (h *SSHKeyHandler) DeleteSSHKey(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid SSH key ID",
		})
		return
	}

	// Admin can delete any key
	if user.IsAdmin {
		if err := h.sshKeyService.DeleteSSHKeyAdmin(c.Request.Context(), keyID); err != nil {
			h.handleError(c, err)
			return
		}
	} else {
		if err := h.sshKeyService.DeleteSSHKey(c.Request.Context(), user.ID, keyID); err != nil {
			h.handleError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "SSH key deleted successfully",
	})
}

// handleError handles errors and returns appropriate HTTP responses
func (h *SSHKeyHandler) handleError(c *gin.Context, err error) {
	if apperrors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": err.Error(),
		})
		return
	}

	if apperrors.IsUnauthorized(err) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
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

	if apperrors.IsConflict(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "conflict",
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

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An unexpected error occurred",
	})
}
