package dto

import (
	"time"

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

// CreateTokenRequest represents a request to create an access token
type CreateTokenRequest struct {
	Name      string   `json:"name" binding:"required"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at"` // RFC3339 format
}

// CreateTokenResponse represents a response after creating a token
type CreateTokenResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token"` // Only returned once on creation
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenInfo represents access token information (without the actual token)
type TokenInfo struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListTokensResponse represents a list of user tokens
type ListTokensResponse struct {
	Tokens []TokenInfo `json:"tokens"`
	Total  int         `json:"total"`
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
