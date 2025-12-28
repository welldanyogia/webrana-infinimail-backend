// Package validator provides input validation and sanitization functions
// for the Infinimail backend security layer.
package validator

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validation errors
var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidDomain    = errors.New("invalid domain format")
	ErrInvalidLocalPart = errors.New("invalid local part format")
	ErrInputTooLong     = errors.New("input exceeds maximum length")
	ErrInvalidCharacter = errors.New("input contains invalid characters")
	ErrEmptyInput       = errors.New("input cannot be empty")
)

// Regex patterns for validation
var (
	// Domain regex: allows lowercase alphanumeric, hyphens, and dots
	// Must start and end with alphanumeric, labels max 63 chars
	domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)

	// Local part regex: allows lowercase alphanumeric, dots, underscores, hyphens
	// Must start with alphanumeric, max 64 chars
	localPartRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
)

// ValidateEmail validates email address format according to RFC 5322.
// Returns nil if valid, or an appropriate error.
func ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return ErrEmptyInput
	}

	// RFC 5321 specifies max email length of 254 characters
	if utf8.RuneCountInString(email) > 254 {
		return ErrInputTooLong
	}

	// Use Go's mail package for RFC 5322 validation
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}

	return nil
}

// ValidateDomain validates domain name format against DNS standards.
// Returns nil if valid, or an appropriate error.
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))

	if domain == "" {
		return ErrEmptyInput
	}

	// RFC 1035 specifies max domain length of 253 characters
	if len(domain) > 253 {
		return ErrInputTooLong
	}

	if !domainRegex.MatchString(domain) {
		return ErrInvalidDomain
	}

	return nil
}

// ValidateLocalPart validates the local part of an email address.
// Returns nil if valid, or an appropriate error.
func ValidateLocalPart(localPart string) error {
	localPart = strings.TrimSpace(strings.ToLower(localPart))

	if localPart == "" {
		return ErrEmptyInput
	}

	// RFC 5321 specifies max local part length of 64 characters
	if len(localPart) > 64 {
		return ErrInputTooLong
	}

	if !localPartRegex.MatchString(localPart) {
		return ErrInvalidLocalPart
	}

	return nil
}


// Pagination constants
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// ValidatePagination validates and sanitizes pagination parameters.
// Returns sanitized limit and offset values.
func ValidatePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

// SanitizeFilename removes dangerous characters from filename.
// Prevents path traversal and removes control characters.
func SanitizeFilename(filename string) string {
	// Remove path separators to prevent path traversal
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove control characters (ASCII 0-31 and 127)
	filename = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, filename)

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Limit length to 255 characters (common filesystem limit)
	if utf8.RuneCountInString(filename) > 255 {
		runes := []rune(filename)
		filename = string(runes[:255])
	}

	// Fallback for empty filename
	if filename == "" {
		return "unnamed"
	}

	return filename
}

// SanitizeString removes potentially dangerous characters and enforces length limits.
// Removes control characters and trims whitespace.
func SanitizeString(input string, maxLength int) string {
	// Remove control characters (ASCII 0-31 and 127)
	input = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, input)

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Enforce maximum length if specified
	if maxLength > 0 && utf8.RuneCountInString(input) > maxLength {
		runes := []rune(input)
		input = string(runes[:maxLength])
	}

	return input
}
