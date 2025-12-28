package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/gorm"
)

// MailboxRepository defines the interface for mailbox data access
type MailboxRepository interface {
	Create(ctx context.Context, mailbox *models.Mailbox) error
	GetByID(ctx context.Context, id uint) (*models.Mailbox, error)
	GetByAddress(ctx context.Context, fullAddress string) (*models.Mailbox, error)
	GetOrCreate(ctx context.Context, localPart string, domainID uint, domainName string) (*models.Mailbox, bool, error)
	ListByDomain(ctx context.Context, domainID uint, limit, offset int) ([]models.MailboxWithUnreadCount, int64, error)
	UpdateLastAccessed(ctx context.Context, id uint) error
	Delete(ctx context.Context, id uint) error
}

// mailboxRepository implements MailboxRepository using GORM
type mailboxRepository struct {
	db *gorm.DB
}

// NewMailboxRepository creates a new MailboxRepository instance
func NewMailboxRepository(db *gorm.DB) MailboxRepository {
	return &mailboxRepository{db: db}
}

// Create creates a new mailbox
func (r *mailboxRepository) Create(ctx context.Context, mailbox *models.Mailbox) error {
	result := r.db.WithContext(ctx).Create(mailbox)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return fmt.Errorf("mailbox with address '%s' already exists: %w", mailbox.FullAddress, ErrDuplicateEntry)
		}
		return fmt.Errorf("failed to create mailbox: %w", result.Error)
	}
	return nil
}

// GetByID retrieves a mailbox by its ID
func (r *mailboxRepository) GetByID(ctx context.Context, id uint) (*models.Mailbox, error) {
	var mailbox models.Mailbox
	result := r.db.WithContext(ctx).First(&mailbox, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get mailbox by ID: %w", result.Error)
	}
	return &mailbox, nil
}

// GetByAddress retrieves a mailbox by its full email address
func (r *mailboxRepository) GetByAddress(ctx context.Context, fullAddress string) (*models.Mailbox, error) {
	var mailbox models.Mailbox
	result := r.db.WithContext(ctx).Where("full_address = ?", fullAddress).First(&mailbox)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get mailbox by address: %w", result.Error)
	}
	return &mailbox, nil
}

// GetOrCreate retrieves a mailbox by address or creates it if it doesn't exist
// Returns the mailbox, a boolean indicating if it was created, and any error
func (r *mailboxRepository) GetOrCreate(ctx context.Context, localPart string, domainID uint, domainName string) (*models.Mailbox, bool, error) {
	fullAddress := fmt.Sprintf("%s@%s", localPart, domainName)
	
	// Try to find existing mailbox
	mailbox, err := r.GetByAddress(ctx, fullAddress)
	if err == nil {
		return mailbox, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	
	// Create new mailbox
	mailbox = &models.Mailbox{
		LocalPart:   localPart,
		DomainID:    domainID,
		FullAddress: fullAddress,
	}
	
	if err := r.Create(ctx, mailbox); err != nil {
		// Handle race condition - another request might have created it
		if errors.Is(err, ErrDuplicateEntry) {
			mailbox, err = r.GetByAddress(ctx, fullAddress)
			if err != nil {
				return nil, false, err
			}
			return mailbox, false, nil
		}
		return nil, false, err
	}
	
	return mailbox, true, nil
}

// ListByDomain retrieves all mailboxes for a domain with pagination and unread count
func (r *mailboxRepository) ListByDomain(ctx context.Context, domainID uint, limit, offset int) ([]models.MailboxWithUnreadCount, int64, error) {
	var total int64
	
	// Count total mailboxes for this domain
	if err := r.db.WithContext(ctx).Model(&models.Mailbox{}).Where("domain_id = ?", domainID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count mailboxes: %w", err)
	}
	
	// Get mailboxes with unread count
	var results []models.MailboxWithUnreadCount
	
	query := `
		SELECT 
			m.*,
			COALESCE((SELECT COUNT(*) FROM messages msg WHERE msg.mailbox_id = m.id AND msg.is_read = false), 0) as unread_count
		FROM mailboxes m
		WHERE m.domain_id = ?
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`
	
	if err := r.db.WithContext(ctx).Raw(query, domainID, limit, offset).Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list mailboxes: %w", err)
	}
	
	return results, total, nil
}

// UpdateLastAccessed updates the last_accessed_at timestamp for a mailbox
func (r *mailboxRepository) UpdateLastAccessed(ctx context.Context, id uint) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Mailbox{}).Where("id = ?", id).Update("last_accessed_at", now)
	if result.Error != nil {
		return fmt.Errorf("failed to update last accessed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a mailbox by its ID (cascade deletes messages and attachments)
func (r *mailboxRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Mailbox{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete mailbox: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
