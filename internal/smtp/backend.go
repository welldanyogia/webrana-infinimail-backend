package smtp

import (
	"crypto/tls"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/websocket"
)

// Security limits
const (
	DefaultMaxMessageSize  = 25 * 1024 * 1024 // 25 MB
	DefaultMaxRecipients   = 100
	DefaultReadTimeout     = 60 * time.Second
	DefaultWriteTimeout    = 60 * time.Second
	DefaultMaxIdleSeconds  = 300
	DefaultMaxLineLength   = 2000
)

// Backend implements the go-smtp Backend interface
type Backend struct {
	domainRepo     repository.DomainRepository
	mailboxRepo    repository.MailboxRepository
	messageRepo    repository.MessageRepository
	attachmentRepo repository.AttachmentRepository
	fileStorage    storage.FileStorage
	wsHub          *websocket.Hub
	autoProvision  bool
	logger         *slog.Logger
}

// BackendConfig holds configuration for the SMTP backend
type BackendConfig struct {
	DomainRepo     repository.DomainRepository
	MailboxRepo    repository.MailboxRepository
	MessageRepo    repository.MessageRepository
	AttachmentRepo repository.AttachmentRepository
	FileStorage    storage.FileStorage
	WSHub          *websocket.Hub
	AutoProvision  bool
	Logger         *slog.Logger
}

// NewBackend creates a new SMTP backend
func NewBackend(cfg *BackendConfig) *Backend {
	return &Backend{
		domainRepo:     cfg.DomainRepo,
		mailboxRepo:    cfg.MailboxRepo,
		messageRepo:    cfg.MessageRepo,
		attachmentRepo: cfg.AttachmentRepo,
		fileStorage:    cfg.FileStorage,
		wsHub:          cfg.WSHub,
		autoProvision:  cfg.AutoProvision,
		logger:         cfg.Logger,
	}
}

// NewSession creates a new SMTP session
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	if b.logger != nil {
		b.logger.Info("new SMTP connection", slog.String("remote_addr", c.Conn().RemoteAddr().String()))
	}
	return NewSession(b), nil
}

// ServerConfig holds security configuration for the SMTP server
type ServerConfig struct {
	Addr            string
	Domain          string
	MaxMessageSize  int64
	MaxRecipients   int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	AllowInsecure   bool
	TLSConfig       *tls.Config
}

// NewSecureServer creates a new SMTP server with security settings
func NewSecureServer(backend *Backend, cfg *ServerConfig) *smtp.Server {
	s := smtp.NewServer(backend)

	s.Addr = cfg.Addr
	s.Domain = cfg.Domain

	// Set message size limit
	if cfg.MaxMessageSize > 0 {
		s.MaxMessageBytes = cfg.MaxMessageSize
	} else {
		s.MaxMessageBytes = DefaultMaxMessageSize
	}

	// Set recipient limit
	if cfg.MaxRecipients > 0 {
		s.MaxRecipients = cfg.MaxRecipients
	} else {
		s.MaxRecipients = DefaultMaxRecipients
	}

	// Set timeouts
	if cfg.ReadTimeout > 0 {
		s.ReadTimeout = cfg.ReadTimeout
	} else {
		s.ReadTimeout = DefaultReadTimeout
	}

	if cfg.WriteTimeout > 0 {
		s.WriteTimeout = cfg.WriteTimeout
	} else {
		s.WriteTimeout = DefaultWriteTimeout
	}

	// Disable insecure authentication by default
	s.AllowInsecureAuth = cfg.AllowInsecure

	// Configure TLS if provided
	if cfg.TLSConfig != nil {
		s.TLSConfig = cfg.TLSConfig
	}

	// Set max line length to prevent buffer overflow attacks
	s.MaxLineLength = DefaultMaxLineLength

	return s
}

// LoadServerConfigFromEnv loads server configuration from environment variables
func LoadServerConfigFromEnv() *ServerConfig {
	cfg := &ServerConfig{
		Addr:           getEnvOrDefault("SMTP_ADDR", ":2525"),
		Domain:         getEnvOrDefault("SMTP_DOMAIN", "localhost"),
		AllowInsecure:  getEnvBool("SMTP_ALLOW_INSECURE", false),
	}

	if maxSize := os.Getenv("SMTP_MAX_MESSAGE_SIZE"); maxSize != "" {
		if size, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			cfg.MaxMessageSize = size
		}
	}

	if maxRecip := os.Getenv("SMTP_MAX_RECIPIENTS"); maxRecip != "" {
		if recip, err := strconv.Atoi(maxRecip); err == nil {
			cfg.MaxRecipients = recip
		}
	}

	if readTimeout := os.Getenv("SMTP_READ_TIMEOUT"); readTimeout != "" {
		if timeout, err := time.ParseDuration(readTimeout); err == nil {
			cfg.ReadTimeout = timeout
		}
	}

	if writeTimeout := os.Getenv("SMTP_WRITE_TIMEOUT"); writeTimeout != "" {
		if timeout, err := time.ParseDuration(writeTimeout); err == nil {
			cfg.WriteTimeout = timeout
		}
	}

	// Load TLS configuration if certificate and key are provided
	certFile := os.Getenv("SMTP_TLS_CERT")
	keyFile := os.Getenv("SMTP_TLS_KEY")
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			cfg.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}
		}
	}

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
