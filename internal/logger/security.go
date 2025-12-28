// Package logger provides secure logging functionality for the Infinimail backend.
package logger

import (
	"log/slog"
	"os"
	"time"
)

// SecurityLogger provides methods for logging security-related events.
// It ensures sensitive data is never logged.
type SecurityLogger struct {
	logger *slog.Logger
}

// NewSecurityLogger creates a new SecurityLogger with JSON output.
func NewSecurityLogger() *SecurityLogger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &SecurityLogger{
		logger: slog.New(handler),
	}
}

// NewSecurityLoggerWithHandler creates a SecurityLogger with a custom handler.
func NewSecurityLoggerWithHandler(handler slog.Handler) *SecurityLogger {
	return &SecurityLogger{
		logger: slog.New(handler),
	}
}

// AuthFailure logs a failed authentication attempt.
// Never logs the actual credentials.
func (s *SecurityLogger) AuthFailure(ip, path, reason string) {
	s.logger.Warn("authentication_failure",
		slog.String("event_type", "auth_failure"),
		slog.String("ip", ip),
		slog.String("path", path),
		slog.String("reason", reason),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// RateLimitExceeded logs when a client exceeds rate limits.
func (s *SecurityLogger) RateLimitExceeded(ip, path string) {
	s.logger.Warn("rate_limit_exceeded",
		slog.String("event_type", "rate_limit"),
		slog.String("ip", ip),
		slog.String("path", path),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// SuspiciousActivity logs potentially malicious activity.
func (s *SecurityLogger) SuspiciousActivity(ip, path, activity string) {
	s.logger.Warn("suspicious_activity",
		slog.String("event_type", "suspicious"),
		slog.String("ip", ip),
		slog.String("path", path),
		slog.String("activity", activity),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// PathTraversalAttempt logs a path traversal attempt.
func (s *SecurityLogger) PathTraversalAttempt(ip, path, attemptedPath string) {
	s.logger.Warn("path_traversal_attempt",
		slog.String("event_type", "path_traversal"),
		slog.String("ip", ip),
		slog.String("path", path),
		slog.String("attempted_path", attemptedPath),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// InvalidOrigin logs a rejected WebSocket connection due to invalid origin.
func (s *SecurityLogger) InvalidOrigin(ip, origin string) {
	s.logger.Warn("invalid_origin",
		slog.String("event_type", "invalid_origin"),
		slog.String("ip", ip),
		slog.String("origin", origin),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// BlockedFileUpload logs a blocked file upload attempt.
func (s *SecurityLogger) BlockedFileUpload(ip, filename, reason string) {
	s.logger.Warn("blocked_file_upload",
		slog.String("event_type", "blocked_upload"),
		slog.String("ip", ip),
		slog.String("filename", filename),
		slog.String("reason", reason),
		slog.Time("timestamp", time.Now().UTC()),
	)
}

// SecurityEvent logs a generic security event.
func (s *SecurityLogger) SecurityEvent(eventType, ip string, details map[string]string) {
	attrs := []any{
		slog.String("event_type", eventType),
		slog.String("ip", ip),
		slog.Time("timestamp", time.Now().UTC()),
	}

	for k, v := range details {
		// Filter out sensitive keys
		if isSensitiveKey(k) {
			continue
		}
		attrs = append(attrs, slog.String(k, v))
	}

	s.logger.Warn("security_event", attrs...)
}

// Info logs an informational message.
func (s *SecurityLogger) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Error logs an error message.
func (s *SecurityLogger) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// GetLogger returns the underlying slog.Logger for use with middleware.
func (s *SecurityLogger) GetLogger() *slog.Logger {
	return s.logger
}

// isSensitiveKey checks if a key might contain sensitive data.
func isSensitiveKey(key string) bool {
	sensitiveKeys := map[string]bool{
		"password":      true,
		"api_key":       true,
		"apikey":        true,
		"token":         true,
		"secret":        true,
		"authorization": true,
		"auth":          true,
		"credential":    true,
		"credentials":   true,
		"session":       true,
		"cookie":        true,
	}
	return sensitiveKeys[key]
}
