package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
)

// MockDomainRepository implements repository.DomainRepository
type MockDomainRepository struct {
	mock.Mock
}

// Create creates a new domain
func (m *MockDomainRepository) Create(ctx context.Context, domain *models.Domain) error {
	args := m.Called(ctx, domain)
	return args.Error(0)
}

// GetByID retrieves a domain by its ID
func (m *MockDomainRepository) GetByID(ctx context.Context, id uint) (*models.Domain, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Domain), args.Error(1)
}

// GetByName retrieves a domain by its name
func (m *MockDomainRepository) GetByName(ctx context.Context, name string) (*models.Domain, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Domain), args.Error(1)
}

// List retrieves all domains, optionally filtering by active status
func (m *MockDomainRepository) List(ctx context.Context, activeOnly bool) ([]models.Domain, error) {
	args := m.Called(ctx, activeOnly)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Domain), args.Error(1)
}


// Update updates an existing domain
func (m *MockDomainRepository) Update(ctx context.Context, domain *models.Domain) error {
	args := m.Called(ctx, domain)
	return args.Error(0)
}

// Delete deletes a domain by its ID
func (m *MockDomainRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockMailboxRepository implements repository.MailboxRepository
type MockMailboxRepository struct {
	mock.Mock
}

// Create creates a new mailbox
func (m *MockMailboxRepository) Create(ctx context.Context, mailbox *models.Mailbox) error {
	args := m.Called(ctx, mailbox)
	return args.Error(0)
}

// GetByID retrieves a mailbox by its ID
func (m *MockMailboxRepository) GetByID(ctx context.Context, id uint) (*models.Mailbox, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Mailbox), args.Error(1)
}

// GetByAddress retrieves a mailbox by its full email address
func (m *MockMailboxRepository) GetByAddress(ctx context.Context, fullAddress string) (*models.Mailbox, error) {
	args := m.Called(ctx, fullAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Mailbox), args.Error(1)
}

// GetOrCreate retrieves a mailbox by address or creates it if it doesn't exist
func (m *MockMailboxRepository) GetOrCreate(ctx context.Context, localPart string, domainID uint, domainName string) (*models.Mailbox, bool, error) {
	args := m.Called(ctx, localPart, domainID, domainName)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*models.Mailbox), args.Bool(1), args.Error(2)
}


// ListByDomain retrieves all mailboxes for a domain with pagination and unread count
func (m *MockMailboxRepository) ListByDomain(ctx context.Context, domainID uint, limit, offset int) ([]models.MailboxWithUnreadCount, int64, error) {
	args := m.Called(ctx, domainID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.MailboxWithUnreadCount), args.Get(1).(int64), args.Error(2)
}

// UpdateLastAccessed updates the last_accessed_at timestamp for a mailbox
func (m *MockMailboxRepository) UpdateLastAccessed(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Delete deletes a mailbox by its ID
func (m *MockMailboxRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockMessageRepository implements repository.MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

// Create creates a new message
func (m *MockMessageRepository) Create(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// CreateWithAttachments creates a message with its attachments in a transaction
func (m *MockMessageRepository) CreateWithAttachments(ctx context.Context, message *models.Message, attachments []models.Attachment) error {
	args := m.Called(ctx, message, attachments)
	return args.Error(0)
}

// GetByID retrieves a message by its ID with preloaded attachments
func (m *MockMessageRepository) GetByID(ctx context.Context, id uint) (*models.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}


// ListByMailbox retrieves messages for a mailbox with pagination
func (m *MockMessageRepository) ListByMailbox(ctx context.Context, mailboxID uint, limit, offset int) ([]models.MessageListItem, int64, error) {
	args := m.Called(ctx, mailboxID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.MessageListItem), args.Get(1).(int64), args.Error(2)
}

// MarkAsRead marks a message as read
func (m *MockMessageRepository) MarkAsRead(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Delete deletes a message by its ID
func (m *MockMessageRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// CountUnread counts unread messages for a mailbox
func (m *MockMessageRepository) CountUnread(ctx context.Context, mailboxID uint) (int64, error) {
	args := m.Called(ctx, mailboxID)
	return args.Get(0).(int64), args.Error(1)
}

// MockAttachmentRepository implements repository.AttachmentRepository
type MockAttachmentRepository struct {
	mock.Mock
}

// Create creates a new attachment record
func (m *MockAttachmentRepository) Create(ctx context.Context, attachment *models.Attachment) error {
	args := m.Called(ctx, attachment)
	return args.Error(0)
}

// GetByID retrieves an attachment by its ID
func (m *MockAttachmentRepository) GetByID(ctx context.Context, id uint) (*models.Attachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Attachment), args.Error(1)
}

// ListByMessage retrieves all attachments for a message
func (m *MockAttachmentRepository) ListByMessage(ctx context.Context, messageID uint) ([]models.Attachment, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Attachment), args.Error(1)
}

// Delete deletes an attachment by its ID
func (m *MockAttachmentRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
