package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/gorm"
)

// MessageRepository defines the interface for message data access
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	CreateWithAttachments(ctx context.Context, message *models.Message, attachments []models.Attachment) error
	GetByID(ctx context.Context, id uint) (*models.Message, error)
	ListByMailbox(ctx context.Context, mailboxID uint, limit, offset int) ([]models.MessageListItem, int64, error)
	MarkAsRead(ctx context.Context, id uint) error
	Delete(ctx context.Context, id uint) error
	CountUnread(ctx context.Context, mailboxID uint) (int64, error)
}

// messageRepository implements MessageRepository using GORM
type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository creates a new MessageRepository instance
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Create creates a new message
func (r *messageRepository) Create(ctx context.Context, message *models.Message) error {
	result := r.db.WithContext(ctx).Create(message)
	if result.Error != nil {
		return fmt.Errorf("failed to create message: %w", result.Error)
	}
	return nil
}

// CreateWithAttachments creates a message with its attachments in a transaction
func (r *messageRepository) CreateWithAttachments(ctx context.Context, message *models.Message, attachments []models.Attachment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create message first
		if err := tx.Create(message).Error; err != nil {
			return fmt.Errorf("failed to create message: %w", err)
		}

		// Create attachments with message ID
		for i := range attachments {
			attachments[i].MessageID = message.ID
			if err := tx.Create(&attachments[i]).Error; err != nil {
				return fmt.Errorf("failed to create attachment: %w", err)
			}
		}

		return nil
	})
}

// GetByID retrieves a message by its ID with preloaded attachments
func (r *messageRepository) GetByID(ctx context.Context, id uint) (*models.Message, error) {
	var message models.Message
	result := r.db.WithContext(ctx).Preload("Attachments").First(&message, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get message by ID: %w", result.Error)
	}
	return &message, nil
}

// ListByMailbox retrieves messages for a mailbox with pagination, ordered by received_at descending
func (r *messageRepository) ListByMailbox(ctx context.Context, mailboxID uint, limit, offset int) ([]models.MessageListItem, int64, error) {
	var total int64

	// Count total messages for this mailbox
	if err := r.db.WithContext(ctx).Model(&models.Message{}).Where("mailbox_id = ?", mailboxID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Get messages with attachment count
	var results []models.MessageListItem

	query := `
		SELECT 
			m.id,
			m.mailbox_id,
			m.sender_email,
			m.sender_name,
			m.subject,
			m.snippet,
			m.is_read,
			m.received_at,
			COALESCE((SELECT COUNT(*) FROM attachments a WHERE a.message_id = m.id), 0) as attachment_count
		FROM messages m
		WHERE m.mailbox_id = ?
		ORDER BY m.received_at DESC
		LIMIT ? OFFSET ?
	`

	if err := r.db.WithContext(ctx).Raw(query, mailboxID, limit, offset).Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list messages: %w", err)
	}

	return results, total, nil
}

// MarkAsRead marks a message as read
func (r *messageRepository) MarkAsRead(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&models.Message{}).Where("id = ?", id).Update("is_read", true)
	if result.Error != nil {
		return fmt.Errorf("failed to mark message as read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a message by its ID (cascade deletes attachments)
func (r *messageRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Message{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CountUnread counts unread messages for a mailbox
func (r *messageRepository) CountUnread(ctx context.Context, mailboxID uint) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Message{}).Where("mailbox_id = ? AND is_read = ?", mailboxID, false).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", result.Error)
	}
	return count, nil
}
