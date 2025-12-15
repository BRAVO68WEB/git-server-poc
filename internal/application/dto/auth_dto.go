package dto

import (
	"time"

	"github.com/google/uuid"
)

// LoginRequest represents a user login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response with token
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents basic user information in responses
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsAdmin  bool      `json:"is_admin"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	User    UserInfo `json:"user"`
	Message string   `json:"message"`
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

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
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

// RefreshTokenRequest represents a request to refresh a token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents a response with new tokens
type RefreshTokenResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RevokeTokenRequest represents a request to revoke a token
type RevokeTokenRequest struct {
	TokenID uuid.UUID `json:"token_id" binding:"required"`
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
