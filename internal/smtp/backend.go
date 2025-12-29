package smtp

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"strconv"
	"sync"
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
	// SNI Support
	GetCertificate  func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	DefaultCertFile string
	DefaultKeyFile  string
}

// SecureSMTPServer wraps smtp.Server with certificate hot reload support
type SecureSMTPServer struct {
	*smtp.Server
	mu              sync.RWMutex
	getCertificate  func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	defaultCert     *tls.Certificate
	logger          *slog.Logger
}

// NewSecureServer creates a new SMTP server with security settings and SNI support
func NewSecureServer(backend *Backend, cfg *ServerConfig) *SecureSMTPServer {
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

	// Set max line length to prevent buffer overflow attacks
	s.MaxLineLength = DefaultMaxLineLength

	// Create secure server wrapper
	secureServer := &SecureSMTPServer{
		Server:         s,
		getCertificate: cfg.GetCertificate,
		logger:         backend.logger,
	}

	// Load default certificate if provided
	if cfg.DefaultCertFile != "" && cfg.DefaultKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.DefaultCertFile, cfg.DefaultKeyFile)
		if err == nil {
			secureServer.defaultCert = &cert
		} else if backend.logger != nil {
			backend.logger.Warn("failed to load default certificate", slog.Any("error", err))
		}
	}

	// Configure TLS with SNI support
	if cfg.TLSConfig != nil {
		s.TLSConfig = cfg.TLSConfig
	} else {
		// Create TLS config with SNI support
		s.TLSConfig = secureServer.createTLSConfig()
	}

	return secureServer
}

// createTLSConfig creates a TLS configuration with SNI support
func (s *SecureSMTPServer) createTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: s.getCertificateWithFallback,
	}
}

// getCertificateWithFallback returns the appropriate certificate for the SNI hostname
// Falls back to default certificate if no match is found
func (s *SecureSMTPServer) getCertificateWithFallback(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Try the configured GetCertificate function first
	if s.getCertificate != nil {
		cert, err := s.getCertificate(hello)
		if err == nil && cert != nil {
			return cert, nil
		}
		// Log the error but continue to fallback
		if s.logger != nil && err != nil {
			s.logger.Debug("SNI certificate lookup failed, using fallback",
				slog.String("server_name", hello.ServerName),
				slog.Any("error", err))
		}
	}

	// Fallback to default certificate
	s.mu.RLock()
	defaultCert := s.defaultCert
	s.mu.RUnlock()

	if defaultCert != nil {
		return defaultCert, nil
	}

	// No certificate available
	return nil, nil
}

// SetGetCertificateFunc sets the function used to retrieve certificates for SNI
func (s *SecureSMTPServer) SetGetCertificateFunc(fn func(*tls.ClientHelloInfo) (*tls.Certificate, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCertificate = fn
}

// SetDefaultCertificate sets the default certificate to use when no SNI match is found
func (s *SecureSMTPServer) SetDefaultCertificate(cert *tls.Certificate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultCert = cert
}

// LoadDefaultCertificate loads a default certificate from files
func (s *SecureSMTPServer) LoadDefaultCertificate(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	s.SetDefaultCertificate(&cert)
	return nil
}

// ReloadCertificates triggers a reload of all certificates
// This is called after new certificates are generated
func (s *SecureSMTPServer) ReloadCertificates(ctx context.Context, reloadFunc func(context.Context) error) error {
	if reloadFunc == nil {
		return nil
	}
	
	if err := reloadFunc(ctx); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to reload certificates", slog.Any("error", err))
		}
		return err
	}
	
	if s.logger != nil {
		s.logger.Info("certificates reloaded successfully")
	}
	return nil
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

	// Load default certificate paths for fallback
	cfg.DefaultCertFile = os.Getenv("SMTP_TLS_CERT")
	cfg.DefaultKeyFile = os.Getenv("SMTP_TLS_KEY")

	// Legacy TLS configuration (used if no SNI GetCertificate is provided)
	certFile := cfg.DefaultCertFile
	keyFile := cfg.DefaultKeyFile
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
