package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
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

	// Security
	APIKey         string
	AllowedOrigins string
	AppEnv         string

	// Rate Limiting
	RateLimitRequests float64
	RateLimitBurst    int

	// Domain Manager / DNS Guide Configuration
	SMTPHostname string
	ServerIP     string

	// ACME / Certificate Configuration
	ACMEDirectoryURL          string
	ACMEEmail                 string
	ACMEStaging               bool
	CertStoragePath           string
	CertRenewalDays           int
	CertRenewalCheckInterval  string
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

	// Security configuration
	cfg.APIKey = os.Getenv("API_KEY")
	cfg.AllowedOrigins = os.Getenv("ALLOWED_ORIGINS")
	cfg.AppEnv = os.Getenv("APP_ENV")
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}

	// Rate limiting configuration
	if rps := os.Getenv("RATE_LIMIT_REQUESTS"); rps != "" {
		if v, err := strconv.ParseFloat(rps, 64); err == nil {
			cfg.RateLimitRequests = v
		}
	} else {
		cfg.RateLimitRequests = 10.0
	}

	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		if v, err := strconv.Atoi(burst); err == nil {
			cfg.RateLimitBurst = v
		}
	} else {
		cfg.RateLimitBurst = 20
	}

	// Domain Manager / DNS Guide Configuration
	cfg.SMTPHostname = os.Getenv("SMTP_HOSTNAME")
	if cfg.SMTPHostname == "" {
		cfg.SMTPHostname = "mail.infinimail.local"
	}

	cfg.ServerIP = os.Getenv("SERVER_IP")
	if cfg.ServerIP == "" {
		cfg.ServerIP = "127.0.0.1"
	}

	// ACME / Certificate Configuration
	cfg.ACMEDirectoryURL = os.Getenv("ACME_DIRECTORY_URL")
	if cfg.ACMEDirectoryURL == "" {
		cfg.ACMEDirectoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	}

	cfg.ACMEEmail = os.Getenv("ACME_EMAIL")

	acmeStaging := os.Getenv("ACME_STAGING")
	if acmeStaging == "" {
		cfg.ACMEStaging = true
	} else {
		staging, err := strconv.ParseBool(acmeStaging)
		if err != nil {
			return nil, fmt.Errorf("ACME_STAGING must be a valid boolean: %w", err)
		}
		cfg.ACMEStaging = staging
	}

	cfg.CertStoragePath = os.Getenv("CERT_STORAGE_PATH")
	if cfg.CertStoragePath == "" {
		cfg.CertStoragePath = "./certs"
	}

	certRenewalDays := os.Getenv("CERT_RENEWAL_DAYS")
	if certRenewalDays == "" {
		cfg.CertRenewalDays = 30
	} else {
		days, err := strconv.Atoi(certRenewalDays)
		if err != nil {
			return nil, fmt.Errorf("CERT_RENEWAL_DAYS must be a valid integer: %w", err)
		}
		cfg.CertRenewalDays = days
	}

	cfg.CertRenewalCheckInterval = os.Getenv("CERT_RENEWAL_CHECK_INTERVAL")
	if cfg.CertRenewalCheckInterval == "" {
		cfg.CertRenewalCheckInterval = "24h"
	}

	return cfg, nil
}

// LoadWithValidation loads and validates configuration, failing fast on errors
func LoadWithValidation() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Production-specific validation
	if cfg.AppEnv == "production" {
		if err := cfg.ValidateProduction(); err != nil {
			return nil, err
		}
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

// ValidateProduction performs additional validation for production environment
func (c *Config) ValidateProduction() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required in production")
	}

	if c.AllowedOrigins == "" {
		return fmt.Errorf("ALLOWED_ORIGINS is required in production")
	}

	// Check for wildcard in production
	if strings.Contains(c.AllowedOrigins, "*") {
		return fmt.Errorf("wildcard (*) origins are not allowed in production")
	}

	// Check for sslmode=disable in database URL
	if strings.Contains(c.DatabaseURL, "sslmode=disable") {
		return fmt.Errorf("sslmode=disable is not allowed in production")
	}

	return nil
}

// LogConfig logs configuration values (excluding secrets)
func (c *Config) LogConfig(logger *slog.Logger) {
	logger.Info("configuration loaded",
		slog.Int("api_port", c.APIPort),
		slog.Int("smtp_port", c.SMTPPort),
		slog.Bool("auto_provisioning", c.AutoProvisioningEnabled),
		slog.String("storage_path", c.AttachmentStoragePath),
		slog.String("log_level", c.LogLevel),
		slog.String("app_env", c.AppEnv),
		slog.Bool("api_key_set", c.APIKey != ""),
		slog.Bool("allowed_origins_set", c.AllowedOrigins != ""),
		slog.Float64("rate_limit_rps", c.RateLimitRequests),
		slog.Int("rate_limit_burst", c.RateLimitBurst),
		slog.String("smtp_hostname", c.SMTPHostname),
		slog.String("server_ip", c.ServerIP),
		slog.String("acme_directory_url", c.ACMEDirectoryURL),
		slog.Bool("acme_email_set", c.ACMEEmail != ""),
		slog.Bool("acme_staging", c.ACMEStaging),
		slog.String("cert_storage_path", c.CertStoragePath),
		slog.Int("cert_renewal_days", c.CertRenewalDays),
		slog.String("cert_renewal_check_interval", c.CertRenewalCheckInterval),
	)
}
