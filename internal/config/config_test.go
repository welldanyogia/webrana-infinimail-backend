package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RequiredDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestLoad_DefaultValues(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.APIPort)
	assert.Equal(t, 2525, cfg.SMTPPort)
	assert.True(t, cfg.AutoProvisioningEnabled)
	assert.Equal(t, "./attachments", cfg.AttachmentStoragePath)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, 10.0, cfg.RateLimitRequests)
	assert.Equal(t, 20, cfg.RateLimitBurst)
}

func TestValidateProduction_RequiresAPIKey(t *testing.T) {
	cfg := &Config{
		DatabaseURL:    "postgres://localhost/test",
		AppEnv:         "production",
		AllowedOrigins: "http://example.com",
		APIKey:         "",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY is required")
}

func TestValidateProduction_RequiresAllowedOrigins(t *testing.T) {
	cfg := &Config{
		DatabaseURL:    "postgres://localhost/test",
		AppEnv:         "production",
		APIKey:         "test-key",
		AllowedOrigins: "",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ALLOWED_ORIGINS is required")
}

func TestValidateProduction_NoWildcardOrigins(t *testing.T) {
	cfg := &Config{
		DatabaseURL:    "postgres://localhost/test",
		AppEnv:         "production",
		APIKey:         "test-key",
		AllowedOrigins: "*",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard")
}

func TestValidateProduction_NoSSLDisable(t *testing.T) {
	cfg := &Config{
		DatabaseURL:    "postgres://localhost/test?sslmode=disable",
		AppEnv:         "production",
		APIKey:         "test-key",
		AllowedOrigins: "http://example.com",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sslmode=disable")
}

func TestValidateProduction_ValidConfig(t *testing.T) {
	cfg := &Config{
		DatabaseURL:    "postgres://localhost/test?sslmode=require",
		AppEnv:         "production",
		APIKey:         "test-key",
		AllowedOrigins: "http://example.com",
	}

	err := cfg.ValidateProduction()
	assert.NoError(t, err)
}

func TestLoadWithValidation_FailFast(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test?sslmode=disable")
	os.Setenv("APP_ENV", "production")
	os.Setenv("API_KEY", "test-key")
	os.Setenv("ALLOWED_ORIGINS", "http://example.com")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("APP_ENV")
		os.Unsetenv("API_KEY")
		os.Unsetenv("ALLOWED_ORIGINS")
	}()

	_, err := LoadWithValidation()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sslmode=disable")
}

func TestLoadWithValidation_DevelopmentAllowsInsecure(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test?sslmode=disable")
	os.Setenv("APP_ENV", "development")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("APP_ENV")
	}()

	cfg, err := LoadWithValidation()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := &Config{
		DatabaseURL:           "postgres://localhost/test",
		APIPort:               0,
		SMTPPort:              2525,
		AttachmentStoragePath: "./attachments",
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "APIPort")
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		DatabaseURL:           "postgres://localhost/test",
		APIPort:               8080,
		SMTPPort:              2525,
		AttachmentStoragePath: "./attachments",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestLoad_SecurityConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("API_KEY", "my-secret-key")
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000,http://example.com")
	os.Setenv("APP_ENV", "staging")
	os.Setenv("RATE_LIMIT_REQUESTS", "20")
	os.Setenv("RATE_LIMIT_BURST", "50")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("API_KEY")
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("APP_ENV")
		os.Unsetenv("RATE_LIMIT_REQUESTS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "my-secret-key", cfg.APIKey)
	assert.Equal(t, "http://localhost:3000,http://example.com", cfg.AllowedOrigins)
	assert.Equal(t, "staging", cfg.AppEnv)
	assert.Equal(t, 20.0, cfg.RateLimitRequests)
	assert.Equal(t, 50, cfg.RateLimitBurst)
}
