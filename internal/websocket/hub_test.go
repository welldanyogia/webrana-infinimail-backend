package websocket

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSecureUpgrader_ValidOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000,http://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_InvalidOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://malicious.com")

	result := upgrader.CheckOrigin(req)
	assert.False(t, result)
}

func TestNewSecureUpgrader_EmptyOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Same-origin requests have empty Origin header
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_DefaultOrigin(t *testing.T) {
	os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Default should allow localhost:3000
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_MultipleOrigins(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, http://example.com, http://app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	tests := []struct {
		origin   string
		expected bool
	}{
		{"http://localhost:3000", true},
		{"http://example.com", true},
		{"http://app.example.com", true},
		{"http://other.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			req.Header.Set("Origin", tt.origin)

			result := upgrader.CheckOrigin(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultUpgrader_AllowsAll(t *testing.T) {
	upgrader := DefaultUpgrader()

	origins := []string{
		"http://localhost:3000",
		"http://example.com",
		"http://malicious.com",
		"",
	}

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if origin != "" {
				req.Header.Set("Origin", origin)
			}

			result := upgrader.CheckOrigin(req)
			assert.True(t, result)
		})
	}
}

func TestNewSecureUpgrader_TrimWhitespace(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "  http://localhost:3000  ,  http://example.com  ")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://example.com")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_BufferSizes(t *testing.T) {
	upgrader := NewSecureUpgrader(nil)

	assert.Equal(t, 1024, upgrader.ReadBufferSize)
	assert.Equal(t, 1024, upgrader.WriteBufferSize)
}

func TestHub_NewHub(t *testing.T) {
	hub := NewHub(nil)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.subscriptions)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_BroadcastNewMessage(t *testing.T) {
	hub := NewHub(nil)

	// Start hub in goroutine
	go hub.Run()

	// Create a test payload
	payload := &NewMessagePayload{
		ID:          1,
		SenderEmail: "test@example.com",
		Subject:     "Test Subject",
		ReceivedAt:  "2025-01-01T00:00:00Z",
	}

	// This should not panic even with no subscribers
	hub.BroadcastNewMessage(1, payload)
}

func TestNewSecureUpgrader_EmptyAllowedOrigins(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Should default to localhost:3000
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_CommaOnlyOrigins(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", ",,,")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Should default to localhost:3000 when all entries are empty
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestNewSecureUpgrader_CaseSensitive(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Origins are case-sensitive
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "HTTP://LOCALHOST:3000")

	result := upgrader.CheckOrigin(req)
	// Should be false because origins are case-sensitive
	assert.False(t, result)
}

func TestNewSecureUpgrader_OriginWithPath(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Origin header should not include path
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000/some/path")

	result := upgrader.CheckOrigin(req)
	// Should be false because origin includes path
	assert.False(t, result)
}

func TestNewSecureUpgrader_FilterEmptyStrings(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000,,http://example.com,")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	upgrader := NewSecureUpgrader(nil)

	// Both valid origins should work
	tests := []struct {
		origin   string
		expected bool
	}{
		{"http://localhost:3000", true},
		{"http://example.com", true},
		{"", true}, // Empty origin (same-origin) should be allowed
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		if tt.origin != "" {
			req.Header.Set("Origin", tt.origin)
		}

		result := upgrader.CheckOrigin(req)
		assert.Equal(t, tt.expected, result, "Origin: %s", tt.origin)
	}
}

// Helper to check if string is in slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == item {
			return true
		}
	}
	return false
}
