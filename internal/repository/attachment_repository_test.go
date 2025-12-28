package repository

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MockFileStorageForRepo is a simple mock for file storage in repository tests
type MockFileStorageForRepo struct {
	DeletedPaths []string
	DeleteError  error
}

func (m *MockFileStorageForRepo) Save(filename string, content io.Reader) (string, error) {
	return "/mock/path/" + filename, nil
}

func (m *MockFileStorageForRepo) Get(filepath string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("mock content"))), nil
}

func (m *MockFileStorageForRepo) Delete(filepath string) error {
	m.DeletedPaths = append(m.DeletedPaths, filepath)
	return m.DeleteError
}

// Ensure MockFileStorageForRepo implements storage.FileStorage
var _ storage.FileStorage = (*MockFileStorageForRepo)(nil)

// AttachmentRepositoryTestSuite is the test suite for AttachmentRepository
type AttachmentRepositoryTestSuite struct {
	suite.Suite
	db          *gorm.DB
	repo        AttachmentRepository
	mockStorage *MockFileStorageForRepo
	testDomain  *models.Domain
	testMailbox *models.Mailbox
	testMessage *models.Message
}

// SetupSuite runs once before all tests
func (s *AttachmentRepositoryTestSuite) SetupSuite() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.Domain{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
	require.NoError(s.T(), err)

	s.db = db
	s.mockStorage = &MockFileStorageForRepo{}
	s.repo = NewAttachmentRepository(db, s.mockStorage)
}

// TearDownSuite runs once after all tests
func (s *AttachmentRepositoryTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test - clean up data and create test fixtures
func (s *AttachmentRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM attachments")
	s.db.Exec("DELETE FROM messages")
	s.db.Exec("DELETE FROM mailboxes")
	s.db.Exec("DELETE FROM domains")

	// Reset mock storage
	s.mockStorage.DeletedPaths = nil
	s.mockStorage.DeleteError = nil

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

	// Create test message
	s.testMessage = &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Test Message",
	}
	err = s.db.Create(s.testMessage).Error
	require.NoError(s.T(), err)
}

// TestAttachmentRepositoryTestSuite runs the test suite
func TestAttachmentRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(AttachmentRepositoryTestSuite))
}

// ==================== Create Tests ====================

func (s *AttachmentRepositoryTestSuite) TestCreate_Success() {
	// Arrange
	attachment := &models.Attachment{
		MessageID:   s.testMessage.ID,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		FilePath:    "/attachments/document.pdf",
		SizeBytes:   1024,
	}

	// Act
	err := s.repo.Create(context.Background(), attachment)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), attachment.ID)
}

func (s *AttachmentRepositoryTestSuite) TestCreate_MultipleAttachments() {
	// Arrange
	attachments := []*models.Attachment{
		{MessageID: s.testMessage.ID, Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024},
		{MessageID: s.testMessage.ID, Filename: "image.png", ContentType: "image/png", FilePath: "/path/image.png", SizeBytes: 2048},
		{MessageID: s.testMessage.ID, Filename: "data.csv", ContentType: "text/csv", FilePath: "/path/data.csv", SizeBytes: 512},
	}

	// Act & Assert
	for _, att := range attachments {
		err := s.repo.Create(context.Background(), att)
		assert.NoError(s.T(), err)
		assert.NotZero(s.T(), att.ID)
	}
}

// ==================== GetByID Tests ====================

func (s *AttachmentRepositoryTestSuite) TestGetByID_Found() {
	// Arrange
	attachment := &models.Attachment{
		MessageID:   s.testMessage.ID,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		FilePath:    "/attachments/document.pdf",
		SizeBytes:   1024,
	}
	err := s.repo.Create(context.Background(), attachment)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByID(context.Background(), attachment.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), attachment.ID, result.ID)
	assert.Equal(s.T(), "document.pdf", result.Filename)
	assert.Equal(s.T(), "application/pdf", result.ContentType)
	assert.Equal(s.T(), int64(1024), result.SizeBytes)
}

func (s *AttachmentRepositoryTestSuite) TestGetByID_NotFound() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *AttachmentRepositoryTestSuite) TestGetByID_ZeroID() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 0)

	// Assert
	assert.Error(s.T(), err)
	assert.Nil(s.T(), result)
}

// ==================== ListByMessage Tests ====================

func (s *AttachmentRepositoryTestSuite) TestListByMessage_ReturnsAttachments() {
	// Arrange
	attachments := []*models.Attachment{
		{MessageID: s.testMessage.ID, Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024},
		{MessageID: s.testMessage.ID, Filename: "doc2.pdf", ContentType: "application/pdf", FilePath: "/path/doc2.pdf", SizeBytes: 2048},
	}
	for _, att := range attachments {
		err := s.repo.Create(context.Background(), att)
		require.NoError(s.T(), err)
	}

	// Act
	result, err := s.repo.ListByMessage(context.Background(), s.testMessage.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 2)
}

func (s *AttachmentRepositoryTestSuite) TestListByMessage_Empty() {
	// Create another message with no attachments
	emptyMessage := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "No Attachments",
	}
	err := s.db.Create(emptyMessage).Error
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.ListByMessage(context.Background(), emptyMessage.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
}

