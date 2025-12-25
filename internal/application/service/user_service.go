package service

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/google/uuid"
)

// UserService handles user-related business logic
type UserService struct {
	userRepo repository.UserRepository
	log      *logger.Logger
}

// NewUserService creates a new UserService instance
func NewUserService(
	userRepo repository.UserRepository,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		log:      logger.Get().WithFields(logger.Component("user-service")),
	}
}

// CreateUserRequest represents a request to create a new user (for OIDC flow)
type CreateUserRequest struct {
	Username    string
	Email       string
	OIDCSubject string
	OIDCIssuer  string
	IsAdmin     bool
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email   *string
	IsAdmin *bool
}

// CreateUser creates a new user (typically from OIDC flow)
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	s.log.Info("Creating new user",
		logger.String("username", req.Username),
		logger.String("email", req.Email),
		logger.Bool("is_admin", req.IsAdmin),
	)

	// Validate username
	if err := s.validateUsername(req.Username); err != nil {
		s.log.Warn("Username validation failed",
			logger.String("username", req.Username),
			logger.Error(err),
		)
		return nil, err
	}

	// Validate email
	if err := s.validateEmail(req.Email); err != nil {
		s.log.Warn("Email validation failed",
			logger.String("email", req.Email),
			logger.Error(err),
		)
		return nil, err
	}

	// Check if username already exists
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		s.log.Error("Failed to check username existence",
			logger.Error(err),
			logger.String("username", req.Username),
		)
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		s.log.Warn("Username already taken",
			logger.String("username", req.Username),
		)
		return nil, apperrors.Conflict("username already taken", apperrors.ErrUserExists)
	}

	// Check if email already exists
	exists, err = s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		s.log.Error("Failed to check email existence",
			logger.Error(err),
			logger.String("email", req.Email),
		)
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		s.log.Warn("Email already registered",
			logger.String("email", req.Email),
		)
		return nil, apperrors.Conflict("email already registered", apperrors.ErrUserExists)
	}

	// Create user
	user := &models.User{
		Username:    req.Username,
		Email:       strings.ToLower(req.Email),
		OIDCSubject: req.OIDCSubject,
		OIDCIssuer:  req.OIDCIssuer,
		IsAdmin:     req.IsAdmin,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error("Failed to create user in database",
			logger.Error(err),
			logger.String("username", req.Username),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.log.Info("User created successfully",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
		logger.String("email", user.Email),
	)

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	s.log.Debug("Getting user by ID",
		logger.String("user_id", id.String()),
	)
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		s.log.Debug("User not found",
			logger.String("user_id", id.String()),
			logger.Error(err),
		)
		return nil, err
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	s.log.Debug("Getting user by username",
		logger.String("username", username),
	)
	return s.userRepo.FindByUsername(ctx, username)
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	s.log.Debug("Getting user by email",
		logger.String("email", email),
	)
	return s.userRepo.FindByEmail(ctx, strings.ToLower(email))
}

// GetUserByOIDCSubject retrieves a user by OIDC subject and issuer
func (s *UserService) GetUserByOIDCSubject(ctx context.Context, subject, issuer string) (*models.User, error) {
	s.log.Debug("Getting user by OIDC subject",
		logger.String("subject", subject),
		logger.String("issuer", issuer),
	)
	return s.userRepo.FindByOIDCSubject(ctx, subject, issuer)
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*models.User, error) {
	s.log.Info("Updating user",
		logger.String("user_id", id.String()),
	)

	// Get existing user
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to find user for update",
			logger.Error(err),
			logger.String("user_id", id.String()),
		)
		return nil, err
	}

	// Update email if provided
	if req.Email != nil {
		if err := s.validateEmail(*req.Email); err != nil {
			s.log.Warn("Email validation failed during update",
				logger.Error(err),
				logger.String("email", *req.Email),
			)
			return nil, err
		}

		normalizedEmail := strings.ToLower(*req.Email)
		if normalizedEmail != user.Email {
			// Check if email is already taken
			exists, err := s.userRepo.ExistsByEmail(ctx, normalizedEmail)
			if err != nil {
				s.log.Error("Failed to check email existence during update",
					logger.Error(err),
				)
				return nil, fmt.Errorf("failed to check email: %w", err)
			}
			if exists {
				s.log.Warn("Email already registered by another user",
					logger.String("email", normalizedEmail),
				)
				return nil, apperrors.Conflict("email already registered", apperrors.ErrUserExists)
			}
			user.Email = normalizedEmail
			s.log.Debug("Updating user email",
				logger.String("user_id", id.String()),
				logger.String("new_email", normalizedEmail),
			)
		}
	}

	// Update admin status if provided
	if req.IsAdmin != nil {
		s.log.Debug("Updating user admin status",
			logger.String("user_id", id.String()),
			logger.Bool("is_admin", *req.IsAdmin),
		)
		user.IsAdmin = *req.IsAdmin
	}

	// Save updates
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update user in database",
			logger.Error(err),
			logger.String("user_id", id.String()),
		)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.log.Info("User updated successfully",
		logger.String("user_id", user.ID.String()),
		logger.String("username", user.Username),
	)

	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	s.log.Info("Deleting user",
		logger.String("user_id", id.String()),
	)

	// Check if user exists
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to find user for deletion",
			logger.Error(err),
			logger.String("user_id", id.String()),
		)
		return err
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, id); err != nil {
		s.log.Error("Failed to delete user from database",
			logger.Error(err),
			logger.String("user_id", id.String()),
		)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.log.Info("User deleted successfully",
		logger.String("user_id", id.String()),
		logger.String("username", user.Username),
	)

	return nil
}

