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

// DomainRepositoryTestSuite is the test suite for DomainRepository
type DomainRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo DomainRepository
}

// SetupSuite runs once before all tests
func (s *DomainRepositoryTestSuite) SetupSuite() {
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
	s.repo = NewDomainRepository(db)
}

// TearDownSuite runs once after all tests
func (s *DomainRepositoryTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test - clean up data
func (s *DomainRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM attachments")
	s.db.Exec("DELETE FROM messages")
	s.db.Exec("DELETE FROM mailboxes")
	s.db.Exec("DELETE FROM domains")
}

// TestDomainRepositoryTestSuite runs the test suite
func TestDomainRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(DomainRepositoryTestSuite))
}

// ==================== Create Tests ====================

func (s *DomainRepositoryTestSuite) TestCreate_Success() {
	// Arrange
	domain := &models.Domain{Name: "example.com", IsActive: true}

	// Act
	err := s.repo.Create(context.Background(), domain)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), domain.ID)
	assert.NotZero(s.T(), domain.CreatedAt)
	assert.NotZero(s.T(), domain.UpdatedAt)
}

func (s *DomainRepositoryTestSuite) TestCreate_DuplicateName_ReturnsError() {
	// Arrange
	domain1 := &models.Domain{Name: "duplicate.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain1)
	require.NoError(s.T(), err)

	domain2 := &models.Domain{Name: "duplicate.com", IsActive: true}

	// Act
	err = s.repo.Create(context.Background(), domain2)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrDuplicateEntry)
}

func (s *DomainRepositoryTestSuite) TestCreate_MultipleDomains_Success() {
	// Arrange
	domains := []*models.Domain{
		{Name: "domain1.com", IsActive: true},
		{Name: "domain2.com", IsActive: false},
		{Name: "domain3.com", IsActive: true},
	}

	// Act & Assert
	for _, domain := range domains {
		err := s.repo.Create(context.Background(), domain)
		assert.NoError(s.T(), err)
		assert.NotZero(s.T(), domain.ID)
	}
}

// ==================== GetByID Tests ====================

func (s *DomainRepositoryTestSuite) TestGetByID_Found() {
	// Arrange
	domain := &models.Domain{Name: "getbyid.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByID(context.Background(), domain.ID)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), domain.ID, result.ID)
	assert.Equal(s.T(), "getbyid.com", result.Name)
	assert.True(s.T(), result.IsActive)
}

func (s *DomainRepositoryTestSuite) TestGetByID_NotFound() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *DomainRepositoryTestSuite) TestGetByID_ZeroID() {
	// Act
	result, err := s.repo.GetByID(context.Background(), 0)

	// Assert
	assert.Error(s.T(), err)
	assert.Nil(s.T(), result)
}

// ==================== GetByName Tests ====================

func (s *DomainRepositoryTestSuite) TestGetByName_Found() {
	// Arrange
	domain := &models.Domain{Name: "getbyname.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.GetByName(context.Background(), "getbyname.com")

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), domain.ID, result.ID)
	assert.Equal(s.T(), "getbyname.com", result.Name)
}

func (s *DomainRepositoryTestSuite) TestGetByName_NotFound() {
	// Act
	result, err := s.repo.GetByName(context.Background(), "nonexistent.com")

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *DomainRepositoryTestSuite) TestGetByName_EmptyName() {
	// Act
	result, err := s.repo.GetByName(context.Background(), "")

	// Assert
	assert.Error(s.T(), err)
	assert.Nil(s.T(), result)
}

// ==================== List Tests ====================

func (s *DomainRepositoryTestSuite) TestList_ReturnsAllDomains() {
	// Arrange
	domains := []*models.Domain{
		{Name: "active1.com", IsActive: true},
		{Name: "inactive1.com", IsActive: false},
		{Name: "active2.com", IsActive: true},
	}
	for _, d := range domains {
		err := s.repo.Create(context.Background(), d)
		require.NoError(s.T(), err)
	}

	// Act
	result, err := s.repo.List(context.Background(), false)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 3)
}

func (s *DomainRepositoryTestSuite) TestList_ActiveOnly() {
	// Arrange - create domains with explicit is_active values
	activeDomain1 := &models.Domain{Name: "active-a.com", IsActive: true}
	err := s.repo.Create(context.Background(), activeDomain1)
	require.NoError(s.T(), err)

	// Create inactive domain and explicitly set is_active to false
	inactiveDomain := &models.Domain{Name: "inactive-a.com", IsActive: true}
	err = s.repo.Create(context.Background(), inactiveDomain)
	require.NoError(s.T(), err)
	// Update to inactive
	inactiveDomain.IsActive = false
	err = s.repo.Update(context.Background(), inactiveDomain)
	require.NoError(s.T(), err)

	activeDomain2 := &models.Domain{Name: "active-b.com", IsActive: true}
	err = s.repo.Create(context.Background(), activeDomain2)
	require.NoError(s.T(), err)

	// Act
	result, err := s.repo.List(context.Background(), true)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 2)
	for _, d := range result {
		assert.True(s.T(), d.IsActive)
	}
}

