package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSecureHeaders_AllHeadersPresent(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check all required security headers
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", rec.Header().Get("Permissions-Policy"))
}

func TestSecureHeaders_XFrameOptions(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}

func TestSecureHeaders_ContentTypeOptions(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

func TestSecureHeaders_XSSProtection(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
}

func TestSecureHeaders_ContentSecurityPolicy(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecureHeaders_ReferrerPolicy(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}

func TestSecureHeaders_PermissionsPolicy(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	pp := rec.Header().Get("Permissions-Policy")
	assert.Contains(t, pp, "geolocation=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "camera=()")
}

func TestSecureHeaders_HSTSNotOnHTTP(t *testing.T) {
	e := echo.New()
	e.Use(SecureHeaders())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// HTTP request (not HTTPS)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// HSTS should NOT be set on HTTP
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
}
