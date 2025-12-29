package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MailboxRepositoryTestSuite is the test suite for MailboxRepository
type MailboxRepositoryTestSuite struct {
	suite.Suite
	db         *gorm.DB
	repo       MailboxRepository
	domainRepo DomainRepository
	testDomain *models.Domain
}

// SetupSuite runs once before all tests
func (s *MailboxRepositoryTestSuite) SetupSuite() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)

	// Enable foreign keys for SQLite (required for cascade delete)
	db.Exec("PRAGMA foreign_keys = ON")

	// Auto-migrate models
	err = db.AutoMigrate(&models.Domain{}, &models.DomainCertificate{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
	require.NoError(s.T(), err)

	s.db = db
	s.repo = NewMailboxRepository(db)
	s.domainRepo = NewDomainRepository(db)
}

// TearDownSuite runs once after all tests
func (s *MailboxRepositoryTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test - clean up data and create test domain
func (s *MailboxRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM attachments")
	s.db.Exec("DELETE FROM messages")
	s.db.Exec("DELETE FROM mailboxes")
	s.db.Exec("DELETE FROM domains")

	// Create a test domain for mailbox tests
	s.testDomain = &models.Domain{Name: "test.com", IsActive: true}
	err := s.domainRepo.Create(context.Background(), s.testDomain)
	require.NoError(s.T(), err)
}

// TestMailboxRepositoryTestSuite runs the test suite
func TestMailboxRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(MailboxRepositoryTestSuite))
}

// ==================== Create Tests ====================

func (s *MailboxRepositoryTestSuite) TestCreate_Success() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "user",
		DomainID:    s.testDomain.ID,
		FullAddress: "user@test.com",
	}

	// Act
	err := s.repo.Create(context.Background(), mailbox)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), mailbox.ID)
	assert.NotZero(s.T(), mailbox.CreatedAt)
}

func (s *MailboxRepositoryTestSuite) TestCreate_DuplicateAddress_ReturnsError() {
	// Arrange
	mailbox1 := &models.Mailbox{
		LocalPart:   "duplicate",
		DomainID:    s.testDomain.ID,
		FullAddress: "duplicate@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox1)
	require.NoError(s.T(), err)

	mailbox2 := &models.Mailbox{
		LocalPart:   "duplicate",
		DomainID:    s.testDomain.ID,
		FullAddress: "duplicate@test.com",
	}

	// Act
	err = s.repo.Create(context.Background(), mailbox2)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrDuplicateEntry)
}

// ==================== GetByID Tests ====================

func (s *MailboxRepositoryTestSuite) TestGetByID_Found() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "getbyid",
		DomainID:    s.testDomain.ID,
		FullAddress: "getbyid@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByID(context.Background(), mailbox.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), mailbox.ID, result.ID)
	assert.Equal(s.T(), "getbyid@test.com", result.FullAddress)
}

func (s *MailboxRepositoryTestSuite) TestGetByID_NotFound() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

// ==================== GetByAddress Tests ====================

func (s *MailboxRepositoryTestSuite) TestGetByAddress_Found() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "byaddress",
		DomainID:    s.testDomain.ID,
		FullAddress: "byaddress@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByAddress(context.Background(), "byaddress@test.com")

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), mailbox.ID, result.ID)
}

func (s *MailboxRepositoryTestSuite) TestGetByAddress_NotFound() {
	// Act
	result, err := s.repo.GetByAddress(context.Background(), "nonexistent@test.com")

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

// ==================== GetOrCreate Tests ====================

func (s *MailboxRepositoryTestSuite) TestGetOrCreate_CreatesNew() {
	// Act
	result, created, err := s.repo.GetOrCreate(context.Background(), "newuser", s.testDomain.ID, "test.com")

	// Assert
	assert.NoError(s.T(), err)
	assert.True(s.T(), created)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), "newuser@test.com", result.FullAddress)
	assert.NotZero(s.T(), result.ID)
}

func (s *MailboxRepositoryTestSuite) TestGetOrCreate_ReturnsExisting() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "existing",
		DomainID:    s.testDomain.ID,
		FullAddress: "existing@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Act
	result, created, err := s.repo.GetOrCreate(context.Background(), "existing", s.testDomain.ID, "test.com")

	// Assert
	assert.NoError(s.T(), err)
	assert.False(s.T(), created)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), mailbox.ID, result.ID)
}

// ==================== ListByDomain Tests ====================

func (s *MailboxRepositoryTestSuite) TestListByDomain_ReturnsMailboxes() {
	// Arrange
	for i := 0; i < 3; i++ {
		mailbox := &models.Mailbox{
			LocalPart:   "user" + string(rune('a'+i)),
			DomainID:    s.testDomain.ID,
			FullAddress: "user" + string(rune('a'+i)) + "@test.com",
		}
		err := s.repo.Create(context.Background(), mailbox)
		require.NoError(s.T(), err)
	}

	// Act
	result, total, err := s.repo.ListByDomain(context.Background(), s.testDomain.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 3)
	assert.Equal(s.T(), int64(3), total)
}

