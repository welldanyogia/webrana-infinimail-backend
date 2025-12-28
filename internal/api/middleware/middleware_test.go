package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRequestLogger_LogsRequestDetails(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	e := echo.New()
	e.Use(RequestLogger(logger))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify log contains expected fields
	logOutput := buf.String()
	assert.Contains(t, logOutput, "method")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "path")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "status")
	assert.Contains(t, logOutput, "latency")
}

func TestRequestLogger_LogsCorrectStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	e := echo.New()
	e.Use(RequestLogger(logger))

	e.GET("/notfound", func(c echo.Context) error {
		return c.String(http.StatusNotFound, "Not Found")
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, buf.String(), "404")
}

func TestCORS_SetsCorrectHeaders(t *testing.T) {
	e := echo.New()
	e.Use(CORS())

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// CORS headers should be set
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_HandlesPreflightOPTIONS(t *testing.T) {
	e := echo.New()
	e.Use(CORS())

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Preflight should return 204 or 200
	assert.True(t, rec.Code == http.StatusNoContent || rec.Code == http.StatusOK)
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
}

func TestRecover_CatchesPanicsAndReturns500(t *testing.T) {
	e := echo.New()
	e.Use(Recover())

	e.GET("/panic", func(c echo.Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		e.ServeHTTP(rec, req)
	})

	// Should return 500
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecover_AllowsNormalRequests(t *testing.T) {
	e := echo.New()
	e.Use(Recover())

	e.GET("/normal", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/normal", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestRequestLogger_HandlesErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	e := echo.New()
	e.Use(RequestLogger(logger))

	e.GET("/error", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Request should still be logged
	assert.Contains(t, buf.String(), "/error")
}

func TestCORS_AllowsAllOrigins(t *testing.T) {
	e := echo.New()
	e.Use(CORS())

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	origins := []string{
		"http://localhost:3000",
		"http://example.com",
		"https://app.example.com",
	}

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestCORS_AllowsRequiredMethods(t *testing.T) {
	e := echo.New()
	e.Use(CORS())

	// Add handlers for all methods
	e.GET("/test", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.POST("/test", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.PUT("/test", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.PATCH("/test", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.DELETE("/test", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "http://example.com")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
