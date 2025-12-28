package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_WithinLimit(t *testing.T) {
	e := echo.New()
	// Allow 10 requests per second with burst of 20
	e.Use(RateLimiterWithConfig(10, 20, nil))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// First request should pass
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	e := echo.New()
	// Very restrictive: 1 request per second, burst of 1
	e.Use(RateLimiterWithConfig(1, 1, nil))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// First request should pass
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	e := echo.New()
	// Very restrictive: 1 request per second, burst of 1
	e.Use(RateLimiterWithConfig(1, 1, nil))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	// Second request should have Retry-After header
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Equal(t, "60", rec2.Header().Get("Retry-After"))
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	e := echo.New()
	// Very restrictive: 1 request per second, burst of 1
	e.Use(RateLimiterWithConfig(1, 1, nil))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Real-IP", "192.168.1.1")
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Request from IP 2 should also pass (different IP)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Real-IP", "192.168.1.2")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Second request from IP 1 should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.Header.Set("X-Real-IP", "192.168.1.1")
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusTooManyRequests, rec3.Code)
}

func TestIPRateLimiter_GetLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(10, 20)

	// Get limiter for IP
	l1 := limiter.GetLimiter("192.168.1.1")
	assert.NotNil(t, l1)

	// Same IP should return same limiter (same pointer)
	l2 := limiter.GetLimiter("192.168.1.1")
	assert.Same(t, l1, l2)

	// Different IP should return different limiter (different pointer)
	l3 := limiter.GetLimiter("192.168.1.2")
	assert.NotSame(t, l1, l3)
}

func TestIPRateLimiter_CleanupOldEntries(t *testing.T) {
	limiter := NewIPRateLimiter(10, 20)

	// Add some entries
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")

	// Cleanup
	limiter.CleanupOldEntries()

	// After cleanup, getting limiter should create new one
	l := limiter.GetLimiter("192.168.1.1")
	assert.NotNil(t, l)
}

func TestRateLimiter_BurstAllowed(t *testing.T) {
	e := echo.New()
	// Allow burst of 5
	e.Use(RateLimiterWithConfig(1, 5, nil))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	// First 5 requests should pass (burst)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "Request %d should pass", i+1)
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