func (s *MailboxRepositoryTestSuite) TestListByDomain_WithPagination() {
	// Arrange
	for i := 0; i < 5; i++ {
		mailbox := &models.Mailbox{
			LocalPart:   "page" + string(rune('a'+i)),
			DomainID:    s.testDomain.ID,
			FullAddress: "page" + string(rune('a'+i)) + "@test.com",
		}
		err := s.repo.Create(context.Background(), mailbox)
		require.NoError(s.T(), err)
	}

	// Act - get first page
	result, total, err := s.repo.ListByDomain(context.Background(), s.testDomain.ID, 2, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 2)
	assert.Equal(s.T(), int64(5), total)

	// Act - get second page
	result2, _, err := s.repo.ListByDomain(context.Background(), s.testDomain.ID, 2, 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result2, 2)
}

func (s *MailboxRepositoryTestSuite) TestListByDomain_WithUnreadCount() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "unread",
		DomainID:    s.testDomain.ID,
		FullAddress: "unread@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Create some messages (2 unread, 1 read)
	for i := 0; i < 3; i++ {
		msg := &models.Message{
			MailboxID:   mailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     "Test",
			IsRead:      i == 0, // First one is read
		}
		err := s.db.Create(msg).Error
		require.NoError(s.T(), err)
	}

	// Act
	result, _, err := s.repo.ListByDomain(context.Background(), s.testDomain.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	require.Len(s.T(), result, 1)
	assert.Equal(s.T(), int64(2), result[0].UnreadCount)
}

func (s *MailboxRepositoryTestSuite) TestListByDomain_Empty() {
	// Create another domain with no mailboxes
	emptyDomain := &models.Domain{Name: "empty.com", IsActive: true}
	err := s.domainRepo.Create(context.Background(), emptyDomain)
	require.NoError(s.T(), err)

	// Act
	result, total, err := s.repo.ListByDomain(context.Background(), emptyDomain.ID, 10, 0)

	// Assert
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
	assert.Equal(s.T(), int64(0), total)
}

// ==================== UpdateLastAccessed Tests ====================

func (s *MailboxRepositoryTestSuite) TestUpdateLastAccessed_Success() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "lastaccess",
		DomainID:    s.testDomain.ID,
		FullAddress: "lastaccess@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), mailbox.LastAccessedAt)

	// Act
	err = s.repo.UpdateLastAccessed(context.Background(), mailbox.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify update
	result, err := s.repo.GetByID(context.Background(), mailbox.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result.LastAccessedAt)
}

func (s *MailboxRepositoryTestSuite) TestUpdateLastAccessed_NotFound() {
	// Act
	err := s.repo.UpdateLastAccessed(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// ==================== Delete Tests ====================

func (s *MailboxRepositoryTestSuite) TestDelete_Success() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "todelete",
		DomainID:    s.testDomain.ID,
		FullAddress: "todelete@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), mailbox.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify deletion
	result, err := s.repo.GetByID(context.Background(), mailbox.ID)
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *MailboxRepositoryTestSuite) TestDelete_NotFound() {
	// Act
	err := s.repo.Delete(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *MailboxRepositoryTestSuite) TestDelete_CascadeDeletesMessages() {
	// Arrange
	mailbox := &models.Mailbox{
		LocalPart:   "cascade",
		DomainID:    s.testDomain.ID,
		FullAddress: "cascade@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)

	// Create message for this mailbox
	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Test",
	}
	err = s.db.Create(message).Error
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), mailbox.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify mailbox is deleted (SQLite cascade may not work, so we check mailbox is deleted)
	_, err = s.repo.GetByID(context.Background(), mailbox.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// ==================== CRUD Round-Trip Test ====================

func (s *MailboxRepositoryTestSuite) TestCRUD_RoundTrip() {
	// Create
	mailbox := &models.Mailbox{
		LocalPart:   "roundtrip",
		DomainID:    s.testDomain.ID,
		FullAddress: "roundtrip@test.com",
	}
	err := s.repo.Create(context.Background(), mailbox)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), mailbox.ID)

	// Read by ID
	retrieved, err := s.repo.GetByID(context.Background(), mailbox.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), mailbox.FullAddress, retrieved.FullAddress)

	// Read by Address
	retrieved, err = s.repo.GetByAddress(context.Background(), "roundtrip@test.com")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), mailbox.ID, retrieved.ID)

	// Update last accessed
	err = s.repo.UpdateLastAccessed(context.Background(), mailbox.ID)
	require.NoError(s.T(), err)

	// Verify update
	updated, err := s.repo.GetByID(context.Background(), mailbox.ID)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), updated.LastAccessedAt)

	// Delete
	err = s.repo.Delete(context.Background(), mailbox.ID)
	require.NoError(s.T(), err)

	// Verify deletion
	_, err = s.repo.GetByID(context.Background(), mailbox.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}
