package smtp

import (
	"os"
	"testing"
	"time"
)

func TestNewSecureServer(t *testing.T) {
	backend := &Backend{}

	t.Run("default configuration", func(t *testing.T) {
		cfg := &ServerConfig{
			Addr:   ":2525",
			Domain: "localhost",
		}

		server := NewSecureServer(backend, cfg)

		if server.Addr != ":2525" {
			t.Errorf("expected addr :2525, got %s", server.Addr)
		}
		if server.Domain != "localhost" {
			t.Errorf("expected domain localhost, got %s", server.Domain)
		}
		if server.MaxMessageBytes != DefaultMaxMessageSize {
			t.Errorf("expected max message size %d, got %d", DefaultMaxMessageSize, server.MaxMessageBytes)
		}
		if server.MaxRecipients != DefaultMaxRecipients {
			t.Errorf("expected max recipients %d, got %d", DefaultMaxRecipients, server.MaxRecipients)
		}
		if server.ReadTimeout != DefaultReadTimeout {
			t.Errorf("expected read timeout %v, got %v", DefaultReadTimeout, server.ReadTimeout)
		}
		if server.WriteTimeout != DefaultWriteTimeout {
			t.Errorf("expected write timeout %v, got %v", DefaultWriteTimeout, server.WriteTimeout)
		}
		if server.AllowInsecureAuth != false {
			t.Error("expected AllowInsecureAuth to be false by default")
		}
		if server.MaxLineLength != DefaultMaxLineLength {
			t.Errorf("expected max line length %d, got %d", DefaultMaxLineLength, server.MaxLineLength)
		}
	})

	t.Run("custom configuration", func(t *testing.T) {
		cfg := &ServerConfig{
			Addr:           ":25",
			Domain:         "mail.example.com",
			MaxMessageSize: 10 * 1024 * 1024, // 10 MB
			MaxRecipients:  50,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			AllowInsecure:  true,
		}

		server := NewSecureServer(backend, cfg)

		if server.MaxMessageBytes != 10*1024*1024 {
			t.Errorf("expected max message size 10MB, got %d", server.MaxMessageBytes)
		}
		if server.MaxRecipients != 50 {
			t.Errorf("expected max recipients 50, got %d", server.MaxRecipients)
		}
		if server.ReadTimeout != 30*time.Second {
			t.Errorf("expected read timeout 30s, got %v", server.ReadTimeout)
		}
		if server.WriteTimeout != 30*time.Second {
			t.Errorf("expected write timeout 30s, got %v", server.WriteTimeout)
		}
		if server.AllowInsecureAuth != true {
			t.Error("expected AllowInsecureAuth to be true when configured")
		}
	})

	t.Run("insecure auth disabled by default", func(t *testing.T) {
		cfg := &ServerConfig{
			Addr:   ":2525",
			Domain: "localhost",
		}

		server := NewSecureServer(backend, cfg)

		if server.AllowInsecureAuth {
			t.Error("AllowInsecureAuth should be disabled by default for security")
		}
	})

	t.Run("message size limit enforced", func(t *testing.T) {
		cfg := &ServerConfig{
			Addr:           ":2525",
			Domain:         "localhost",
			MaxMessageSize: 5 * 1024 * 1024, // 5 MB
		}

		server := NewSecureServer(backend, cfg)

		if server.MaxMessageBytes != 5*1024*1024 {
			t.Errorf("message size limit not enforced: expected 5MB, got %d", server.MaxMessageBytes)
		}
	})

	t.Run("recipient limit enforced", func(t *testing.T) {
		cfg := &ServerConfig{
			Addr:          ":2525",
			Domain:        "localhost",
			MaxRecipients: 10,
		}

		server := NewSecureServer(backend, cfg)

		if server.MaxRecipients != 10 {
			t.Errorf("recipient limit not enforced: expected 10, got %d", server.MaxRecipients)
		}
	})
}

