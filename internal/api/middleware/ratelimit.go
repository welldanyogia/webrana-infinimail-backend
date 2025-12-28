package middleware

import (
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// IPRateLimiter manages rate limiters per IP address
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// GetLimiter returns the rate limiter for the given IP
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter
}

// CleanupOldEntries removes old entries from the limiter map
// Should be called periodically to prevent memory leaks
func (i *IPRateLimiter) CleanupOldEntries() {
	i.mu.Lock()
	defer i.mu.Unlock()
	// Simple cleanup: clear all entries periodically
	// In production, you'd want to track last access time
	i.limiters = make(map[string]*rate.Limiter)
}

// RateLimiter returns rate limiting middleware
func RateLimiter(logger *slog.Logger) echo.MiddlewareFunc {
	// Read configuration from environment
	requestsPerSecond := 10.0 // default
	burst := 20               // default

	if rps := os.Getenv("RATE_LIMIT_REQUESTS"); rps != "" {
		if v, err := strconv.ParseFloat(rps, 64); err == nil {
			requestsPerSecond = v
		}
	}

	if b := os.Getenv("RATE_LIMIT_BURST"); b != "" {
		if v, err := strconv.Atoi(b); err == nil {
			burst = v
		}
	}

	limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), burst)

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.CleanupOldEntries()
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			l := limiter.GetLimiter(ip)

			if !l.Allow() {
				if logger != nil {
					logger.Warn("rate limit exceeded",
						slog.String("ip", ip),
						slog.String("path", c.Path()))
				}

				c.Response().Header().Set("Retry-After", "60")
				return echo.NewHTTPError(429, map[string]string{
					"error":       "rate limit exceeded",
					"code":        "RATE_LIMITED",
					"retry_after": "60",
				})
			}

			return next(c)
		}
	}
}

// RateLimiterWithConfig returns rate limiting middleware with custom config
func RateLimiterWithConfig(requestsPerSecond float64, burst int, logger *slog.Logger) echo.MiddlewareFunc {
	limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), burst)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			l := limiter.GetLimiter(ip)

			if !l.Allow() {
				if logger != nil {
					logger.Warn("rate limit exceeded",
						slog.String("ip", ip),
						slog.String("path", c.Path()))
				}

				c.Response().Header().Set("Retry-After", "60")
				return echo.NewHTTPError(429, map[string]string{
					"error":       "rate limit exceeded",
					"code":        "RATE_LIMITED",
					"retry_after": "60",
				})
			}

			return next(c)
		}
	}
}
