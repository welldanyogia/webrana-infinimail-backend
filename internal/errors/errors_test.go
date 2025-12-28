package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppError_CreatesErrorWithCorrectFields(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := NewAppError(baseErr, "custom message", CodeNotFound)

	assert.Equal(t, baseErr, appErr.Err)
	assert.Equal(t, "custom message", appErr.Message)
	assert.Equal(t, CodeNotFound, appErr.Code)
}

func TestAppError_Error_ReturnsMessage(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := NewAppError(baseErr, "custom message", CodeNotFound)

	assert.Equal(t, "custom message", appErr.Error())
}

func TestAppError_Error_ReturnsBaseErrorWhenNoMessage(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := NewAppError(baseErr, "", CodeNotFound)

	assert.Equal(t, "base error", appErr.Error())
}

func TestAppError_Unwrap_ReturnsWrappedError(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := NewAppError(baseErr, "custom message", CodeNotFound)

	unwrapped := appErr.Unwrap()
	assert.Equal(t, baseErr, unwrapped)
}

func TestWrap_WrapsErrorWithContext(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := Wrap(baseErr, "context")

	assert.Contains(t, wrapped.Error(), "context")
	assert.Contains(t, wrapped.Error(), "base error")
}

func TestWrap_ReturnsNilForNilError(t *testing.T) {
	wrapped := Wrap(nil, "context")
	assert.Nil(t, wrapped)
}

func TestIsNotFound_ReturnsTrueForNotFoundErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrNotFound", ErrNotFound, true},
		{"ErrMailboxNotFound", ErrMailboxNotFound, true},
		{"ErrMessageNotFound", ErrMessageNotFound, true},
		{"ErrAttachmentNotFound", ErrAttachmentNotFound, true},
		{"ErrDomainNotFound", ErrDomainNotFound, true},
		{"wrapped ErrNotFound", Wrap(ErrNotFound, "context"), true},
		{"other error", errors.New("other"), false},
		{"ErrDuplicateEntry", ErrDuplicateEntry, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsDuplicateEntry_ReturnsTrueForDuplicateErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrDuplicateEntry", ErrDuplicateEntry, true},
		{"wrapped ErrDuplicateEntry", Wrap(ErrDuplicateEntry, "context"), true},
		{"other error", errors.New("other"), false},
		{"ErrNotFound", ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicateEntry(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsInvalidInput_ReturnsTrueForInvalidInputErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrInvalidInput", ErrInvalidInput, true},
		{"wrapped ErrInvalidInput", Wrap(ErrInvalidInput, "context"), true},
		{"other error", errors.New("other"), false},
		{"ErrNotFound", ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInvalidInput(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsDomainNotActive_ReturnsTrueForDomainNotActiveErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrDomainNotActive", ErrDomainNotActive, true},
		{"wrapped ErrDomainNotActive", Wrap(ErrDomainNotActive, "context"), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDomainNotActive(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetErrorCode_ReturnsCorrectCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNotFound", ErrNotFound, CodeNotFound},
		{"ErrMailboxNotFound", ErrMailboxNotFound, CodeNotFound},
		{"ErrMessageNotFound", ErrMessageNotFound, CodeNotFound},
		{"ErrAttachmentNotFound", ErrAttachmentNotFound, CodeNotFound},
		{"ErrDomainNotFound", ErrDomainNotFound, CodeNotFound},
		{"ErrDuplicateEntry", ErrDuplicateEntry, CodeDuplicateEntry},
		{"ErrInvalidInput", ErrInvalidInput, CodeInvalidInput},
		{"ErrDomainNotActive", ErrDomainNotActive, CodeDomainNotActive},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized},
		{"ErrForbidden", ErrForbidden, CodeForbidden},
		{"unknown error", errors.New("unknown"), CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorCode(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestErrorCodes_AreCorrectValues(t *testing.T) {
	assert.Equal(t, "NOT_FOUND", CodeNotFound)
	assert.Equal(t, "DUPLICATE_ENTRY", CodeDuplicateEntry)
	assert.Equal(t, "INVALID_INPUT", CodeInvalidInput)
	assert.Equal(t, "DOMAIN_NOT_ACTIVE", CodeDomainNotActive)
	assert.Equal(t, "UNAUTHORIZED", CodeUnauthorized)
	assert.Equal(t, "FORBIDDEN", CodeForbidden)
	assert.Equal(t, "INTERNAL_ERROR", CodeInternalError)
}

func TestAppError_ImplementsErrorInterface(t *testing.T) {
	var err error = NewAppError(ErrNotFound, "test", CodeNotFound)
	assert.NotNil(t, err)
	assert.Equal(t, "test", err.Error())
}

func TestAppError_CanBeUnwrappedWithErrorsIs(t *testing.T) {
	appErr := NewAppError(ErrNotFound, "test", CodeNotFound)
	
	// errors.Is should work through Unwrap
	assert.True(t, errors.Is(appErr, ErrNotFound))
}
