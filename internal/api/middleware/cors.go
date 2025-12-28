package middleware

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SecureCORS returns CORS middleware with secure configuration.
// Reads allowed origins from ALLOWED_ORIGINS environment variable.
// Does NOT allow wildcard (*) origin in production.
func SecureCORS() echo.MiddlewareFunc {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		// Default to localhost only in development
		allowedOrigins = "http://localhost:3000"
	}

	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	// Filter out wildcard in production
	env := os.Getenv("APP_ENV")
	if env == "production" {
		filteredOrigins := make([]string, 0, len(origins))
		for _, origin := range origins {
			if origin != "*" {
				filteredOrigins = append(filteredOrigins, origin)
			}
		}
		origins = filteredOrigins
		if len(origins) == 0 {
			origins = []string{"http://localhost:3000"}
		}
	}

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
