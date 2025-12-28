package api

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
