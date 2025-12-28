package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		// Valid emails
		{"valid simple email", "test@example.com", nil},
		{"valid with subdomain", "user@mail.example.com", nil},
		{"valid with plus", "user+tag@example.com", nil},
		{"valid with dots", "first.last@example.com", nil},
		{"valid uppercase normalized", "TEST@EXAMPLE.COM", nil},
		{"valid with whitespace trimmed", "  test@example.com  ", nil},

		// Invalid emails
		{"empty string", "", ErrEmptyInput},
		{"whitespace only", "   ", ErrEmptyInput},
		{"missing @", "testexample.com", ErrInvalidEmail},
		{"missing domain", "test@", ErrInvalidEmail},
		{"missing local part", "@example.com", ErrInvalidEmail},
		{"double @", "test@@example.com", ErrInvalidEmail},
		{"invalid chars", "test<>@example.com", ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail_TooLong(t *testing.T) {
	// Create email longer than 254 characters
	// Need to ensure total length > 254
	longLocal := strings.Repeat("a", 250)
	longEmail := longLocal + "@example.com" // Total: 250 + 12 = 262 chars
	err := ValidateEmail(longEmail)
	assert.ErrorIs(t, err, ErrInputTooLong)
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr error
	}{
		// Valid domains
		{"valid simple", "example.com", nil},
		{"valid subdomain", "mail.example.com", nil},
		{"valid with hyphen", "my-domain.com", nil},
		{"valid short tld", "example.co", nil},
		{"valid uppercase normalized", "EXAMPLE.COM", nil},
		{"valid with whitespace trimmed", "  example.com  ", nil},
		{"valid single label", "localhost", nil},

		// Invalid domains
		{"empty string", "", ErrEmptyInput},
		{"whitespace only", "   ", ErrEmptyInput},
		{"starts with hyphen", "-example.com", ErrInvalidDomain},
		{"ends with hyphen", "example-.com", ErrInvalidDomain},
		{"double dot", "example..com", ErrInvalidDomain},
		{"starts with dot", ".example.com", ErrInvalidDomain},
		{"contains underscore", "my_domain.com", ErrInvalidDomain},
		{"contains space", "my domain.com", ErrInvalidDomain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDomain_TooLong(t *testing.T) {
	// Create domain longer than 253 characters
	longDomain := strings.Repeat("a", 254)
	err := ValidateDomain(longDomain)
	assert.ErrorIs(t, err, ErrInputTooLong)
}

func TestValidateLocalPart(t *testing.T) {
	tests := []struct {
		name      string
		localPart string
		wantErr   error
	}{
		// Valid local parts
		{"valid simple", "user", nil},
		{"valid with dot", "first.last", nil},
		{"valid with underscore", "user_name", nil},
		{"valid with hyphen", "user-name", nil},
		{"valid alphanumeric", "user123", nil},
		{"valid uppercase normalized", "USER", nil},
		{"valid with whitespace trimmed", "  user  ", nil},

		// Invalid local parts
		{"empty string", "", ErrEmptyInput},
		{"whitespace only", "   ", ErrEmptyInput},
		{"starts with dot", ".user", ErrInvalidLocalPart},
		{"starts with hyphen", "-user", ErrInvalidLocalPart},
		{"starts with underscore", "_user", ErrInvalidLocalPart},
		{"contains space", "user name", ErrInvalidLocalPart},
		{"contains @", "user@name", ErrInvalidLocalPart},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocalPart(tt.localPart)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLocalPart_TooLong(t *testing.T) {
	// Create local part longer than 64 characters
	longLocalPart := "a" + strings.Repeat("b", 64)
	err := ValidateLocalPart(longLocalPart)
	assert.ErrorIs(t, err, ErrInputTooLong)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal filename", "document.pdf", "document.pdf"},
		{"with spaces", "my document.pdf", "my document.pdf"},
		{"path traversal dots", "../../../etc/passwd", "______etc_passwd"},
		{"forward slash", "path/to/file.txt", "path_to_file.txt"},
		{"backslash", "path\\to\\file.txt", "path_to_file.txt"},
		{"control chars", "file\x00name.txt", "filename.txt"},
		{"tab character", "file\tname.txt", "filename.txt"},
		{"newline", "file\nname.txt", "filename.txt"},
		{"empty string", "", "unnamed"},
		{"whitespace only", "   ", "unnamed"},
		{"double dots", "file..name", "file_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename_LongFilename(t *testing.T) {
	// Create filename longer than 255 characters
	longFilename := strings.Repeat("a", 300) + ".txt"
	result := SanitizeFilename(longFilename)
	assert.LessOrEqual(t, len(result), 255)
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{"normal string", "hello world", 0, "hello world"},
		{"with control chars", "hello\x00world", 0, "helloworld"},
		{"with tab", "hello\tworld", 0, "helloworld"},
		{"with newline", "hello\nworld", 0, "helloworld"},
		{"trim whitespace", "  hello  ", 0, "hello"},
		{"enforce max length", "hello world", 5, "hello"},
		{"max length zero means no limit", "hello world", 0, "hello world"},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		expectedLimit  int
		expectedOffset int
	}{
		{"valid values", 10, 20, 10, 20},
		{"zero limit uses default", 0, 0, DefaultLimit, 0},
		{"negative limit uses default", -5, 0, DefaultLimit, 0},
		{"limit exceeds max", 200, 0, MaxLimit, 0},
		{"negative offset becomes zero", 10, -5, 10, 0},
		{"all defaults", 0, -1, DefaultLimit, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := ValidatePagination(tt.inputLimit, tt.inputOffset)
			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}
