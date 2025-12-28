package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSSLMode_DisabledNotAllowed(t *testing.T) {
	err := validateSSLMode("postgres://user:pass@localhost:5432/db?sslmode=disable")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSL mode cannot be disabled")
}

func TestValidateSSLMode_RequireAllowed(t *testing.T) {
	err := validateSSLMode("postgres://user:pass@localhost:5432/db?sslmode=require")
	assert.NoError(t, err)
}

func TestValidateSSLMode_VerifyFullAllowed(t *testing.T) {
	err := validateSSLMode("postgres://user:pass@localhost:5432/db?sslmode=verify-full")
	assert.NoError(t, err)
}

func TestValidateSSLMode_NoSSLModeAllowed(t *testing.T) {
	// If no sslmode specified, it's okay (defaults to prefer/require)
	err := validateSSLMode("postgres://user:pass@localhost:5432/db")
	assert.NoError(t, err)
}

func TestConnect_ProductionSSLRequired(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	// This should fail because sslmode=disable is not allowed in production
	_, err := Connect("postgres://user:pass@localhost:5432/db?sslmode=disable")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSL mode cannot be disabled")
}

func TestConnect_DevelopmentSSLNotRequired(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	defer os.Unsetenv("APP_ENV")

	// In development, sslmode=disable should be allowed
	// Note: This will fail to connect but should not fail SSL validation
	_, err := Connect("postgres://user:pass@localhost:5432/db?sslmode=disable")
	// Error should be about connection, not SSL
	if err != nil {
		assert.NotContains(t, err.Error(), "SSL mode cannot be disabled")
	}
}

func TestConnectionPoolDefaults(t *testing.T) {
	assert.Equal(t, 10, DefaultMaxIdleConns)
	assert.Equal(t, 100, DefaultMaxOpenConns)
}
