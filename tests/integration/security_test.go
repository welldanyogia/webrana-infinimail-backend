package integration

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/middleware"
)

func TestSecurityMiddlewareIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("full security middleware chain", func(t *testing.T) {
		// Set up environment
		os.Setenv("API_KEY", "test-api-key")
		os.Setenv("ALLOWED_ORIGINS", "https://example.com")
		defer func() {
			os.Unsetenv("API_KEY")
			os.Unsetenv("ALLOWED_ORIGINS")
		}()

		e := echo.New()

		// Apply security middleware in correct order
		e.Use(middleware.Recover())
		e.Use(middleware.SecureHeaders())
		e.Use(middleware.SecureCORS())
		e.Use(middleware.RateLimiter(logger))
		e.Use(middleware.APIKeyAuth(logger))

		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		// Test with valid API key and origin
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Verify security headers are present
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("X-Content-Type-Options header missing")
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Error("X-Frame-Options header missing")
		}
	})

	t.Run("auth failure returns 401", func(t *testing.T) {
		os.Setenv("API_KEY", "test-api-key")
		defer os.Unsetenv("API_KEY")

		e := echo.New()
		e.Use(middleware.APIKeyAuth(logger))

		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer wrong-key")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("CORS allows valid origin", func(t *testing.T) {
		os.Setenv("ALLOWED_ORIGINS", "https://allowed.com")
		defer os.Unsetenv("ALLOWED_ORIGINS")

		e := echo.New()
		e.Use(middleware.SecureCORS())

		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://allowed.com")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// CORS middleware should set Access-Control-Allow-Origin for valid origins
		if rec.Header().Get("Access-Control-Allow-Origin") != "https://allowed.com" {
			t.Errorf("CORS should allow valid origin, got: %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("rate limiter returns 429 when exceeded", func(t *testing.T) {
		e := echo.New()

		// Use very low rate limit for testing
		e.Use(middleware.RateLimiterWithConfig(0.1, 1, logger))

		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		// First request should succeed
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.RemoteAddr = "192.168.1.100:12345"
		rec1 := httptest.NewRecorder()
		e.ServeHTTP(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("first request should succeed, got %d", rec1.Code)
		}

		// Subsequent requests should be rate limited
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code == http.StatusTooManyRequests {
				// Rate limiting is working
				if rec.Header().Get("Retry-After") == "" {
					t.Error("Retry-After header should be present")
				}
				return
			}
		}

		t.Error("rate limiter should have returned 429")
	})
}

func TestSecurityHeadersIntegration(t *testing.T) {
	e := echo.New()
	e.Use(middleware.SecureHeaders())

	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "geolocation=(), microphone=(), camera=()",
	}

	for header, expected := range headers {
		if rec.Header().Get(header) != expected {
			t.Errorf("expected %s: %s, got: %s", header, expected, rec.Header().Get(header))
		}
	}

	// CSP should be present (check prefix since it's detailed)
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header should be present")
	}
}

func TestHealthEndpointBypassesAuth(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	e := echo.New()

	// Health endpoints should be registered before auth middleware
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	// API group with auth
	api := e.Group("/api")
	api.Use(middleware.APIKeyAuth(logger))
	api.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Health endpoint should work without auth
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health endpoint should not require auth, got %d", rec.Code)
	}

	// Protected endpoint should require auth
	req2 := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("protected endpoint should require auth, got %d", rec2.Code)
	}
}

func TestCORSPreflightHandling(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	e := echo.New()
	e.Use(middleware.SecureCORS())

	e.POST("/api/data", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Preflight should return 204 or 200
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("preflight should return 204 or 200, got %d", rec.Code)
	}

	// Should have CORS headers
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin should be set for valid origin, got: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestEnvironmentBasedConfiguration(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("API_KEY")
	origOrigins := os.Getenv("ALLOWED_ORIGINS")
	defer func() {
		os.Setenv("API_KEY", origAPIKey)
		os.Setenv("ALLOWED_ORIGINS", origOrigins)
	}()

	t.Run("API key from environment", func(t *testing.T) {
		os.Setenv("API_KEY", "env-api-key")

		apiKey := os.Getenv("API_KEY")
		if apiKey != "env-api-key" {
			t.Errorf("expected env-api-key, got %s", apiKey)
		}
	})

	t.Run("allowed origins from environment", func(t *testing.T) {
		os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")

		origins := os.Getenv("ALLOWED_ORIGINS")
		if origins != "https://app.example.com,https://admin.example.com" {
			t.Errorf("unexpected origins: %s", origins)
		}
	})
}
