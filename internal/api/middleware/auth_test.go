package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_HealthEndpointSkipsAuth(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/health")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_ReadyEndpointSkipsAuth(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/ready")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ready")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_NoAPIKeyConfigured(t *testing.T) {
	os.Unsetenv("API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")

	handler := APIKeyAuth(nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_WithLogger(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	logger := slog.Default()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")

	handler := APIKeyAuth(logger)(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	assert.Error(t, err)
}
