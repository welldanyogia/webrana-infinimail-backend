package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityLogger(t *testing.T) {
	logger := NewSecurityLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestSecurityLogger_AuthFailure_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.AuthFailure("192.168.1.1", "/api/test", "invalid_key")

	// Parse JSON output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "auth_failure", logEntry["event_type"])
	assert.Equal(t, "192.168.1.1", logEntry["ip"])
	assert.Equal(t, "/api/test", logEntry["path"])
	assert.Equal(t, "invalid_key", logEntry["reason"])
	assert.Contains(t, logEntry, "timestamp")
}

func TestSecurityLogger_RateLimitExceeded_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.RateLimitExceeded("192.168.1.1", "/api/test")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "rate_limit", logEntry["event_type"])
	assert.Equal(t, "192.168.1.1", logEntry["ip"])
	assert.Equal(t, "/api/test", logEntry["path"])
}

func TestSecurityLogger_SuspiciousActivity_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.SuspiciousActivity("192.168.1.1", "/api/test", "sql_injection_attempt")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "suspicious", logEntry["event_type"])
	assert.Equal(t, "sql_injection_attempt", logEntry["activity"])
}

func TestSecurityLogger_PathTraversalAttempt(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.PathTraversalAttempt("192.168.1.1", "/api/files", "../../../etc/passwd")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "path_traversal", logEntry["event_type"])
	assert.Equal(t, "../../../etc/passwd", logEntry["attempted_path"])
}

func TestSecurityLogger_InvalidOrigin(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.InvalidOrigin("192.168.1.1", "http://malicious.com")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "invalid_origin", logEntry["event_type"])
	assert.Equal(t, "http://malicious.com", logEntry["origin"])
}

func TestSecurityLogger_BlockedFileUpload(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.BlockedFileUpload("192.168.1.1", "malware.exe", "blocked_extension")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "blocked_upload", logEntry["event_type"])
	assert.Equal(t, "malware.exe", logEntry["filename"])
	assert.Equal(t, "blocked_extension", logEntry["reason"])
}

func TestSecurityLogger_SensitiveDataNotLogged(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	// Try to log sensitive data
	details := map[string]string{
		"username": "testuser",
		"password": "secret123",
		"api_key":  "sk-12345",
		"token":    "jwt-token",
		"path":     "/api/test",
	}

	logger.SecurityEvent("test_event", "192.168.1.1", details)

	output := buf.String()

	// Sensitive data should NOT be in output
	assert.NotContains(t, output, "secret123")
	assert.NotContains(t, output, "sk-12345")
	assert.NotContains(t, output, "jwt-token")

	// Non-sensitive data should be in output
	assert.Contains(t, output, "testuser")
	assert.Contains(t, output, "/api/test")
}

func TestSecurityLogger_GetLogger(t *testing.T) {
	logger := NewSecurityLogger()
	slogger := logger.GetLogger()

	assert.NotNil(t, slogger)
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"api_key", true},
		{"apikey", true},
		{"token", true},
		{"secret", true},
		{"authorization", true},
		{"credential", true},
		{"session", true},
		{"cookie", true},
		{"username", false},
		{"email", false},
		{"path", false},
		{"ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.Info("test message", slog.String("key", "value"))

	assert.Contains(t, buf.String(), "test message")
	assert.Contains(t, buf.String(), "value")
}

func TestSecurityLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.Error("error message", slog.String("error", "something went wrong"))

	assert.Contains(t, buf.String(), "error message")
	assert.Contains(t, buf.String(), "something went wrong")
}

func TestSecurityLogger_TimestampPresent(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	logger.AuthFailure("192.168.1.1", "/api/test", "test")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Check timestamp is present and is a valid time string
	timestamp, ok := logEntry["timestamp"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, timestamp)
	// Should be in RFC3339 format
	assert.True(t, strings.Contains(timestamp, "T"))
}

func TestSecurityLogger_IPAddressPresent(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := NewSecurityLoggerWithHandler(handler)

	testIP := "10.0.0.1"
	logger.AuthFailure(testIP, "/api/test", "test")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, testIP, logEntry["ip"])
}
