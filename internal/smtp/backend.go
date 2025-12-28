package smtp

import (
	"log/slog"

	"github.com/emersion/go-smtp"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/websocket"
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
