package api

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/handlers"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/middleware"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"gorm.io/gorm"
)

// RouterConfig holds dependencies for the router
type RouterConfig struct {
	DB          *gorm.DB
	FileStorage storage.FileStorage
	Logger      *slog.Logger
}

// NewRouter creates and configures the Echo router with all routes
func NewRouter(cfg *RouterConfig) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
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
	domainHandler := handlers.NewDomainHandler(domainRepo)
	mailboxHandler := handlers.NewMailboxHandler(mailboxRepo, domainRepo)
	messageHandler := handlers.NewMessageHandler(messageRepo, mailboxRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, messageRepo, cfg.FileStorage)

	// Health routes
	e.GET("/health", healthHandler.Health)
	e.GET("/ready", healthHandler.Ready)

	// API routes
	api := e.Group("/api")

	// Domain routes
	domains := api.Group("/domains")
	domains.POST("", domainHandler.Create)
	domains.GET("", domainHandler.List)
	domains.GET("/:id", domainHandler.Get)
	domains.PUT("/:id", domainHandler.Update)
	domains.DELETE("/:id", domainHandler.Delete)

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

	return e
}
