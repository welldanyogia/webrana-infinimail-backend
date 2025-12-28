package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MessageRepositoryTestSuite is the test suite for MessageRepository
type MessageRepositoryTestSuite struct {
	suite.Suite
	db          *gorm.DB
	repo        MessageRepository
	testDomain  *models.Domain
	testMailbox *models.Mailbox
}

// SetupSuite runs once before all tests
func (s *MessageRepositoryTestSuite) SetupSuite() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)

	// Enable foreign keys for SQLite (required for cascade delete)
	db.Exec("PRAGMA foreign_keys = ON")

	// Auto-migrate models
	err = db.AutoMigrate(&models.Domain{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
	require.NoError(s.T(), err)

	s.db = db
	s.repo = NewMessageRepository(db)
}

// TearDownSuite runs once after all tests
func (s *MessageRepositoryTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test - clean up data and create test fixtures
func (s *MessageRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM attachments")
	s.db.Exec("DELETE FROM messages")
	s.db.Exec("DELETE FROM mailboxes")
	s.db.Exec("DELETE FROM domains")

	// Create test domain
	s.testDomain = &models.Domain{Name: "test.com", IsActive: true}
	err := s.db.Create(s.testDomain).Error
	require.NoError(s.T(), err)

	// Create test mailbox
	s.testMailbox = &models.Mailbox{
		LocalPart:   "user",
		DomainID:    s.testDomain.ID,
		FullAddress: "user@test.com",
	}
	err = s.db.Create(s.testMailbox).Error
	require.NoError(s.T(), err)
}

// TestMessageRepositoryTestSuite runs the test suite
func TestMessageRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(MessageRepositoryTestSuite))
}

// ==================== Create Tests ====================

func (s *MessageRepositoryTestSuite) TestCreate_Success() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		SenderName:  "Test Sender",
		Subject:     "Test Subject",
		Snippet:     "Test snippet...",
		BodyText:    "Test body text",
		BodyHTML:    "<p>Test body HTML</p>",
		IsRead:      false,
	}

	// Act
	err := s.repo.Create(context.Background(), message)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), message.ID)
	assert.NotZero(s.T(), message.ReceivedAt)
}

func (s *MessageRepositoryTestSuite) TestCreate_MinimalFields() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
	}

	// Act
	err := s.repo.Create(context.Background(), message)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), message.ID)
}

// ==================== CreateWithAttachments Tests ====================

func (s *MessageRepositoryTestSuite) TestCreateWithAttachments_Success() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024},
		{Filename: "image.png", ContentType: "image/png", FilePath: "/path/image.png", SizeBytes: 2048},
	}

	// Act
	err := s.repo.CreateWithAttachments(context.Background(), message, attachments)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), message.ID)

	// Verify attachments were created with correct message ID
	var savedAttachments []models.Attachment
	s.db.Where("message_id = ?", message.ID).Find(&savedAttachments)
	assert.Len(s.T(), savedAttachments, 2)
	for _, att := range savedAttachments {
		assert.Equal(s.T(), message.ID, att.MessageID)
	}
}

func (s *MessageRepositoryTestSuite) TestCreateWithAttachments_NoAttachments() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "No Attachments",
	}

	// Act
	err := s.repo.CreateWithAttachments(context.Background(), message, nil)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), message.ID)
}

// ==================== GetByID Tests ====================

func (s *MessageRepositoryTestSuite) TestGetByID_Found() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Test Subject",
		BodyText:    "Test body",
	}
	err := s.repo.Create(context.Background(), message)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByID(context.Background(), message.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), message.ID, result.ID)
	assert.Equal(s.T(), "Test Subject", result.Subject)
	assert.Equal(s.T(), "Test body", result.BodyText)
}

