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
		DatabaseURL:      "postgres://localhost/test?sslmode=require",
		AppEnv:           "production",
		APIKey:           "test-key",
		AllowedOrigins:   "http://example.com",
		ACMEStaging:      false,
		ACMEEmail:        "admin@example.com",
		ACMEDirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	}

	err := cfg.ValidateProduction()
	assert.NoError(t, err)
}

func TestValidateProduction_ACMEStagingNotAllowed(t *testing.T) {
	cfg := &Config{
		DatabaseURL:      "postgres://localhost/test?sslmode=require",
		AppEnv:           "production",
		APIKey:           "test-key",
		AllowedOrigins:   "http://example.com",
		ACMEStaging:      true,
		ACMEEmail:        "admin@example.com",
		ACMEDirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME_STAGING must be false")
}

func TestValidateProduction_ACMEEmailRequired(t *testing.T) {
	cfg := &Config{
		DatabaseURL:      "postgres://localhost/test?sslmode=require",
		AppEnv:           "production",
		APIKey:           "test-key",
		AllowedOrigins:   "http://example.com",
		ACMEStaging:      false,
		ACMEEmail:        "",
		ACMEDirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME_EMAIL is required")
}

func TestValidateProduction_ACMEStagingURLNotAllowed(t *testing.T) {
	cfg := &Config{
		DatabaseURL:      "postgres://localhost/test?sslmode=require",
		AppEnv:           "production",
		APIKey:           "test-key",
		AllowedOrigins:   "http://example.com",
		ACMEStaging:      false,
		ACMEEmail:        "admin@example.com",
		ACMEDirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
	}

	err := cfg.ValidateProduction()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME_DIRECTORY_URL should use production endpoint")
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

func TestLoad_ACMEConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("ACME_DIRECTORY_URL", "https://acme-v02.api.letsencrypt.org/directory")
	os.Setenv("ACME_EMAIL", "admin@example.com")
	os.Setenv("ACME_STAGING", "false")
	os.Setenv("CERT_STORAGE_PATH", "/etc/certs")
	os.Setenv("CERT_RENEWAL_DAYS", "14")
	os.Setenv("CERT_RENEWAL_CHECK_INTERVAL", "12h")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ACME_DIRECTORY_URL")
		os.Unsetenv("ACME_EMAIL")
		os.Unsetenv("ACME_STAGING")
		os.Unsetenv("CERT_STORAGE_PATH")
		os.Unsetenv("CERT_RENEWAL_DAYS")
		os.Unsetenv("CERT_RENEWAL_CHECK_INTERVAL")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "https://acme-v02.api.letsencrypt.org/directory", cfg.ACMEDirectoryURL)
	assert.Equal(t, "admin@example.com", cfg.ACMEEmail)
	assert.False(t, cfg.ACMEStaging)
	assert.Equal(t, "/etc/certs", cfg.CertStoragePath)
	assert.Equal(t, 14, cfg.CertRenewalDays)
	assert.Equal(t, "12h", cfg.CertRenewalCheckInterval)
}

func TestLoad_ACMEConfigDefaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	require.NoError(t, err)

	// Check ACME defaults
	assert.Equal(t, "https://acme-staging-v02.api.letsencrypt.org/directory", cfg.ACMEDirectoryURL)
	assert.Equal(t, "", cfg.ACMEEmail)
	assert.True(t, cfg.ACMEStaging)
	assert.Equal(t, "./certs", cfg.CertStoragePath)
	assert.Equal(t, 30, cfg.CertRenewalDays)
	assert.Equal(t, "24h", cfg.CertRenewalCheckInterval)
}

func TestLoad_DNSGuideConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("SMTP_HOSTNAME", "mail.example.com")
	os.Setenv("SERVER_IP", "192.168.1.100")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("SMTP_HOSTNAME")
		os.Unsetenv("SERVER_IP")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "mail.example.com", cfg.SMTPHostname)
	assert.Equal(t, "192.168.1.100", cfg.ServerIP)
}

func TestLoad_DNSGuideConfigDefaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	require.NoError(t, err)

	// Check DNS guide defaults
	assert.Equal(t, "mail.infinimail.local", cfg.SMTPHostname)
	assert.Equal(t, "127.0.0.1", cfg.ServerIP)
}

func TestLoad_InvalidACMEStaging(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("ACME_STAGING", "invalid")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ACME_STAGING")
	}()

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME_STAGING must be a valid boolean")
}

func TestLoad_InvalidCertRenewalDays(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("CERT_RENEWAL_DAYS", "invalid")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("CERT_RENEWAL_DAYS")
	}()

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CERT_RENEWAL_DAYS must be a valid integer")
}
