package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/bravo68web/githut/internal/config"
	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	apperrors "github.com/bravo68web/githut/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// OIDCService handles OpenID Connect authentication
type OIDCService struct {
	config      *config.OIDCConfig
	provider    *oidc.Provider
	oauth2Cfg   *oauth2.Config
	verifier    *oidc.IDTokenVerifier
	userRepo    repository.UserRepository
	initialized bool
}

// OIDCClaims represents the claims from an OIDC ID token
type OIDCClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Username      string `json:"preferred_username"`
	Picture       string `json:"picture"`
}

// SessionClaims represents the claims in the session JWT
type SessionClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
}

// NewOIDCService creates a new OIDCService instance
func NewOIDCService(cfg *config.OIDCConfig, userRepo repository.UserRepository) *OIDCService {
	return &OIDCService{
		config:      cfg,
		userRepo:    userRepo,
		initialized: false,
	}
}

// Initialize sets up the OIDC provider connection
// This should be called after the service is created and before any other methods
func (s *OIDCService) Initialize(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}

	provider, err := oidc.NewProvider(ctx, s.config.IssuerURL)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	s.provider = provider
	s.oauth2Cfg = &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		RedirectURL:  s.config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       s.config.Scopes,
	}

	s.verifier = provider.Verifier(&oidc.Config{ClientID: s.config.ClientID})
	s.initialized = true

	return nil
}

// IsEnabled returns whether OIDC is enabled
func (s *OIDCService) IsEnabled() bool {
	return s.config.Enabled
}

// IsInitialized returns whether the service has been initialized
func (s *OIDCService) IsInitialized() bool {
	return s.initialized
}

// GenerateAuthURL generates the authorization URL for OIDC login
// Returns the URL and the state parameter (which should be stored in session/cookie)
func (s *OIDCService) GenerateAuthURL() (string, string, error) {
	if !s.initialized {
		return "", "", fmt.Errorf("OIDC service not initialized")
	}

	state, err := generateRandomState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	url := s.oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
	return url, state, nil
}

// HandleCallback processes the OIDC callback and returns the authenticated user
func (s *OIDCService) HandleCallback(ctx context.Context, code, state, expectedState string) (*models.User, string, error) {
	if !s.initialized {
		return nil, "", fmt.Errorf("OIDC service not initialized")
	}

	// Verify state
	if state != expectedState {
		return nil, "", apperrors.Unauthorized("invalid state parameter", apperrors.ErrInvalidCredentials)
	}

	// Exchange code for tokens
	oauth2Token, err := s.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, "", fmt.Errorf("no id_token in token response")
	}

	// Verify ID token
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to verify id_token: %w", err)
	}

	// Extract claims
	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, idToken.Issuer, claims)
	if err != nil {
		return nil, "", err
	}

	// Generate session JWT
	sessionToken, err := s.GenerateSessionToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, sessionToken, nil
}

// findOrCreateUser finds an existing user or creates a new one based on OIDC claims
func (s *OIDCService) findOrCreateUser(ctx context.Context, issuer string, claims OIDCClaims) (*models.User, error) {
	// Try to find user by OIDC subject
	user, err := s.userRepo.FindByOIDCSubject(ctx, claims.Subject, issuer)
	if err == nil {
		// User found, update email if changed
		if user.Email != claims.Email && claims.Email != "" {
			user.Email = strings.ToLower(claims.Email)
			if err := s.userRepo.Update(ctx, user); err != nil {
				// Log but don't fail - email update is not critical
				fmt.Printf("failed to update user email: %v\n", err)
			}
		}
		return user, nil
	}

	// User not found, check if it's a "not found" error
	if !apperrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Create new user
	username := s.generateUsername(claims)
	email := strings.ToLower(claims.Email)

	// Check if email already exists (user might have registered via another method before)
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		// User with this email exists, link OIDC to existing account
		existingUser.OIDCSubject = claims.Subject
		existingUser.OIDCIssuer = issuer
		if err := s.userRepo.Update(ctx, existingUser); err != nil {
			return nil, fmt.Errorf("failed to link OIDC to existing user: %w", err)
		}
		return existingUser, nil
	}

	// Create new user
	newUser := &models.User{
		Username:    username,
		Email:       email,
		OIDCSubject: claims.Subject,
		OIDCIssuer:  issuer,
		IsAdmin:     false,
	}

	// Ensure username is unique
	for i := 0; i < 10; i++ {
		exists, err := s.userRepo.ExistsByUsername(ctx, newUser.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if !exists {
			break
		}
		// Username exists, append random suffix
		newUser.Username = fmt.Sprintf("%s%d", username, time.Now().UnixNano()%1000)
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// generateUsername generates a username from OIDC claims
func (s *OIDCService) generateUsername(claims OIDCClaims) string {
	// Prefer preferred_username, then email prefix, then name
	if claims.Username != "" {
		return sanitizeUsername(claims.Username)
	}
	if claims.Email != "" {
		parts := strings.Split(claims.Email, "@")
		return sanitizeUsername(parts[0])
	}
	if claims.Name != "" {
		return sanitizeUsername(strings.ReplaceAll(claims.Name, " ", ""))
	}
	return fmt.Sprintf("user%d", time.Now().UnixNano()%100000)
}

// sanitizeUsername removes invalid characters from username
func sanitizeUsername(s string) string {
	// Keep only alphanumeric, underscore, hyphen
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	username := result.String()

	// Ensure it starts with a letter
	if len(username) > 0 && !((username[0] >= 'a' && username[0] <= 'z') || (username[0] >= 'A' && username[0] <= 'Z')) {
		username = "u" + username
	}

	// Ensure minimum length
	if len(username) < 3 {
		username = username + "user"
	}

	// Truncate if too long
	if len(username) > 50 {
		username = username[:50]
	}

	return strings.ToLower(username)
}

// GenerateSessionToken generates a JWT session token for the user
func (s *OIDCService) GenerateSessionToken(user *models.User) (string, error) {
	now := time.Now()
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "githut",
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)), // 24 hour expiry
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:   user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		IsAdmin:  user.IsAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign session token: %w", err)
	}

	return signedToken, nil
}

// ValidateSessionToken validates a session JWT and returns the claims
func (s *OIDCService) ValidateSessionToken(tokenString string) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, apperrors.Unauthorized("invalid session token", err)
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, apperrors.Unauthorized("invalid session token claims", nil)
	}

	return claims, nil
}

// GetLogoutURL returns the OIDC logout URL if supported by the provider
func (s *OIDCService) GetLogoutURL(idTokenHint, postLogoutRedirectURI string) (string, error) {
	if !s.initialized {
		return "", fmt.Errorf("OIDC service not initialized")
	}

	// Try to get the end_session_endpoint from provider metadata
	var providerClaims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}

	if err := s.provider.Claims(&providerClaims); err != nil {
		// Provider doesn't support discovery of logout endpoint
		return "", nil
	}

	if providerClaims.EndSessionEndpoint == "" {
		return "", nil
	}

	logoutURL := providerClaims.EndSessionEndpoint
	if postLogoutRedirectURI != "" {
		logoutURL = fmt.Sprintf("%s?post_logout_redirect_uri=%s", logoutURL, postLogoutRedirectURI)
	}

	return logoutURL, nil
}

// generateRandomState generates a random state string for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
