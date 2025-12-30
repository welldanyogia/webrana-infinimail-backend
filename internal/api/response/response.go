package response

import (
	"net/http"

	apperrors "github.com/welldanyogia/webrana-infinimail-backend/internal/errors"
	"github.com/labstack/echo/v4"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

// Meta contains pagination metadata
type Meta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// Success returns a successful response with data
func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage returns a successful response with a message
func SuccessWithMessage(c echo.Context, data interface{}, message string) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Created returns a 201 Created response
func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent returns a 204 No Content response
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Paginated returns a paginated response
func Paginated(c echo.Context, data interface{}, total int64, limit, offset int) error {
	return c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Error returns an error response with appropriate status code
func Error(c echo.Context, err error) error {
	code := apperrors.GetErrorCode(err)
	status := getHTTPStatus(code)
	
	return c.JSON(status, ErrorResponse{
		Success: false,
		Error:   err.Error(),
		Code:    code,
	})
}

// BadRequest returns a 400 Bad Request response
func BadRequest(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    apperrors.CodeInvalidInput,
	})
}

// BadRequestWithData returns a 400 Bad Request response with additional data
func BadRequestWithData(c echo.Context, message string, data interface{}) error {
	return c.JSON(http.StatusBadRequest, map[string]interface{}{
		"success": false,
		"error":   message,
		"code":    apperrors.CodeInvalidInput,
		"data":    data,
	})
}

// NotFound returns a 404 Not Found response
func NotFound(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    apperrors.CodeNotFound,
	})
}

// Conflict returns a 409 Conflict response
func Conflict(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    apperrors.CodeDuplicateEntry,
	})
}

// InternalError returns a 500 Internal Server Error response
func InternalError(c echo.Context, message string) error {
	return c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    apperrors.CodeInternalError,
	})
}

// ACMEErrorResponse represents an ACME-specific error response with detailed context
type ACMEErrorResponse struct {
	Success         bool     `json:"success"`
	Error           string   `json:"error"`
	Code            string   `json:"code"`
	ExpectedValue   string   `json:"expected_value,omitempty"`
	FoundValues     []string `json:"found_values,omitempty"`
	SuggestedAction string   `json:"suggested_action"`
	Domain          string   `json:"domain,omitempty"`
	TXTRecordName   string   `json:"txt_record_name,omitempty"`
	CurrentStatus   string   `json:"current_status,omitempty"`
	RequiredStatus  string   `json:"required_status,omitempty"`
}

// ACMEError returns an ACME-specific error response with detailed context
// This includes expected/found values for DNS errors and suggested actions
func ACMEError(c echo.Context, acmeErr *apperrors.ACMEError) error {
	status := getHTTPStatusForACME(acmeErr.Code)
	
	return c.JSON(status, ACMEErrorResponse{
		Success:         false,
		Error:           acmeErr.Message,
		Code:            acmeErr.Code,
		ExpectedValue:   acmeErr.ExpectedValue,
		FoundValues:     acmeErr.FoundValues,
		SuggestedAction: acmeErr.SuggestedAction,
		Domain:          acmeErr.Domain,
		TXTRecordName:   acmeErr.TXTRecordName,
		CurrentStatus:   acmeErr.CurrentStatus,
		RequiredStatus:  acmeErr.RequiredStatus,
	})
}

// ACMEErrorFromError checks if the error is an ACMEError and returns appropriate response
// If not an ACMEError, returns a generic internal error
func ACMEErrorFromError(c echo.Context, err error) error {
	if acmeErr := apperrors.GetACMEError(err); acmeErr != nil {
		return ACMEError(c, acmeErr)
	}
	return InternalError(c, err.Error())
}

// getHTTPStatusForACME maps ACME error codes to HTTP status codes
func getHTTPStatusForACME(code string) int {
	switch code {
	case apperrors.CodeACMEChallengeFailed:
		return http.StatusInternalServerError
	case apperrors.CodeACMEChallengeExpired:
		return http.StatusBadRequest
	case apperrors.CodeACMEDNSVerificationFailed:
		return http.StatusBadRequest
	case apperrors.CodeACMEValidationFailed:
		return http.StatusBadRequest
	case apperrors.CodeInvalidDomainStatus:
		return http.StatusBadRequest
	case apperrors.CodeNoChallengeFound:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// getHTTPStatus maps error codes to HTTP status codes
func getHTTPStatus(code string) int {
	switch code {
	case apperrors.CodeNotFound:
		return http.StatusNotFound
	case apperrors.CodeDuplicateEntry:
		return http.StatusConflict
	case apperrors.CodeInvalidInput:
		return http.StatusBadRequest
	case apperrors.CodeDomainNotActive:
		return http.StatusBadRequest
	case apperrors.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperrors.CodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
