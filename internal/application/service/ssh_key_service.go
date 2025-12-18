package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/repository"
	apperrors "github.com/bravo68web/githut/pkg/errors"
)

// SSHKeyService handles SSH key operations
type SSHKeyService struct {
	sshKeyRepo repository.SSHKeyRepository
	userRepo   repository.UserRepository
}

// NewSSHKeyService creates a new SSHKeyService instance
func NewSSHKeyService(
	sshKeyRepo repository.SSHKeyRepository,
	userRepo repository.UserRepository,
) *SSHKeyService {
	return &SSHKeyService{
		sshKeyRepo: sshKeyRepo,
		userRepo:   userRepo,
	}
}

// AddSSHKeyRequest represents a request to add an SSH key
type AddSSHKeyRequest struct {
	UserID    uuid.UUID
	Title     string
	PublicKey string
}

// AddSSHKeyResponse represents the response after adding an SSH key
type AddSSHKeyResponse struct {
	Key *models.SSHKey
}

// AddSSHKey adds a new SSH public key for a user
func (s *SSHKeyService) AddSSHKey(ctx context.Context, req AddSSHKeyRequest) (*AddSSHKeyResponse, error) {
	// Validate the public key format
	parsedKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		return nil, apperrors.BadRequest("invalid ssh key format", apperrors.ErrInvalidSSHKey)
	}

	// Generate fingerprint (SHA256 format like OpenSSH)
	fingerprint := generateFingerprint(parsedKey)

	// Check if key already exists
	exists, err := s.sshKeyRepo.ExistsByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to check ssh key existence: %w", err)
	}
	if exists {
		return nil, apperrors.Conflict("ssh key already exists", apperrors.ErrSSHKeyExists)
	}

	// Determine key type
	keyType := parsedKey.Type()

	// Use comment from key if title not provided
	title := req.Title
	if title == "" && comment != "" {
		title = comment
	}
	if title == "" {
		title = fmt.Sprintf("%s key", keyType)
	}

	// Create the SSH key record
	sshKey := &models.SSHKey{
		UserID:      req.UserID,
		Title:       title,
		PublicKey:   strings.TrimSpace(req.PublicKey),
		Fingerprint: fingerprint,
		KeyType:     keyType,
	}

	if err := s.sshKeyRepo.Create(ctx, sshKey); err != nil {
		return nil, fmt.Errorf("failed to create ssh key: %w", err)
	}

	return &AddSSHKeyResponse{Key: sshKey}, nil
}

// ListSSHKeys returns all SSH keys for a user
func (s *SSHKeyService) ListSSHKeys(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error) {
	keys, err := s.sshKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list ssh keys: %w", err)
	}
	return keys, nil
}

// GetSSHKey returns a specific SSH key by ID
func (s *SSHKeyService) GetSSHKey(ctx context.Context, keyID uuid.UUID) (*models.SSHKey, error) {
	key, err := s.sshKeyRepo.FindByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSSHKey removes an SSH key
func (s *SSHKeyService) DeleteSSHKey(ctx context.Context, userID, keyID uuid.UUID) error {
	// Get the key first to verify ownership
	key, err := s.sshKeyRepo.FindByID(ctx, keyID)
	if err != nil {
		return err
	}

	// Verify the key belongs to the user
	if key.UserID != userID {
		return apperrors.Forbidden("you do not own this ssh key", apperrors.ErrForbidden)
	}

	return s.sshKeyRepo.Delete(ctx, keyID)
}

// DeleteSSHKeyAdmin removes an SSH key without ownership check (for admins)
func (s *SSHKeyService) DeleteSSHKeyAdmin(ctx context.Context, keyID uuid.UUID) error {
	return s.sshKeyRepo.Delete(ctx, keyID)
}

// GetSSHKeyByFingerprint returns an SSH key by its fingerprint
func (s *SSHKeyService) GetSSHKeyByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error) {
	return s.sshKeyRepo.FindByFingerprint(ctx, fingerprint)
}

// CountUserSSHKeys returns the number of SSH keys for a user
func (s *SSHKeyService) CountUserSSHKeys(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.sshKeyRepo.CountByUserID(ctx, userID)
}

// generateFingerprint generates a SHA256 fingerprint for an SSH public key
// Returns the fingerprint in the format "SHA256:base64encodedHash"
func generateFingerprint(pubKey ssh.PublicKey) string {
	hash := sha256.Sum256(pubKey.Marshal())
	encoded := base64.RawStdEncoding.EncodeToString(hash[:])
	return "SHA256:" + encoded
}

// ValidateSSHKey validates an SSH public key string without storing it
func (s *SSHKeyService) ValidateSSHKey(publicKey string) (keyType string, fingerprint string, err error) {
	parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", "", apperrors.BadRequest("invalid ssh key format", apperrors.ErrInvalidSSHKey)
	}

	return parsedKey.Type(), generateFingerprint(parsedKey), nil
}
