package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/welldanyogia/webrana-infinimail-backend/internal/errors"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestSuccess_Returns200WithData(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]string{"key": "value"}
	err := Success(c, data)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestSuccessWithMessage_Returns200WithMessage(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]string{"key": "value"}
	err := SuccessWithMessage(c, data, "Operation successful")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, "Operation successful", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestCreated_Returns201WithData(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]int{"id": 1}
	err := Created(c, data)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestNoContent_Returns204(t *testing.T) {
	c, rec := setupTestContext()

	err := NoContent(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestPaginated_ReturnsDataWithMeta(t *testing.T) {
	c, rec := setupTestContext()

	data := []string{"item1", "item2"}
	err := Paginated(c, data, 100, 20, 0)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp PaginatedResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, int64(100), resp.Meta.Total)
	assert.Equal(t, 20, resp.Meta.Limit)
	assert.Equal(t, 0, resp.Meta.Offset)
}

func TestError_ReturnsCorrectStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "not found error",
			err:        apperrors.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   apperrors.CodeNotFound,
		},
		{
			name:       "duplicate entry error",
			err:        apperrors.ErrDuplicateEntry,
			wantStatus: http.StatusConflict,
			wantCode:   apperrors.CodeDuplicateEntry,
		},
		{
			name:       "invalid input error",
			err:        apperrors.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCode:   apperrors.CodeInvalidInput,
		},
		{
			name:       "unauthorized error",
			err:        apperrors.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperrors.CodeUnauthorized,
		},
		{
			name:       "forbidden error",
			err:        apperrors.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   apperrors.CodeForbidden,
		},
		{
			name:       "unknown error",
			err:        errors.New("unknown error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperrors.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := setupTestContext()

			err := Error(c, tt.err)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.False(t, resp.Success)
			assert.Equal(t, tt.wantCode, resp.Code)
		})
	}
}

func TestBadRequest_Returns400(t *testing.T) {
	c, rec := setupTestContext()

	err := BadRequest(c, "invalid input")

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, "invalid input", resp.Error)
	assert.Equal(t, apperrors.CodeInvalidInput, resp.Code)
}

func TestNotFound_Returns404(t *testing.T) {
	c, rec := setupTestContext()

	err := NotFound(c, "resource not found")

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, "resource not found", resp.Error)
	assert.Equal(t, apperrors.CodeNotFound, resp.Code)
}

func TestConflict_Returns409(t *testing.T) {
	c, rec := setupTestContext()

	err := Conflict(c, "duplicate entry")

	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, "duplicate entry", resp.Error)
	assert.Equal(t, apperrors.CodeDuplicateEntry, resp.Code)
}

func TestInternalError_Returns500(t *testing.T) {
	c, rec := setupTestContext()

	err := InternalError(c, "internal server error")

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, "internal server error", resp.Error)
	assert.Equal(t, apperrors.CodeInternalError, resp.Code)
}

func TestGetHTTPStatus_MapsCodesCorrectly(t *testing.T) {
	tests := []struct {
		code   string
		status int
	}{
		{apperrors.CodeNotFound, http.StatusNotFound},
		{apperrors.CodeDuplicateEntry, http.StatusConflict},
		{apperrors.CodeInvalidInput, http.StatusBadRequest},
		{apperrors.CodeDomainNotActive, http.StatusBadRequest},
		{apperrors.CodeUnauthorized, http.StatusUnauthorized},
		{apperrors.CodeForbidden, http.StatusForbidden},
		{apperrors.CodeInternalError, http.StatusInternalServerError},
		{"UNKNOWN_CODE", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			status := getHTTPStatus(tt.code)
			assert.Equal(t, tt.status, status)
		})
	}
}

func TestSuccess_WithNilData(t *testing.T) {
	c, rec := setupTestContext()

	err := Success(c, nil)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
}

func TestPaginated_WithEmptyData(t *testing.T) {
	c, rec := setupTestContext()

	data := []string{}
	err := Paginated(c, data, 0, 20, 0)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp PaginatedResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, int64(0), resp.Meta.Total)
}

func TestAPIResponse_JSONStructure(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]interface{}{
		"id":   1,
		"name": "test",
	}
	err := Success(c, data)

	require.NoError(t, err)

	// Verify JSON structure
	var raw map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "success")
	assert.Contains(t, raw, "data")
}

func TestErrorResponse_JSONStructure(t *testing.T) {
	c, rec := setupTestContext()

	err := BadRequest(c, "test error")

	require.NoError(t, err)

	// Verify JSON structure
	var raw map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "success")
	assert.Contains(t, raw, "error")
	assert.Contains(t, raw, "code")
}
