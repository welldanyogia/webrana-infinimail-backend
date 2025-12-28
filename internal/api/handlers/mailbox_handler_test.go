package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/tests/mocks"
)

// MailboxHandlerTestSuite is the test suite for MailboxHandler
type MailboxHandlerTestSuite struct {
	suite.Suite
	echo            *echo.Echo
	handler         *MailboxHandler
	mockMailboxRepo *mocks.MockMailboxRepository
	mockDomainRepo  *mocks.MockDomainRepository
}

// SetupTest runs before each test
func (s *MailboxHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.mockMailboxRepo = new(mocks.MockMailboxRepository)
	s.mockDomainRepo = new(mocks.MockDomainRepository)
	s.handler = NewMailboxHandler(s.mockMailboxRepo, s.mockDomainRepo)
}

// TearDownTest runs after each test
func (s *MailboxHandlerTestSuite) TearDownTest() {
	s.mockMailboxRepo.AssertExpectations(s.T())
	s.mockDomainRepo.AssertExpectations(s.T())
}

// TestMailboxHandlerTestSuite runs the test suite
func TestMailboxHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MailboxHandlerTestSuite))
}

// Helper function to create a test context
func (s *MailboxHandlerTestSuite) createContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	return c, rec
}

// Helper function to create a test domain
func (s *MailboxHandlerTestSuite) createTestDomain(id uint, name string, active bool) *models.Domain {
	now := time.Now()
	return &models.Domain{
		ID:        id,
		Name:      name,
		IsActive:  active,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Helper function to create a test mailbox
func (s *MailboxHandlerTestSuite) createTestMailbox(id uint, localPart string, domainID uint, fullAddress string) *models.Mailbox {
	now := time.Now()
	return &models.Mailbox{
		ID:          id,
		LocalPart:   localPart,
		DomainID:    domainID,
		FullAddress: fullAddress,
		CreatedAt:   now,
	}
}

// ==================== Create Tests ====================

// TestCreate_ValidInput tests creating a mailbox with valid input
func (s *MailboxHandlerTestSuite) TestCreate_ValidInput() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"local_part": "user", "domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockMailboxRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Mailbox")).
		Run(func(args mock.Arguments) {
			mailbox := args.Get(1).(*models.Mailbox)
			mailbox.ID = 1
		}).
		Return(nil)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestCreate_InvalidDomainID tests creating a mailbox with non-existent domain
