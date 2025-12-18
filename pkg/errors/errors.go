package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard sentinel errors for common error cases
var (
	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrUnauthorized indicates the request lacks valid authentication
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the user doesn't have permission for this action
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidInput indicates the provided input is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrRepositoryExists indicates a repository with the same name already exists
	ErrRepositoryExists = errors.New("repository already exists")

	// ErrInternalServer indicates an internal server error occurred
	ErrInternalServer = errors.New("internal server error")

	// ErrUserExists indicates a user with the same username/email already exists
	ErrUserExists = errors.New("user already exists")

	// ErrInvalidCredentials indicates the provided credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenExpired indicates the access token has expired
	ErrTokenExpired = errors.New("token expired")

	// ErrSSHKeyExists indicates an SSH key with the same fingerprint already exists
	ErrSSHKeyExists = errors.New("ssh key already exists")

	// ErrInvalidSSHKey indicates the SSH key format is invalid
	ErrInvalidSSHKey = errors.New("invalid ssh key")

	// ErrBranchNotFound indicates the branch was not found
	ErrBranchNotFound = errors.New("branch not found")

	// ErrBranchExists indicates a branch with the same name already exists
	ErrBranchExists = errors.New("branch already exists")

	// ErrTagNotFound indicates the tag was not found
	ErrTagNotFound = errors.New("tag not found")

	// ErrTagExists indicates a tag with the same name already exists
	ErrTagExists = errors.New("tag already exists")

	// ErrDefaultBranch indicates an operation on the default branch is not allowed
	ErrDefaultBranch = errors.New("operation not allowed on default branch")

	// ErrStorageError indicates a storage operation failed
	ErrStorageError = errors.New("storage error")

	// ErrGitOperationFailed indicates a git operation failed
	ErrGitOperationFailed = errors.New("git operation failed")

	// ErrConfigError indicates a configuration error
	ErrConfigError = errors.New("configuration error")

	// ErrDatabaseError indicates a database operation failed
	ErrDatabaseError = errors.New("database error")
)

// ErrorCode represents HTTP-like error codes
type ErrorCode int

const (
	CodeBadRequest          ErrorCode = http.StatusBadRequest
	CodeUnauthorized        ErrorCode = http.StatusUnauthorized
	CodeForbidden           ErrorCode = http.StatusForbidden
	CodeNotFound            ErrorCode = http.StatusNotFound
	CodeConflict            ErrorCode = http.StatusConflict
	CodeInternalServerError ErrorCode = http.StatusInternalServerError
	CodeServiceUnavailable  ErrorCode = http.StatusServiceUnavailable
)

// AppError represents an application-level error with additional context
type AppError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Err     error                  `json:"-"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is interface for comparison
func (e *AppError) Is(target error) bool {
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

// HTTPStatus returns the HTTP status code for this error
func (e *AppError) HTTPStatus() int {
	return int(e.Code)
}

// WithDetails adds additional details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// NewAppError creates a new AppError with the given code, message, and underlying error
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NotFound creates a new not found error
func NotFound(resource string, err error) *AppError {
	return NewAppError(CodeNotFound, fmt.Sprintf("%s not found", resource), err)
}

// Unauthorized creates a new unauthorized error
func Unauthorized(message string, err error) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return NewAppError(CodeUnauthorized, message, err)
}

// Forbidden creates a new forbidden error
func Forbidden(message string, err error) *AppError {
	if message == "" {
		message = "access denied"
	}
	return NewAppError(CodeForbidden, message, err)
}

// BadRequest creates a new bad request error
func BadRequest(message string, err error) *AppError {
	if message == "" {
		message = "invalid request"
	}
	return NewAppError(CodeBadRequest, message, err)
}

// Conflict creates a new conflict error (for duplicate resources)
func Conflict(message string, err error) *AppError {
	return NewAppError(CodeConflict, message, err)
}

// InternalError creates a new internal server error
func InternalError(message string, err error) *AppError {
	if message == "" {
		message = "an internal error occurred"
	}
	return NewAppError(CodeInternalServerError, message, err)
}

// DatabaseError creates a new database error
func DatabaseError(operation string, err error) *AppError {
	return NewAppError(CodeInternalServerError, fmt.Sprintf("database %s failed", operation), err)
}

// StorageError creates a new storage error
func StorageError(operation string, err error) *AppError {
	return NewAppError(CodeInternalServerError, fmt.Sprintf("storage %s failed", operation), err)
}

// GitError creates a new git operation error
func GitError(operation string, err error) *AppError {
	return NewAppError(CodeInternalServerError, fmt.Sprintf("git %s failed", operation), err)
}

// ValidationError creates a new validation error with field details
func ValidationError(field, message string) *AppError {
	return NewAppError(CodeBadRequest, message, ErrInvalidInput).WithDetails(map[string]interface{}{
		"field": field,
	})
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized
	}
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeForbidden
	}
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if an error is a conflict error
func IsConflict(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeConflict
	}
	return errors.Is(err, ErrRepositoryExists) || errors.Is(err, ErrUserExists) ||
		errors.Is(err, ErrSSHKeyExists) || errors.Is(err, ErrBranchExists) ||
		errors.Is(err, ErrTagExists)
}

// IsBadRequest checks if an error is a bad request error
func IsBadRequest(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeBadRequest
	}
	return errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrInvalidSSHKey)
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// WrapWithCode wraps an error with a specific error code
func WrapWithCode(err error, code ErrorCode, message string) *AppError {
	return NewAppError(code, message, err)
}
