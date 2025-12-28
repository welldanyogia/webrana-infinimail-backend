package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"gorm.io/gorm"
)

// AttachmentRepository defines the interface for attachment data access
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *models.Attachment) error
	GetByID(ctx context.Context, id uint) (*models.Attachment, error)
	ListByMessage(ctx context.Context, messageID uint) ([]models.Attachment, error)
	Delete(ctx context.Context, id uint) error
}

// attachmentRepository implements AttachmentRepository using GORM
type attachmentRepository struct {
	db          *gorm.DB
	fileStorage storage.FileStorage
}

// NewAttachmentRepository creates a new AttachmentRepository instance
func NewAttachmentRepository(db *gorm.DB, fileStorage storage.FileStorage) AttachmentRepository {
	return &attachmentRepository{
		db:          db,
		fileStorage: fileStorage,
	}
}

// Create creates a new attachment record
func (r *attachmentRepository) Create(ctx context.Context, attachment *models.Attachment) error {
	result := r.db.WithContext(ctx).Create(attachment)
	if result.Error != nil {
		return fmt.Errorf("failed to create attachment: %w", result.Error)
	}
	return nil
}

// GetByID retrieves an attachment by its ID
func (r *attachmentRepository) GetByID(ctx context.Context, id uint) (*models.Attachment, error) {
	var attachment models.Attachment
	result := r.db.WithContext(ctx).First(&attachment, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get attachment by ID: %w", result.Error)
	}
	return &attachment, nil
}

// ListByMessage retrieves all attachments for a message
func (r *attachmentRepository) ListByMessage(ctx context.Context, messageID uint) ([]models.Attachment, error) {
	var attachments []models.Attachment
	result := r.db.WithContext(ctx).Where("message_id = ?", messageID).Find(&attachments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", result.Error)
	}
	return attachments, nil
}

// Delete deletes an attachment by its ID and removes the associated file
func (r *attachmentRepository) Delete(ctx context.Context, id uint) error {
	// Get attachment first to get file path
	attachment, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from database
	result := r.db.WithContext(ctx).Delete(&models.Attachment{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete attachment: %w", result.Error)
	}

	// Delete associated file (ignore errors as file might already be deleted)
	if attachment.FilePath != "" && r.fileStorage != nil {
		_ = r.fileStorage.Delete(attachment.FilePath)
	}

	return nil
}