// ListUsers lists users with pagination
func (s *UserService) ListUsers(ctx context.Context, page, perPage int) ([]*models.User, int64, error) {
	s.log.Debug("Listing users",
		logger.Int("page", page),
		logger.Int("per_page", perPage),
	)

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	users, err := s.userRepo.List(ctx, perPage, offset)
	if err != nil {
		s.log.Error("Failed to list users",
			logger.Error(err),
		)
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	total, err := s.userRepo.Count(ctx)
	if err != nil {
		s.log.Error("Failed to count users",
			logger.Error(err),
		)
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	s.log.Debug("Users listed successfully",
		logger.Int("count", len(users)),
		logger.Int64("total", total),
	)

	return users, total, nil
}

// validateUsername validates a username
func (s *UserService) validateUsername(username string) error {
	if username == "" {
		return apperrors.ValidationError("username", "username is required")
	}

	if len(username) < 3 {
		return apperrors.ValidationError("username", "username must be at least 3 characters")
	}

	if len(username) > 50 {
		return apperrors.ValidationError("username", "username must be 50 characters or less")
	}

	// Username must start with a letter and contain only alphanumeric, underscore, or hyphen
	usernameRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	if !usernameRegex.MatchString(username) {
		return apperrors.ValidationError("username", "username must start with a letter and contain only letters, numbers, underscores, or hyphens")
	}

	// Check for reserved usernames
	reservedUsernames := []string{"admin", "root", "system", "api", "git", "www", "mail", "ftp", "ssh"}
	lowerUsername := strings.ToLower(username)
	if ok := slices.Contains(reservedUsernames, lowerUsername); ok {
		return apperrors.ValidationError("username", "username is reserved")
	}

	return nil
}

// validateEmail validates an email address
func (s *UserService) validateEmail(email string) error {
	if email == "" {
		return apperrors.ValidationError("email", "email is required")
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return apperrors.ValidationError("email", "invalid email format")
	}

	return nil
}

// UserExists checks if a user exists by username
func (s *UserService) UserExists(ctx context.Context, username string) (bool, error) {
	return s.userRepo.ExistsByUsername(ctx, username)
}

// EmailExists checks if an email is already registered
func (s *UserService) EmailExists(ctx context.Context, email string) (bool, error) {
	return s.userRepo.ExistsByEmail(ctx, strings.ToLower(email))
}

// CountUsers returns the total number of users
func (s *UserService) CountUsers(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

// SetAdminStatus sets the admin status of a user
func (s *UserService) SetAdminStatus(ctx context.Context, userID uuid.UUID, isAdmin bool) error {
	s.log.Info("Setting user admin status",
		logger.String("user_id", userID.String()),
		logger.Bool("is_admin", isAdmin),
	)

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.log.Error("Failed to find user for admin status update",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return err
	}

	user.IsAdmin = isAdmin
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update admin status",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return fmt.Errorf("failed to update admin status: %w", err)
	}

	s.log.Info("User admin status updated successfully",
		logger.String("user_id", userID.String()),
		logger.String("username", user.Username),
		logger.Bool("is_admin", isAdmin),
	)

	return nil
}
