package handlers

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
)

// AttachmentHandler handles attachment-related HTTP requests
type AttachmentHandler struct {
	attachmentRepo repository.AttachmentRepository
	messageRepo    repository.MessageRepository
	fileStorage    storage.FileStorage
}

// NewAttachmentHandler creates a new AttachmentHandler
func NewAttachmentHandler(
	attachmentRepo repository.AttachmentRepository,
	messageRepo repository.MessageRepository,
	fileStorage storage.FileStorage,
) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentRepo: attachmentRepo,
		messageRepo:    messageRepo,
		fileStorage:    fileStorage,
	}
}

// List handles GET /api/messages/:message_id/attachments
func (h *AttachmentHandler) List(c echo.Context) error {
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid message ID")
	}

	// Verify message exists
	_, err = h.messageRepo.GetByID(c.Request().Context(), uint(messageID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "message not found")
		}
		return api.InternalError(c, "failed to get message")
	}

	attachments, err := h.attachmentRepo.ListByMessage(c.Request().Context(), uint(messageID))
	if err != nil {
		return api.InternalError(c, "failed to list attachments")
	}

	return api.Success(c, attachments)
}

// Get handles GET /api/attachments/:id
func (h *AttachmentHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid attachment ID")
	}

	attachment, err := h.attachmentRepo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "attachment not found")
		}
		return api.InternalError(c, "failed to get attachment")
	}

	return api.Success(c, attachment)
}

// Download handles GET /api/attachments/:id/download
func (h *AttachmentHandler) Download(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid attachment ID")
	}

	attachment, err := h.attachmentRepo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "attachment not found")
		}
		return api.InternalError(c, "failed to get attachment")
	}

	// Get file from storage
	file, err := h.fileStorage.Get(attachment.FilePath)
	if err != nil {
		return api.InternalError(c, "failed to retrieve file")
	}
	defer file.Close()

	// Set headers for download
	c.Response().Header().Set("Content-Type", attachment.ContentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, attachment.Filename))
	if attachment.SizeBytes > 0 {
		c.Response().Header().Set("Content-Length", strconv.FormatInt(attachment.SizeBytes, 10))
	}

	// Stream file to response
	_, err = io.Copy(c.Response().Writer, file)
	if err != nil {
		return api.InternalError(c, "failed to send file")
	}

	return nil
}