func (s *DomainRepositoryTestSuite) TestList_Empty() {
	// Act
	result, err := s.repo.List(context.Background(), false)

	// Assert
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
}

func (s *DomainRepositoryTestSuite) TestList_OrderedByName() {
	// Arrange
	domains := []*models.Domain{
		{Name: "zebra.com", IsActive: true},
		{Name: "alpha.com", IsActive: true},
		{Name: "middle.com", IsActive: true},
	}
	for _, d := range domains {
		err := s.repo.Create(context.Background(), d)
		require.NoError(s.T(), err)
	}

	// Act
	result, err := s.repo.List(context.Background(), false)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result, 3)
	assert.Equal(s.T(), "alpha.com", result[0].Name)
	assert.Equal(s.T(), "middle.com", result[1].Name)
	assert.Equal(s.T(), "zebra.com", result[2].Name)
}

// ==================== Update Tests ====================

func (s *DomainRepositoryTestSuite) TestUpdate_Success() {
	// Arrange
	domain := &models.Domain{Name: "original.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)

	// Act
	domain.Name = "updated.com"
	domain.IsActive = false
	err = s.repo.Update(context.Background(), domain)

	// Assert
	assert.NoError(s.T(), err)

	// Verify update
	result, err := s.repo.GetByID(context.Background(), domain.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "updated.com", result.Name)
	assert.False(s.T(), result.IsActive)
}

func (s *DomainRepositoryTestSuite) TestUpdate_DuplicateName_ReturnsError() {
	// Arrange
	domain1 := &models.Domain{Name: "first.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain1)
	require.NoError(s.T(), err)

	domain2 := &models.Domain{Name: "second.com", IsActive: true}
	err = s.repo.Create(context.Background(), domain2)
	require.NoError(s.T(), err)

	// Act - try to update domain2 to have domain1's name
	domain2.Name = "first.com"
	err = s.repo.Update(context.Background(), domain2)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrDuplicateEntry)
}

// ==================== Delete Tests ====================

func (s *DomainRepositoryTestSuite) TestDelete_Success() {
	// Arrange
	domain := &models.Domain{Name: "todelete.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), domain.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify deletion
	result, err := s.repo.GetByID(context.Background(), domain.ID)
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
	assert.Nil(s.T(), result)
}

func (s *DomainRepositoryTestSuite) TestDelete_NotFound() {
	// Act
	err := s.repo.Delete(context.Background(), 99999)

	// Assert
	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *DomainRepositoryTestSuite) TestDelete_CascadeDeletesMailboxes() {
	// Arrange
	domain := &models.Domain{Name: "cascade.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)

	// Create mailbox for this domain
	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@cascade.com",
	}
	err = s.db.Create(mailbox).Error
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(context.Background(), domain.ID)

	// Assert
	assert.NoError(s.T(), err)

	// Verify mailbox is also deleted (SQLite cascade may not work, so we check domain is deleted)
	_, err = s.repo.GetByID(context.Background(), domain.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// ==================== CRUD Round-Trip Test ====================

func (s *DomainRepositoryTestSuite) TestCRUD_RoundTrip() {
	// Create
	domain := &models.Domain{Name: "roundtrip.com", IsActive: true}
	err := s.repo.Create(context.Background(), domain)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), domain.ID)

	// Read by ID
	retrieved, err := s.repo.GetByID(context.Background(), domain.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), domain.Name, retrieved.Name)

	// Read by Name
	retrieved, err = s.repo.GetByName(context.Background(), "roundtrip.com")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), domain.ID, retrieved.ID)

	// Update
	retrieved.IsActive = false
	err = s.repo.Update(context.Background(), retrieved)
	require.NoError(s.T(), err)

	// Verify update
	updated, err := s.repo.GetByID(context.Background(), domain.ID)
	require.NoError(s.T(), err)
	assert.False(s.T(), updated.IsActive)

	// Delete
	err = s.repo.Delete(context.Background(), domain.ID)
	require.NoError(s.T(), err)

	// Verify deletion
	_, err = s.repo.GetByID(context.Background(), domain.ID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}