func (s *MailboxHandlerTestSuite) TestCreate_InvalidDomainID() {
	// Arrange
	body := `{"local_part": "user", "domain_id": 999}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestCreate_DuplicateAddress tests creating a mailbox with duplicate address
func (s *MailboxHandlerTestSuite) TestCreate_DuplicateAddress() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"local_part": "user", "domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockMailboxRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Mailbox")).
		Return(repository.ErrDuplicateEntry)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusConflict, rec.Code)
}

// TestCreate_EmptyLocalPart tests creating a mailbox with empty local_part
func (s *MailboxHandlerTestSuite) TestCreate_EmptyLocalPart() {
	// Arrange
	body := `{"local_part": "", "domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestCreate_MissingDomainID tests creating a mailbox without domain_id
func (s *MailboxHandlerTestSuite) TestCreate_MissingDomainID() {
	// Arrange
	body := `{"local_part": "user"}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestCreate_InactiveDomain tests creating a mailbox with inactive domain
func (s *MailboxHandlerTestSuite) TestCreate_InactiveDomain() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", false)
	body := `{"local_part": "user", "domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}


// ==================== CreateRandom Tests ====================

// TestCreateRandom_ValidInput tests creating a random mailbox with valid input
func (s *MailboxHandlerTestSuite) TestCreateRandom_ValidInput() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes/random", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockMailboxRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *models.Mailbox) bool {
		// Verify local_part is 8 characters alphanumeric
		return len(m.LocalPart) == 8 && m.DomainID == 1
	})).
		Run(func(args mock.Arguments) {
			mailbox := args.Get(1).(*models.Mailbox)
			mailbox.ID = 1
		}).
		Return(nil)

	// Act
	err := s.handler.CreateRandom(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestCreateRandom_InvalidDomainID tests creating a random mailbox with non-existent domain
func (s *MailboxHandlerTestSuite) TestCreateRandom_InvalidDomainID() {
	// Arrange
	body := `{"domain_id": 999}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes/random", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.CreateRandom(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestCreateRandom_MissingDomainID tests creating a random mailbox without domain_id
func (s *MailboxHandlerTestSuite) TestCreateRandom_MissingDomainID() {
	// Arrange
	body := `{}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes/random", body)

	// Act
	err := s.handler.CreateRandom(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestCreateRandom_InactiveDomain tests creating a random mailbox with inactive domain
func (s *MailboxHandlerTestSuite) TestCreateRandom_InactiveDomain() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", false)
	body := `{"domain_id": 1}`
	c, rec := s.createContext(http.MethodPost, "/api/mailboxes/random", body)

	s.mockDomainRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)

	// Act
	err := s.handler.CreateRandom(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// ==================== Get Tests ====================

// TestGet_ValidID tests getting a mailbox with valid ID
func (s *MailboxHandlerTestSuite) TestGet_ValidID() {
	// Arrange
	mailbox := s.createTestMailbox(1, "user", 1, "user@example.com")
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(1)).Return(mailbox, nil)
	s.mockMailboxRepo.On("UpdateLastAccessed", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestGet_NonExistentID tests getting a mailbox with non-existent ID
func (s *MailboxHandlerTestSuite) TestGet_NonExistentID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestGet_InvalidID tests getting a mailbox with invalid ID format
func (s *MailboxHandlerTestSuite) TestGet_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestGet_UpdatesLastAccessed tests that Get updates last_accessed_at
func (s *MailboxHandlerTestSuite) TestGet_UpdatesLastAccessed() {
	// Arrange
	mailbox := s.createTestMailbox(1, "user", 1, "user@example.com")
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(1)).Return(mailbox, nil)
	s.mockMailboxRepo.On("UpdateLastAccessed", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.mockMailboxRepo.AssertCalled(s.T(), "UpdateLastAccessed", mock.Anything, uint(1))
}

// ==================== List Tests ====================

// TestList_WithDomainID tests listing mailboxes with domain_id filter
func (s *MailboxHandlerTestSuite) TestList_WithDomainID() {
	// Arrange
	mailboxes := []models.MailboxWithUnreadCount{
		{
			Mailbox:     *s.createTestMailbox(1, "user1", 1, "user1@example.com"),
			UnreadCount: 5,
		},
		{
			Mailbox:     *s.createTestMailbox(2, "user2", 1, "user2@example.com"),
			UnreadCount: 0,
		},
	}
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes?domain_id=1", "")

	s.mockMailboxRepo.On("ListByDomain", mock.Anything, uint(1), 20, 0).Return(mailboxes, int64(2), nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.PaginatedResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
	s.Equal(int64(2), resp.Meta.Total)
}

// TestList_WithPagination tests listing mailboxes with pagination
func (s *MailboxHandlerTestSuite) TestList_WithPagination() {
	// Arrange
	mailboxes := []models.MailboxWithUnreadCount{
		{
			Mailbox:     *s.createTestMailbox(3, "user3", 1, "user3@example.com"),
			UnreadCount: 0,
		},
	}
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes?domain_id=1&limit=10&offset=20", "")

	s.mockMailboxRepo.On("ListByDomain", mock.Anything, uint(1), 10, 20).Return(mailboxes, int64(25), nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.PaginatedResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
	s.Equal(10, resp.Meta.Limit)
	s.Equal(20, resp.Meta.Offset)
}

// TestList_MissingDomainID tests listing mailboxes without domain_id
func (s *MailboxHandlerTestSuite) TestList_MissingDomainID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes", "")

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestList_InvalidDomainID tests listing mailboxes with invalid domain_id
func (s *MailboxHandlerTestSuite) TestList_InvalidDomainID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes?domain_id=invalid", "")

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestList_InternalError tests listing mailboxes when repository returns error
func (s *MailboxHandlerTestSuite) TestList_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes?domain_id=1", "")

	s.mockMailboxRepo.On("ListByDomain", mock.Anything, uint(1), 20, 0).Return(nil, int64(0), errors.New("database error"))

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== Delete Tests ====================

// TestDelete_ValidID tests deleting a mailbox with valid ID
func (s *MailboxHandlerTestSuite) TestDelete_ValidID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/mailboxes/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNoContent, rec.Code)
}

// TestDelete_NonExistentID tests deleting a non-existent mailbox
func (s *MailboxHandlerTestSuite) TestDelete_NonExistentID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/mailboxes/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockMailboxRepo.On("Delete", mock.Anything, uint(999)).Return(repository.ErrNotFound)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestDelete_InvalidID tests deleting a mailbox with invalid ID format
func (s *MailboxHandlerTestSuite) TestDelete_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/mailboxes/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestDelete_InternalError tests deleting a mailbox when repository returns error
func (s *MailboxHandlerTestSuite) TestDelete_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/mailboxes/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("Delete", mock.Anything, uint(1)).Return(errors.New("database error"))

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== Helper Function Tests ====================

// TestGenerateRandomString tests that generateRandomString produces correct length
func TestGenerateRandomString(t *testing.T) {
	// Test multiple times to ensure consistency
	for i := 0; i < 10; i++ {
		result := generateRandomString(8)
		if len(result) != 8 {
			t.Errorf("Expected length 8, got %d", len(result))
		}
		// Verify all characters are alphanumeric
		for _, c := range result {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				t.Errorf("Invalid character in random string: %c", c)
			}
		}
	}
}
