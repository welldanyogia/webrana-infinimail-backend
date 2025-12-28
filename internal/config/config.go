package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Database
	DatabaseURL string

	// Server ports
	APIPort  int
	SMTPPort int

	// Features
	AutoProvisioningEnabled bool

	// Storage
	AttachmentStoragePath string

	// Logging
	LogLevel string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Required: DATABASE_URL
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required but not set")
	}

	// API_PORT (default: 8080)
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		cfg.APIPort = 8080
	} else {
		port, err := strconv.Atoi(apiPort)
		if err != nil {
			return nil, fmt.Errorf("API_PORT must be a valid integer: %w", err)
		}
		cfg.APIPort = port
	}

	// SMTP_PORT (default: 2525)
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		cfg.SMTPPort = 2525
	} else {
		port, err := strconv.Atoi(smtpPort)
		if err != nil {
			return nil, fmt.Errorf("SMTP_PORT must be a valid integer: %w", err)
		}
		cfg.SMTPPort = port
	}

	// AUTO_PROVISIONING_ENABLED (default: true)
	autoProvisioning := os.Getenv("AUTO_PROVISIONING_ENABLED")
	if autoProvisioning == "" {
		cfg.AutoProvisioningEnabled = true
	} else {
		enabled, err := strconv.ParseBool(autoProvisioning)
		if err != nil {
			return nil, fmt.Errorf("AUTO_PROVISIONING_ENABLED must be a valid boolean: %w", err)
		}
		cfg.AutoProvisioningEnabled = enabled
	}

	// ATTACHMENT_STORAGE_PATH (default: ./attachments)
	cfg.AttachmentStoragePath = os.Getenv("ATTACHMENT_STORAGE_PATH")
	if cfg.AttachmentStoragePath == "" {
		cfg.AttachmentStoragePath = "./attachments"
	}

	// LOG_LEVEL (default: info)
	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DatabaseURL cannot be empty")
	}
	if c.APIPort <= 0 || c.APIPort > 65535 {
		return fmt.Errorf("APIPort must be between 1 and 65535")
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("SMTPPort must be between 1 and 65535")
	}
	if c.AttachmentStoragePath == "" {
		return fmt.Errorf("AttachmentStoragePath cannot be empty")
	}
	return nil
}