func (s *AttachmentRepositoryTestSuite) TestListByMessage_OnlyReturnsForSpecificMessage() {
	// Arrange - create attachments for test message
	att1 := &models.Attachment{MessageID: s.testMessage.ID, Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024}
	err := s.repo.Create(context.Background(), att1)
	require.NoError(s.T(), err)

	// Create another message with its own attachment
	otherMessage := &models.Message{
		MailboxID:   s.testMailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Other Message",
	}
	err = s.db.Create(otherMessage).Error
	require.NoError(s.T(), err)

	att2 := &models.Attachment{MessageID: otherMessage.ID, Filename: "other.pdf", ContentType: "application/pdf", FilePath: "/path/other.pdf", SizeBytes: 512}
	err = s.repo.Create(context.Background(), att2)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.ListByMessage(context.Background(), s.testMessage.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 1)
	assert.Equal(s.T(), "doc1.pdf", result[0].Filename)
}

// ==================== Delete Tests ====================

func (s *AttachmentRepositoryTestSuite) TestDelete_Success() {
	// Arrange
	attachment := &models.Attachment{
		MessageID:   s.testMessage.ID,
		Filename:    "todelete.pdf",
		ContentType: "application/pdf",
		FilePath:    "/attachments/todelete.pdf",
		SizeBytes:   1024,
	}
	err := s.repo.Create(context.Background(), attachment)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), attachment.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify deletion from database
	result, err := s.repo.GetByID(context.Background(), attachment.ID)
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)

	// Verify file storage delete was called
	assert.Contains(s.T(), s.mockStorage.DeletedPaths, "/attachments/todelete.pdf")
}

func (s *AttachmentRepositoryTestSuite) TestDelete_NotFound() {
	// Act
	err := s.repo.Delete(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *AttachmentRepositoryTestSuite) TestDelete_EmptyFilePath() {
	// Arrange - attachment with no file path
	attachment := &models.Attachment{
		MessageID:   s.testMessage.ID,
		Filename:    "nopath.pdf",
		ContentType: "application/pdf",
		FilePath:    "",
		SizeBytes:   0,
	}
	err := s.repo.Create(context.Background(), attachment)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), attachment.ID)

	// Assert - should succeed without calling file storage
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), s.mockStorage.DeletedPaths)
}

// ==================== CRUD Round-Trip Test ====================

func (s *AttachmentRepositoryTestSuite) TestCRUD_RoundTrip() {
	// Create
	attachment := &models.Attachment{
		MessageID:   s.testMessage.ID,
		Filename:    "roundtrip.pdf",
		ContentType: "application/pdf",
		FilePath:    "/attachments/roundtrip.pdf",
		SizeBytes:   2048,
	}
	err := s.repo.Create(context.Background(), attachment)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), attachment.ID)

	// Read by ID
	retrieved, err := s.repo.GetByID(context.Background(), attachment.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), attachment.Filename, retrieved.Filename)
	assert.Equal(s.T(), attachment.ContentType, retrieved.ContentType)
	assert.Equal(s.T(), attachment.SizeBytes, retrieved.SizeBytes)

	// List by Message
	list, err := s.repo.ListByMessage(context.Background(), s.testMessage.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), list, 1)
	assert.Equal(s.T(), attachment.ID, list[0].ID)

	// Delete
	err = s.repo.Delete(context.Background(), attachment.ID)
	require.NoError(s.T(), err)

	// Verify deletion
	_, err = s.repo.GetByID(context.Background(), attachment.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// ==================== Content Type Tests ====================

func (s *AttachmentRepositoryTestSuite) TestCreate_VariousContentTypes() {
	// Arrange
	contentTypes := []struct {
		filename    string
		contentType string
	}{
		{"document.pdf", "application/pdf"},
		{"image.png", "image/png"},
		{"image.jpg", "image/jpeg"},
		{"data.json", "application/json"},
		{"text.txt", "text/plain"},
		{"archive.zip", "application/zip"},
	}

	// Act & Assert
	for _, ct := range contentTypes {
		attachment := &models.Attachment{
			MessageID:   s.testMessage.ID,
			Filename:    ct.filename,
			ContentType: ct.contentType,
			FilePath:    "/path/" + ct.filename,
			SizeBytes:   1024,
		}
		err := s.repo.Create(context.Background(), attachment)
		assert.NoError(s.T(), err)

		// Verify retrieval
		retrieved, err := s.repo.GetByID(context.Background(), attachment.ID)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), ct.contentType, retrieved.ContentType)
	}
}

// ==================== Size Tests ====================

func (s *AttachmentRepositoryTestSuite) TestCreate_VariousSizes() {
	// Arrange
	sizes := []int64{0, 1, 1024, 1024 * 1024, 10 * 1024 * 1024}

	// Act & Assert
	for i, size := range sizes {
		attachment := &models.Attachment{
			MessageID:   s.testMessage.ID,
			Filename:    "file" + string(rune('a'+i)) + ".bin",
			ContentType: "application/octet-stream",
			FilePath:    "/path/file" + string(rune('a'+i)) + ".bin",
			SizeBytes:   size,
		}
		err := s.repo.Create(context.Background(), attachment)
		assert.NoError(s.T(), err)

		// Verify retrieval
		retrieved, err := s.repo.GetByID(context.Background(), attachment.ID)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), size, retrieved.SizeBytes)
	}
}