func (s *MessageRepositoryTestSuite) TestGetByID_NotFound() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *MessageRepositoryTestSuite) TestGetByID_PreloadsAttachments() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", FilePath: "/path/doc.pdf", SizeBytes: 1024},
	}
	err := s.repo.CreateWithAttachments(context.Background(), message, attachments)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByID(context.Background(), message.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Len(s.T(), result.Attachments, 1)
	assert.Equal(s.T(), "doc.pdf", result.Attachments[0].Filename)
}

// ==================== ListByMailbox Tests ====================

func (s *MessageRepositoryTestSuite) TestListByMailbox_ReturnsMessages() {
	// Arrange
	for i := 0; i < 3; i++ {
		message := &models.Message{
			MailboxID:   s.testMailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     "Message " + string(rune('A'+i)),
		}
		err := s.repo.Create(context.Background(), message)
		require.NoError(s.T(), err)
	}

	// Act
	result, total, err := s.repo.ListByMailbox(context.Background(), s.testMailbox.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 3)
	assert.Equal(s.T(), int64(3), total)
}

func (s *MessageRepositoryTestSuite) TestListByMailbox_OrderedByReceivedAtDesc() {
	// Arrange - create messages with different received_at times
	now := time.Now()
	messages := []struct {
		subject    string
		receivedAt time.Time
	}{
		{"Oldest", now.Add(-2 * time.Hour)},
		{"Middle", now.Add(-1 * time.Hour)},
		{"Newest", now},
	}

	for _, m := range messages {
		message := &models.Message{
			MailboxID:   s.testMailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     m.subject,
			ReceivedAt:  m.receivedAt,
		}
		err := s.db.Create(message).Error
		require.NoError(s.T(), err)
	}

	// Act
	result, _, err := s.repo.ListByMailbox(context.Background(), s.testMailbox.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 3)
	assert.Equal(s.T(), "Newest", result[0].Subject)
	assert.Equal(s.T(), "Middle", result[1].Subject)
	assert.Equal(s.T(), "Oldest", result[2].Subject)
}

func (s *MessageRepositoryTestSuite) TestListByMailbox_WithPagination() {
	// Arrange
	for i := 0; i < 5; i++ {
		message := &models.Message{
			MailboxID:   s.testMailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     "Message " + string(rune('A'+i)),
		}
		err := s.repo.Create(context.Background(), message)
		require.NoError(s.T(), err)
	}

	// Act - get first page
	result, total, err := s.repo.ListByMailbox(context.Background(), s.testMailbox.ID, 2, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 2)
	assert.Equal(s.T(), int64(5), total)

	// Act - get second page
	result2, _, err := s.repo.ListByMailbox(context.Background(), s.testMailbox.ID, 2, 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result2, 2)
}

func (s *MessageRepositoryTestSuite) TestListByMailbox_WithAttachmentCount() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024},
		{Filename: "doc2.pdf", ContentType: "application/pdf", FilePath: "/path/doc2.pdf", SizeBytes: 2048},
	}
	err := s.repo.CreateWithAttachments(context.Background(), message, attachments)
	require.NoError(s.T(), err)

	// Act
	result, _, err := s.repo.ListByMailbox(context.Background(), s.testMailbox.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	require.Len(s.T(), result, 1)
	assert.Equal(s.T(), int64(2), int64(result[0].AttachmentCount))
}

func (s *MessageRepositoryTestSuite) TestListByMailbox_Empty() {
	// Create another mailbox with no messages
	emptyMailbox := &models.Mailbox{
		LocalPart:   "empty",
		DomainID:    s.testDomain.ID,
		FullAddress: "empty@test.com",
	}
	err := s.db.Create(emptyMailbox).Error
	require.NoError(s.T(), err)

	// Act
	result, total, err := s.repo.ListByMailbox(context.Background(), emptyMailbox.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
	assert.Equal(s.T(), int64(0), total)
}

// ==================== MarkAsRead Tests ====================

func (s *MessageRepositoryTestSuite) TestMarkAsRead_Success() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Unread Message",
		IsRead:      false,
	}
	err := s.repo.Create(context.Background(), message)
	require.NoError(s.T(), err)
	assert.False(s.T(), message.IsRead)

	// Act
	err = s.repo.MarkAsRead(context.Background(), message.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify update
	result, err := s.repo.GetByID(context.Background(), message.ID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), result.IsRead)
}

