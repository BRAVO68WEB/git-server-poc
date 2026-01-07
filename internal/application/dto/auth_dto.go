package dto

import (
	"time"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/google/uuid"
)

// UserInfo represents basic user information in responses
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsAdmin  bool      `json:"is_admin"`
}

// OIDCCallbackResponse represents the response after successful OIDC authentication
type OIDCCallbackResponse struct {
	Token   string   `json:"token"`
	User    UserInfo `json:"user"`
	Message string   `json:"message"`
}

// OIDCConfigResponse represents the OIDC configuration status
type OIDCConfigResponse struct {
	OIDCEnabled     bool `json:"oidc_enabled"`
	OIDCInitialized bool `json:"oidc_initialized"`
}

// CreateTokenRequest represents the request body for creating a token
type CreateTokenRequest struct {
	Name      string   `json:"name" binding:"required,min=1,max=255"`
	Scopes    []string `json:"scopes"`     // optional, empty = all repos
	ExpiresIn *int     `json:"expires_in"` // optional, days until expiration
}

// TokenResponse represents a token in API responses
type TokenResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token,omitempty"` // Only returned on creation
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenInfo represents access token information (without the actual token)
type TokenInfo struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListTokensResponse represents a list of user tokens
type ListTokensResponse struct {
	Tokens []TokenInfo `json:"tokens"`
	Total  int         `json:"total"`
}

func TokenInfoFromModels(tokens []*models.Token) []TokenInfo {
	var tokenInfo []TokenInfo

	for _, t := range tokens {
		tokenInfo = append(tokenInfo, TokenInfo{
			ID:        t.ID,
			Name:      t.Name,
			Scopes:    []string(t.Scope),
			ExpiresAt: t.ExpiresAt,
			LastUsed:  t.LastUsed,
			CreatedAt: t.CreatedAt,
		})
	}

	return tokenInfo
}

// AddSSHKeyRequest represents a request to add an SSH key
type AddSSHKeyRequest struct {
	Title string `json:"title" binding:"required"`
	Key   string `json:"key" binding:"required"`
}

// SSHKeyInfo represents SSH key information
type SSHKeyInfo struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddSSHKeyResponse represents a response after adding an SSH key
type AddSSHKeyResponse struct {
	Key     SSHKeyInfo `json:"key"`
	Message string     `json:"message"`
}

// ListSSHKeysResponse represents a list of user SSH keys
type ListSSHKeysResponse struct {
	Keys  []SSHKeyInfo `json:"keys"`
	Total int          `json:"total"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	Email *string `json:"email,omitempty" binding:"omitempty,email"`
}

// UpdateProfileResponse represents a response after updating profile
type UpdateProfileResponse struct {
	User    UserInfo `json:"user"`
	Message string   `json:"message"`
}

// AuthenticatedUser represents the authenticated user context
type AuthenticatedUser struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsAdmin  bool      `json:"is_admin"`
	Scopes   []string  `json:"scopes,omitempty"` // For token-based auth
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ValidateTokenResponse represents the response for token validation
type ValidateTokenResponse struct {
	Valid bool       `json:"valid"`
	User  *UserInfo  `json:"user,omitempty"`
	Token *TokenInfo `json:"token,omitempty"`
}

// UpdateUserRequest represents a request to update user information
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=1,max=255"`
}
