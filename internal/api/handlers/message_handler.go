package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// MessageHandler handles message-related HTTP requests
type MessageHandler struct {
	messageRepo repository.MessageRepository
	mailboxRepo repository.MailboxRepository
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(messageRepo repository.MessageRepository, mailboxRepo repository.MailboxRepository) *MessageHandler {
	return &MessageHandler{
		messageRepo: messageRepo,
		mailboxRepo: mailboxRepo,
	}
}

// List handles GET /api/mailboxes/:mailbox_id/messages
func (h *MessageHandler) List(c echo.Context) error {
	mailboxID, err := strconv.ParseUint(c.Param("mailbox_id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid mailbox ID")
	}

	// Verify mailbox exists
	_, err = h.mailboxRepo.GetByID(c.Request().Context(), uint(mailboxID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "mailbox not found")
		}
		return response.InternalError(c, "failed to get mailbox")
	}

	limit := 20
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, total, err := h.messageRepo.ListByMailbox(c.Request().Context(), uint(mailboxID), limit, offset)
	if err != nil {
		return response.InternalError(c, "failed to list messages")
	}

	return response.Paginated(c, messages, total, limit, offset)
}

// Get handles GET /api/messages/:id
func (h *MessageHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid message ID")
	}

	message, err := h.messageRepo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "message not found")
		}
		return response.InternalError(c, "failed to get message")
	}

	// Auto mark as read
	if !message.IsRead {
		_ = h.messageRepo.MarkAsRead(c.Request().Context(), uint(id))
		message.IsRead = true
	}

	return response.Success(c, message)
}

// MarkAsRead handles PATCH /api/messages/:id/read
func (h *MessageHandler) MarkAsRead(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid message ID")
	}

	if err := h.messageRepo.MarkAsRead(c.Request().Context(), uint(id)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "message not found")
		}
		return response.InternalError(c, "failed to mark message as read")
	}

	return response.SuccessWithMessage(c, nil, "message marked as read")
}

// Delete handles DELETE /api/messages/:id
func (h *MessageHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid message ID")
	}

	if err := h.messageRepo.Delete(c.Request().Context(), uint(id)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "message not found")
		}
		return response.InternalError(c, "failed to delete message")
	}

	return response.NoContent(c)
}
