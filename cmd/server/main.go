package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/config"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/database"
	seclogger "github.com/welldanyogia/webrana-infinimail-backend/internal/logger"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/services"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/smtp"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	ws "github.com/welldanyogia/webrana-infinimail-backend/internal/websocket"
)

func main() {
	// Setup logger
	logLevel := slog.LevelInfo
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Update log level from config
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting Infinimail backend",
		slog.Int("api_port", cfg.APIPort),
		slog.Int("smtp_port", cfg.SMTPPort))

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		logger.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database migrations completed")

	// Initialize file storage
	fileStorage, err := storage.NewLocalStorage(cfg.AttachmentStoragePath)
	if err != nil {
		logger.Error("failed to initialize file storage", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize WebSocket hub
	wsHub := ws.NewHub(logger)
	go wsHub.Run()

	// Initialize security logger
	securityLogger := seclogger.NewSecurityLogger()

	// Initialize repositories
	domainRepo := repository.NewDomainRepository(db)
	mailboxRepo := repository.NewMailboxRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db, fileStorage)

	// Parse allowed origins for CORS and WebSocket
	var allowedOrigins []string
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}

	// Initialize HTTP router with security configuration
	router := api.NewRouter(&api.RouterConfig{
		DB:             db,
		FileStorage:    fileStorage,
		Logger:         logger,
		APIKey:         cfg.APIKey,
		AllowedOrigins: allowedOrigins,
		RateLimit:      int(cfg.RateLimitRequests),
		RateBurst:      cfg.RateLimitBurst,
		EnableAuth:     cfg.APIKey != "",
	})

	// Create secure WebSocket upgrader
	upgrader := ws.NewSecureUpgrader(logger)

	router.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			securityLogger.SuspiciousActivity(c.RealIP(), "/ws", "websocket_upgrade_failed")
			logger.Error("websocket upgrade failed", slog.Any("error", err))
			return err
		}

		client := ws.NewClient(wsHub, conn, logger)
		wsHub.Register(client)

		go client.WritePump()
		go client.ReadPump()

		return nil
	})

	// Initialize SMTP server with security configuration
	smtpBackend := smtp.NewBackend(&smtp.BackendConfig{
		DomainRepo:     domainRepo,
		MailboxRepo:    mailboxRepo,
		MessageRepo:    messageRepo,
		AttachmentRepo: attachmentRepo,
		FileStorage:    fileStorage,
		WSHub:          wsHub,
		AutoProvision:  cfg.AutoProvisioningEnabled,
		Logger:         logger,
	})

	// Load SMTP security configuration from environment
	smtpConfig := smtp.LoadServerConfigFromEnv()
	smtpConfig.Addr = fmt.Sprintf(":%d", cfg.SMTPPort)

	// Initialize certificate store for SNI support
	var certStore services.CertificateStore
	certRepo := repository.NewCertificateRepository(db)
	
	// Create certificate storage
	certStorage, err := services.NewCertStorage(services.CertStorageConfig{
		BasePath: cfg.CertStoragePath,
	})
	if err != nil {
		logger.Warn("failed to initialize certificate storage, SNI disabled", slog.Any("error", err))
	} else {
		// Create certificate store
		certStore, err = services.NewCertificateStore(services.CertificateStoreConfig{
			CertStorage: certStorage,
			CertRepo:    certRepo,
		})
		if err != nil {
			logger.Warn("failed to initialize certificate store, SNI disabled", slog.Any("error", err))
		} else {
			// Load all certificates from database
			ctx := context.Background()
			if err := certStore.ReloadAll(ctx); err != nil {
				logger.Warn("failed to load certificates", slog.Any("error", err))
			} else {
				logger.Info("certificate store initialized", slog.Int("certificates_loaded", certStore.Count()))
			}
			
			// Set the GetCertificate function for SNI support
			smtpConfig.GetCertificate = certStore.GetCertificateFunc()
		}
	}

	smtpServer := smtp.NewSecureServer(smtpBackend, smtpConfig)

	logger.Info("SMTP server configured",
		slog.Int64("max_message_bytes", smtpServer.MaxMessageBytes),
		slog.Int("max_recipients", smtpServer.MaxRecipients),
		slog.Bool("allow_insecure_auth", smtpServer.AllowInsecureAuth),
		slog.Bool("sni_enabled", certStore != nil))

	// Start servers
	errChan := make(chan error, 2)

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.APIPort)
		logger.Info("starting HTTP server", slog.String("addr", addr))
		if err := router.Start(addr); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Start SMTP server
	go func() {
		logger.Info("starting SMTP server", slog.String("addr", smtpServer.Addr))
		if err := smtpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("SMTP server error: %w", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("server error", slog.Any("error", err))
	case sig := <-quit:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Info("shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := router.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.Any("error", err))
	}

	// Shutdown SMTP server
	if err := smtpServer.Close(); err != nil {
		logger.Error("SMTP server shutdown error", slog.Any("error", err))
	}

	// Close database connection
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	logger.Info("servers stopped")
}
