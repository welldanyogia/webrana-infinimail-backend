package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecureHeaders adds security headers to responses
func SecureHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()

			// Prevent clickjacking
			h.Set("X-Frame-Options", "DENY")

			// Prevent MIME sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// XSS Protection (legacy browsers)
			h.Set("X-XSS-Protection", "1; mode=block")

			// Content Security Policy
			h.Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

			// HSTS (only enable over HTTPS)
			if c.Scheme() == "https" {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Referrer policy
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions policy
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}
