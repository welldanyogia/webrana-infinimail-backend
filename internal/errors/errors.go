package errors

import (
	"errors"
	"fmt"
)

// Domain-specific error types
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrDuplicateEntry indicates a unique constraint violation
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrInvalidInput indicates invalid input data
	ErrInvalidInput = errors.New("invalid input")

	// ErrDomainNotActive indicates the domain is not active
	ErrDomainNotActive = errors.New("domain is not active")

	// ErrMailboxNotFound indicates the mailbox was not found
	ErrMailboxNotFound = errors.New("mailbox not found")

	// ErrMessageNotFound indicates the message was not found
	ErrMessageNotFound = errors.New("message not found")

	// ErrAttachmentNotFound indicates the attachment was not found
	ErrAttachmentNotFound = errors.New("attachment not found")

	// ErrDomainNotFound indicates the domain was not found
	ErrDomainNotFound = errors.New("domain not found")

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates forbidden access
	ErrForbidden = errors.New("forbidden")

	// ErrInternal indicates an internal server error
	ErrInternal = errors.New("internal server error")
)

// Error codes for API responses
const (
	CodeNotFound       = "NOT_FOUND"
	CodeDuplicateEntry = "DUPLICATE_ENTRY"
	CodeInvalidInput   = "INVALID_INPUT"
	CodeDomainNotActive = "DOMAIN_NOT_ACTIVE"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeInternalError  = "INTERNAL_ERROR"
)

// AppError represents an application error with context
type AppError struct {
	Err     error
	Message string
	Code    string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(err error, message string, code string) *AppError {
	return &AppError{
		Err:     err,
		Message: message,
		Code:    code,
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrMailboxNotFound) ||
		errors.Is(err, ErrMessageNotFound) ||
		errors.Is(err, ErrAttachmentNotFound) ||
		errors.Is(err, ErrDomainNotFound)
}

// IsDuplicateEntry checks if the error is a duplicate entry error
func IsDuplicateEntry(err error) bool {
	return errors.Is(err, ErrDuplicateEntry)
}

// IsInvalidInput checks if the error is an invalid input error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsDomainNotActive checks if the error is a domain not active error
func IsDomainNotActive(err error) bool {
	return errors.Is(err, ErrDomainNotActive)
}

// GetErrorCode returns the appropriate error code for an error
func GetErrorCode(err error) string {
	switch {
	case IsNotFound(err):
		return CodeNotFound
	case IsDuplicateEntry(err):
		return CodeDuplicateEntry
	case IsInvalidInput(err):
		return CodeInvalidInput
	case IsDomainNotActive(err):
		return CodeDomainNotActive
	case errors.Is(err, ErrUnauthorized):
		return CodeUnauthorized
	case errors.Is(err, ErrForbidden):
		return CodeForbidden
	default:
		return CodeInternalError
	}
}