func TestLoadServerConfigFromEnv(t *testing.T) {
	// Save original env vars
	origAddr := os.Getenv("SMTP_ADDR")
	origDomain := os.Getenv("SMTP_DOMAIN")
	origAllowInsecure := os.Getenv("SMTP_ALLOW_INSECURE")
	origMaxSize := os.Getenv("SMTP_MAX_MESSAGE_SIZE")
	origMaxRecip := os.Getenv("SMTP_MAX_RECIPIENTS")
	origReadTimeout := os.Getenv("SMTP_READ_TIMEOUT")
	origWriteTimeout := os.Getenv("SMTP_WRITE_TIMEOUT")

	// Restore env vars after test
	defer func() {
		os.Setenv("SMTP_ADDR", origAddr)
		os.Setenv("SMTP_DOMAIN", origDomain)
		os.Setenv("SMTP_ALLOW_INSECURE", origAllowInsecure)
		os.Setenv("SMTP_MAX_MESSAGE_SIZE", origMaxSize)
		os.Setenv("SMTP_MAX_RECIPIENTS", origMaxRecip)
		os.Setenv("SMTP_READ_TIMEOUT", origReadTimeout)
		os.Setenv("SMTP_WRITE_TIMEOUT", origWriteTimeout)
	}()

	t.Run("default values", func(t *testing.T) {
		os.Unsetenv("SMTP_ADDR")
		os.Unsetenv("SMTP_DOMAIN")
		os.Unsetenv("SMTP_ALLOW_INSECURE")
		os.Unsetenv("SMTP_MAX_MESSAGE_SIZE")
		os.Unsetenv("SMTP_MAX_RECIPIENTS")
		os.Unsetenv("SMTP_READ_TIMEOUT")
		os.Unsetenv("SMTP_WRITE_TIMEOUT")

		cfg := LoadServerConfigFromEnv()

		if cfg.Addr != ":2525" {
			t.Errorf("expected default addr :2525, got %s", cfg.Addr)
		}
		if cfg.Domain != "localhost" {
			t.Errorf("expected default domain localhost, got %s", cfg.Domain)
		}
		if cfg.AllowInsecure != false {
			t.Error("expected AllowInsecure to be false by default")
		}
	})

	t.Run("custom values from env", func(t *testing.T) {
		os.Setenv("SMTP_ADDR", ":25")
		os.Setenv("SMTP_DOMAIN", "mail.example.com")
		os.Setenv("SMTP_ALLOW_INSECURE", "true")
		os.Setenv("SMTP_MAX_MESSAGE_SIZE", "10485760")
		os.Setenv("SMTP_MAX_RECIPIENTS", "50")
		os.Setenv("SMTP_READ_TIMEOUT", "30s")
		os.Setenv("SMTP_WRITE_TIMEOUT", "45s")

		cfg := LoadServerConfigFromEnv()

		if cfg.Addr != ":25" {
			t.Errorf("expected addr :25, got %s", cfg.Addr)
		}
		if cfg.Domain != "mail.example.com" {
			t.Errorf("expected domain mail.example.com, got %s", cfg.Domain)
		}
		if cfg.AllowInsecure != true {
			t.Error("expected AllowInsecure to be true")
		}
		if cfg.MaxMessageSize != 10485760 {
			t.Errorf("expected max message size 10485760, got %d", cfg.MaxMessageSize)
		}
		if cfg.MaxRecipients != 50 {
			t.Errorf("expected max recipients 50, got %d", cfg.MaxRecipients)
		}
		if cfg.ReadTimeout != 30*time.Second {
			t.Errorf("expected read timeout 30s, got %v", cfg.ReadTimeout)
		}
		if cfg.WriteTimeout != 45*time.Second {
			t.Errorf("expected write timeout 45s, got %v", cfg.WriteTimeout)
		}
	})

	t.Run("invalid values use defaults", func(t *testing.T) {
		os.Setenv("SMTP_MAX_MESSAGE_SIZE", "invalid")
		os.Setenv("SMTP_MAX_RECIPIENTS", "invalid")
		os.Setenv("SMTP_READ_TIMEOUT", "invalid")
		os.Setenv("SMTP_WRITE_TIMEOUT", "invalid")
		os.Setenv("SMTP_ALLOW_INSECURE", "invalid")

		cfg := LoadServerConfigFromEnv()

		// Invalid values should result in zero/default values
		if cfg.MaxMessageSize != 0 {
			t.Errorf("expected max message size 0 for invalid input, got %d", cfg.MaxMessageSize)
		}
		if cfg.MaxRecipients != 0 {
			t.Errorf("expected max recipients 0 for invalid input, got %d", cfg.MaxRecipients)
		}
		if cfg.AllowInsecure != false {
			t.Error("expected AllowInsecure to be false for invalid input")
		}
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_KEY", "test_value")
		defer os.Unsetenv("TEST_KEY")

		result := getEnvOrDefault("TEST_KEY", "default")
		if result != "test_value" {
			t.Errorf("expected test_value, got %s", result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_KEY_NOT_SET")

		result := getEnvOrDefault("TEST_KEY_NOT_SET", "default")
		if result != "default" {
			t.Errorf("expected default, got %s", result)
		}
	})
}

func TestGetEnvBool(t *testing.T) {
	t.Run("returns true for true value", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "true")
		defer os.Unsetenv("TEST_BOOL")

		result := getEnvBool("TEST_BOOL", false)
		if result != true {
			t.Error("expected true")
		}
	})

	t.Run("returns false for false value", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "false")
		defer os.Unsetenv("TEST_BOOL")

		result := getEnvBool("TEST_BOOL", true)
		if result != false {
			t.Error("expected false")
		}
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "invalid")
		defer os.Unsetenv("TEST_BOOL")

		result := getEnvBool("TEST_BOOL", true)
		if result != true {
			t.Error("expected default value true")
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_BOOL_NOT_SET")

		result := getEnvBool("TEST_BOOL_NOT_SET", true)
		if result != true {
			t.Error("expected default value true")
		}
	})
}

func TestSecurityDefaults(t *testing.T) {
	t.Run("default max message size is 25MB", func(t *testing.T) {
		expected := int64(25 * 1024 * 1024)
		if DefaultMaxMessageSize != expected {
			t.Errorf("expected default max message size %d, got %d", expected, DefaultMaxMessageSize)
		}
	})

	t.Run("default max recipients is 100", func(t *testing.T) {
		if DefaultMaxRecipients != 100 {
			t.Errorf("expected default max recipients 100, got %d", DefaultMaxRecipients)
		}
	})

	t.Run("default read timeout is 60 seconds", func(t *testing.T) {
		if DefaultReadTimeout != 60*time.Second {
			t.Errorf("expected default read timeout 60s, got %v", DefaultReadTimeout)
		}
	})

	t.Run("default write timeout is 60 seconds", func(t *testing.T) {
		if DefaultWriteTimeout != 60*time.Second {
			t.Errorf("expected default write timeout 60s, got %v", DefaultWriteTimeout)
		}
	})

	t.Run("default max line length is 2000", func(t *testing.T) {
		if DefaultMaxLineLength != 2000 {
			t.Errorf("expected default max line length 2000, got %d", DefaultMaxLineLength)
		}
	})
}