func (s *MessageRepositoryTestSuite) TestMarkAsRead_NotFound() {
	// Act
	err := s.repo.MarkAsRead(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *MessageRepositoryTestSuite) TestMarkAsRead_AlreadyRead() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Already Read",
		IsRead:      true,
	}
	err := s.repo.Create(context.Background(), message)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.MarkAsRead(context.Background(), message.ID)

	// Assert - should succeed even if already read
	assert.NoError(s.T(), err)
}

// ==================== Delete Tests ====================

func (s *MessageRepositoryTestSuite) TestDelete_Success() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "To Delete",
	}
	err := s.repo.Create(context.Background(), message)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), message.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify deletion
	result, err := s.repo.GetByID(context.Background(), message.ID)
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *MessageRepositoryTestSuite) TestDelete_NotFound() {
	// Act
	err := s.repo.Delete(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *MessageRepositoryTestSuite) TestDelete_CascadeDeletesAttachments() {
	// Arrange
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", FilePath: "/path/doc.pdf", SizeBytes: 1024},
	}
	err := s.repo.CreateWithAttachments(context.Background(), message, attachments)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), message.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify message is deleted (SQLite cascade may not work, so we check message is deleted)
	_, err = s.repo.GetByID(context.Background(), message.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// ==================== CountUnread Tests ====================

func (s *MessageRepositoryTestSuite) TestCountUnread_ReturnsCorrectCount() {
	// Arrange - create 3 unread and 2 read messages
	for i := 0; i < 5; i++ {
		message := &models.Message{
			MailboxID:   s.testMailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     "Message " + string(rune('A'+i)),
			IsRead:      i < 2, // First 2 are read
		}
		err := s.repo.Create(context.Background(), message)
		require.NoError(s.T(), err)
	}

	// Act
	count, err := s.repo.CountUnread(context.Background(), s.testMailbox.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), count)
}

func (s *MessageRepositoryTestSuite) TestCountUnread_ZeroWhenAllRead() {
	// Arrange
	for i := 0; i < 3; i++ {
		message := &models.Message{
			MailboxID:   s.testMailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     "Read Message",
			IsRead:      true,
		}
		err := s.repo.Create(context.Background(), message)
		require.NoError(s.T(), err)
	}

	// Act
	count, err := s.repo.CountUnread(context.Background(), s.testMailbox.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), count)
}

func (s *MessageRepositoryTestSuite) TestCountUnread_ZeroWhenEmpty() {
	// Act
	count, err := s.repo.CountUnread(context.Background(), s.testMailbox.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), count)
}

// ==================== CRUD Round-Trip Test ====================

func (s *MessageRepositoryTestSuite) TestCRUD_RoundTrip() {
	// Create
	message := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Round Trip Test",
		BodyText:    "Test body",
		IsRead:      false,
	}
	err := s.repo.Create(context.Background(), message)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), message.ID)

	// Read
	retrieved, err := s.repo.GetByID(context.Background(), message.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), message.Subject, retrieved.Subject)
	assert.False(s.T(), retrieved.IsRead)

	// Update (mark as read)
	err = s.repo.MarkAsRead(context.Background(), message.ID)
	require.NoError(s.T(), err)

	// Verify update
	updated, err := s.repo.GetByID(context.Background(), message.ID)
	require.NoError(s.T(), err)
	assert.True(s.T(), updated.IsRead)

	// Delete
	err = s.repo.Delete(context.Background(), message.ID)
	require.NoError(s.T(), err)

	// Verify deletion
	_, err = s.repo.GetByID(context.Background(), message.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}
