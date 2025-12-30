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

	// ACME-specific errors
	// ErrACMEChallengeFailed indicates ACME challenge request failed
	ErrACMEChallengeFailed = errors.New("ACME challenge failed")

	// ErrACMEChallengeExpired indicates ACME challenge has expired
	ErrACMEChallengeExpired = errors.New("ACME challenge expired")

	// ErrACMEDNSVerificationFailed indicates DNS verification failed
	ErrACMEDNSVerificationFailed = errors.New("DNS verification failed")

	// ErrACMEValidationFailed indicates Let's Encrypt validation failed
	ErrACMEValidationFailed = errors.New("ACME validation failed")

	// ErrInvalidDomainStatus indicates domain is not in the correct status
	ErrInvalidDomainStatus = errors.New("invalid domain status")

	// ErrNoChallengeFound indicates no active ACME challenge exists
	ErrNoChallengeFound = errors.New("no active ACME challenge found")
)

// Error codes for API responses
const (
	CodeNotFound            = "NOT_FOUND"
	CodeDuplicateEntry      = "DUPLICATE_ENTRY"
	CodeInvalidInput        = "INVALID_INPUT"
	CodeDomainNotActive     = "DOMAIN_NOT_ACTIVE"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeInternalError       = "INTERNAL_ERROR"
	CodeACMEChallengeFailed = "ACME_CHALLENGE_FAILED"
	CodeACMEChallengeExpired = "ACME_CHALLENGE_EXPIRED"
	CodeACMEDNSVerificationFailed = "DNS_VERIFICATION_FAILED"
	CodeACMEValidationFailed = "ACME_VALIDATION_FAILED"
	CodeInvalidDomainStatus = "INVALID_STATUS"
	CodeNoChallengeFound    = "NO_CHALLENGE_FOUND"
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

// ACMEError represents an ACME-specific error with detailed context
// This error type includes expected/found values for DNS verification
// and suggested actions for the user to take
type ACMEError struct {
	Err             error    `json:"-"`
	Code            string   `json:"code"`
	Message         string   `json:"message"`
	ExpectedValue   string   `json:"expected_value,omitempty"`
	FoundValues     []string `json:"found_values,omitempty"`
	SuggestedAction string   `json:"suggested_action"`
	Domain          string   `json:"domain,omitempty"`
	TXTRecordName   string   `json:"txt_record_name,omitempty"`
	CurrentStatus   string   `json:"current_status,omitempty"`
	RequiredStatus  string   `json:"required_status,omitempty"`
}

// Error implements the error interface
func (e *ACMEError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error
func (e *ACMEError) Unwrap() error {
	return e.Err
}

// NewACMEError creates a new ACMEError with basic info
func NewACMEError(err error, code, message, suggestedAction string) *ACMEError {
	return &ACMEError{
		Err:             err,
		Code:            code,
		Message:         message,
		SuggestedAction: suggestedAction,
	}
}

// NewACMEDNSVerificationError creates an ACMEError for DNS verification failures
// with expected and found values included
func NewACMEDNSVerificationError(domain, txtRecordName, expectedValue string, foundValues []string) *ACMEError {
	var message string
	var suggestedAction string

	if len(foundValues) == 0 {
		message = fmt.Sprintf("DNS TXT record not found for %s", txtRecordName)
		suggestedAction = fmt.Sprintf("Please add a TXT record with name '%s' and value '%s' to your DNS provider. DNS propagation may take up to 24 hours.", txtRecordName, expectedValue)
	} else {
		message = fmt.Sprintf("DNS TXT record value mismatch for %s", txtRecordName)
		suggestedAction = fmt.Sprintf("The TXT record exists but has incorrect value. Expected: '%s', Found: %v. Please update the record with the correct value.", expectedValue, foundValues)
	}

	return &ACMEError{
		Err:             ErrACMEDNSVerificationFailed,
		Code:            CodeACMEDNSVerificationFailed,
		Message:         message,
		ExpectedValue:   expectedValue,
		FoundValues:     foundValues,
		SuggestedAction: suggestedAction,
		Domain:          domain,
		TXTRecordName:   txtRecordName,
	}
}

// NewACMEChallengeExpiredError creates an ACMEError for expired challenges
func NewACMEChallengeExpiredError(domain string) *ACMEError {
	return &ACMEError{
		Err:             ErrACMEChallengeExpired,
		Code:            CodeACMEChallengeExpired,
		Message:         fmt.Sprintf("ACME challenge has expired for domain %s", domain),
		SuggestedAction: "Please request a new ACME challenge using POST /api/domains/:id/request-acme-challenge",
		Domain:          domain,
	}
}

// NewACMEValidationError creates an ACMEError for Let's Encrypt validation failures
func NewACMEValidationError(domain string, originalErr error) *ACMEError {
	return &ACMEError{
		Err:             ErrACMEValidationFailed,
		Code:            CodeACMEValidationFailed,
		Message:         fmt.Sprintf("Let's Encrypt validation failed for domain %s: %v", domain, originalErr),
		SuggestedAction: "Please verify the TXT record is correctly configured and try again. If the problem persists, request a new challenge.",
		Domain:          domain,
	}
}

// NewInvalidDomainStatusError creates an ACMEError for invalid domain status
func NewInvalidDomainStatusError(domain, currentStatus, requiredStatus, action string) *ACMEError {
	return &ACMEError{
		Err:             ErrInvalidDomainStatus,
		Code:            CodeInvalidDomainStatus,
		Message:         fmt.Sprintf("Domain %s is in '%s' status, but '%s' status is required to %s", domain, currentStatus, requiredStatus, action),
		SuggestedAction: fmt.Sprintf("Please complete the previous step first. Current status: %s, Required: %s", currentStatus, requiredStatus),
		Domain:          domain,
		CurrentStatus:   currentStatus,
		RequiredStatus:  requiredStatus,
	}
}

// NewNoChallengeFoundError creates an ACMEError when no active challenge exists
func NewNoChallengeFoundError(domain string) *ACMEError {
	return &ACMEError{
		Err:             ErrNoChallengeFound,
		Code:            CodeNoChallengeFound,
		Message:         fmt.Sprintf("No active ACME challenge found for domain %s", domain),
		SuggestedAction: "Please request an ACME challenge first using POST /api/domains/:id/request-acme-challenge",
		Domain:          domain,
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
	case errors.Is(err, ErrACMEChallengeFailed):
		return CodeACMEChallengeFailed
	case errors.Is(err, ErrACMEChallengeExpired):
		return CodeACMEChallengeExpired
	case errors.Is(err, ErrACMEDNSVerificationFailed):
		return CodeACMEDNSVerificationFailed
	case errors.Is(err, ErrACMEValidationFailed):
		return CodeACMEValidationFailed
	case errors.Is(err, ErrInvalidDomainStatus):
		return CodeInvalidDomainStatus
	case errors.Is(err, ErrNoChallengeFound):
		return CodeNoChallengeFound
	default:
		return CodeInternalError
	}
}

// IsACMEError checks if the error is an ACME-related error
func IsACMEError(err error) bool {
	var acmeErr *ACMEError
	return errors.As(err, &acmeErr)
}

// GetACMEError extracts ACMEError from an error if it exists
func GetACMEError(err error) *ACMEError {
	var acmeErr *ACMEError
	if errors.As(err, &acmeErr) {
		return acmeErr
	}
	return nil
}
