// Package middleware provides HTTP middleware for the Infinimail API.
package middleware

import (
	"crypto/subtle"
	"log/slog"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// APIKeyAuth validates API key from Authorization header.
// Uses constant-time comparison to prevent timing attacks.
func APIKeyAuth(logger *slog.Logger) echo.MiddlewareFunc {
	validAPIKey := os.Getenv("API_KEY")
	if validAPIKey == "" && logger != nil {
		logger.Warn("API_KEY not set - API is UNSECURED")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()

			// Skip auth for health endpoints
			if strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/ready") {
				return next(c)
			}

			// Skip if API_KEY not configured (development mode)
			if validAPIKey == "" {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				if logger != nil {
					logger.Warn("missing authorization header",
						slog.String("ip", c.RealIP()),
						slog.String("path", path))
				}
				return echo.NewHTTPError(401, map[string]string{
					"error": "missing authorization header",
					"code":  "UNAUTHORIZED",
				})
			}

			// Extract token from "Bearer <token>" format
			token := strings.TrimPrefix(authHeader, "Bearer ")
			token = strings.TrimSpace(token)

			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(token), []byte(validAPIKey)) != 1 {
				if logger != nil {
					logger.Warn("invalid API key attempt",
						slog.String("ip", c.RealIP()),
						slog.String("path", path))
				}
				return echo.NewHTTPError(401, map[string]string{
					"error": "invalid API key",
					"code":  "UNAUTHORIZED",
				})
			}

			return next(c)
		}
	}
}
