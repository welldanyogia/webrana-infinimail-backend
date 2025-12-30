package api

import (
	"log/slog"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/handlers"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/middleware"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/services"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"gorm.io/gorm"
)

// RouterConfig holds dependencies for the router
type RouterConfig struct {
	DB          *gorm.DB
	FileStorage storage.FileStorage
	Logger      *slog.Logger
	// Security configuration
	APIKey         string   // API key for authentication (empty = disabled)
	AllowedOrigins []string // Allowed CORS origins
	RateLimit      int      // Requests per second (0 = use env default)
	RateBurst      int      // Burst size for rate limiter
	EnableAuth     bool     // Enable API key authentication
	// SSL Domain Setup services (optional)
	DomainManager  services.DomainManagerService
	DNSVerifier    services.DNSVerifierService
	DNSExporter    services.DNSExporter
	CertManager    services.CertificateManagerService
}

// NewRouter creates and configures the Echo router with all routes
func NewRouter(cfg *RouterConfig) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Security Middleware (applied in correct order)
	// 1. Recover from panics
	e.Use(middleware.Recover())

	// 2. Security headers (applied to all responses)
	e.Use(middleware.SecureHeaders())

	// 3. CORS - Set environment variable if origins provided in config
	if len(cfg.AllowedOrigins) > 0 {
		os.Setenv("ALLOWED_ORIGINS", strings.Join(cfg.AllowedOrigins, ","))
	}
	e.Use(middleware.SecureCORS())

	// 4. Rate limiting - use RateLimiterWithConfig if custom values provided
	if cfg.RateLimit > 0 {
		e.Use(middleware.RateLimiterWithConfig(float64(cfg.RateLimit), cfg.RateBurst, cfg.Logger))
	} else {
		e.Use(middleware.RateLimiter(cfg.Logger))
	}

	// 5. Request logging
	if cfg.Logger != nil {
		e.Use(middleware.RequestLogger(cfg.Logger))
	}

	// Initialize repositories
	domainRepo := repository.NewDomainRepository(cfg.DB)
	mailboxRepo := repository.NewMailboxRepository(cfg.DB)
	messageRepo := repository.NewMessageRepository(cfg.DB)
	attachmentRepo := repository.NewAttachmentRepository(cfg.DB, cfg.FileStorage)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(cfg.DB)
	mailboxHandler := handlers.NewMailboxHandler(mailboxRepo, domainRepo)
	messageHandler := handlers.NewMessageHandler(messageRepo, mailboxRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, messageRepo, cfg.FileStorage)

	// Initialize domain handler with optional SSL services
	var domainHandler *handlers.DomainHandler
	if cfg.DomainManager != nil {
		domainHandler = handlers.NewDomainHandlerWithServices(
			domainRepo,
			cfg.DomainManager,
			cfg.DNSVerifier,
			cfg.DNSExporter,
			cfg.CertManager,
		)
	} else {
		domainHandler = handlers.NewDomainHandler(domainRepo)
	}

	// Health routes (no auth required)
	e.GET("/health", healthHandler.Health)
	e.GET("/ready", healthHandler.Ready)

	// API routes
	api := e.Group("/api")

	// Apply API key authentication if enabled
	// Set API_KEY env var if provided in config
	if cfg.EnableAuth && cfg.APIKey != "" {
		os.Setenv("API_KEY", cfg.APIKey)
	}
	api.Use(middleware.APIKeyAuth(cfg.Logger))

	// Domain routes
	domains := api.Group("/domains")
	domains.POST("", domainHandler.Create)
	domains.GET("", domainHandler.List)
	domains.GET("/:id", domainHandler.Get)
	domains.PUT("/:id", domainHandler.Update)
	domains.DELETE("/:id", domainHandler.Delete)
	// SSL Domain Setup routes
	domains.GET("/:id/dns-guide", domainHandler.GetDNSGuide)
	domains.GET("/:id/dns-export", domainHandler.GetDNSExport)
	domains.POST("/:id/verify-dns", domainHandler.VerifyDNS)
	domains.POST("/:id/generate-cert", domainHandler.GenerateCertificate)
	domains.GET("/:id/status", domainHandler.GetStatus)
	domains.POST("/:id/retry", domainHandler.Retry)
	// Manual DNS Verification routes (ACME challenge flow)
	domains.POST("/:id/request-acme-challenge", domainHandler.RequestACMEChallenge)
	domains.POST("/:id/verify-acme-dns", domainHandler.VerifyACMEDNS)
	domains.POST("/:id/submit-acme-challenge", domainHandler.SubmitACMEChallenge)
	domains.GET("/:id/acme-status", domainHandler.GetACMEStatus)

	// Mailbox routes
	mailboxes := api.Group("/mailboxes")
	mailboxes.POST("", mailboxHandler.Create)
	mailboxes.POST("/random", mailboxHandler.CreateRandom)
	mailboxes.GET("", mailboxHandler.List)
	mailboxes.GET("/:id", mailboxHandler.Get)
	mailboxes.DELETE("/:id", mailboxHandler.Delete)

	// Message routes (nested under mailboxes)
	mailboxes.GET("/:mailbox_id/messages", messageHandler.List)

	// Message routes (standalone)
	messages := api.Group("/messages")
	messages.GET("/:id", messageHandler.Get)
	messages.PATCH("/:id/read", messageHandler.MarkAsRead)
	messages.DELETE("/:id", messageHandler.Delete)

	// Attachment routes (nested under messages)
	messages.GET("/:message_id/attachments", attachmentHandler.List)

	// Attachment routes (standalone)
	attachments := api.Group("/attachments")
	attachments.GET("/:id", attachmentHandler.Get)
	attachments.GET("/:id/download", attachmentHandler.Download)

	// ACME Log routes (for debugging certificate generation)
	acmeLogHandler := handlers.NewACMELogHandler()
	// JSON API endpoints
	acmeLogs := api.Group("/acme/logs")
	acmeLogs.GET("", acmeLogHandler.ListLogs)
	acmeLogs.GET("/:domain", acmeLogHandler.GetDomainLog)
	// Browser-friendly HTML endpoints (no auth required for easy access)
	e.GET("/acme/logs", acmeLogHandler.ViewLogsHTML)
	e.GET("/acme/logs/:domain", acmeLogHandler.ViewDomainLogHTML)

	return e
}
