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

// DomainHandlerTestSuite is the test suite for DomainHandler
type DomainHandlerTestSuite struct {
	suite.Suite
	echo     *echo.Echo
	handler  *DomainHandler
	mockRepo *mocks.MockDomainRepository
}

// SetupTest runs before each test
func (s *DomainHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.mockRepo = new(mocks.MockDomainRepository)
	s.handler = NewDomainHandler(s.mockRepo)
}

// TearDownTest runs after each test
func (s *DomainHandlerTestSuite) TearDownTest() {
	s.mockRepo.AssertExpectations(s.T())
}

// TestDomainHandlerTestSuite runs the test suite
func TestDomainHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DomainHandlerTestSuite))
}

// Helper function to create a test context
func (s *DomainHandlerTestSuite) createContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	return c, rec
}

// Helper function to create a test domain
func (s *DomainHandlerTestSuite) createTestDomain(id uint, name string, active bool) *models.Domain {
	now := time.Now()
	return &models.Domain{
		ID:        id,
		Name:      name,
		IsActive:  active,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// parseAPIResponse parses the API response from the recorder
func parseAPIResponse(rec *httptest.ResponseRecorder) (*response.APIResponse, error) {
	var resp response.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	return &resp, err
}

// parseErrorResponse parses the error response from the recorder
func parseErrorResponse(rec *httptest.ResponseRecorder) (*response.ErrorResponse, error) {
	var resp response.ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	return &resp, err
}

// ==================== Create Tests ====================

// TestCreate_ValidInput tests creating a domain with valid input
func (s *DomainHandlerTestSuite) TestCreate_ValidInput() {
	// Arrange
	body := `{"name": "example.com"}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Domain")).
		Run(func(args mock.Arguments) {
			domain := args.Get(1).(*models.Domain)
			domain.ID = 1
		}).
		Return(nil)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	resp, err := parseAPIResponse(rec)
	s.NoError(err)
	s.True(resp.Success)
}

// TestCreate_EmptyName tests creating a domain with empty name
func (s *DomainHandlerTestSuite) TestCreate_EmptyName() {
	// Arrange
	body := `{"name": ""}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)

	resp, err := parseErrorResponse(rec)
	s.NoError(err)
	s.False(resp.Success)
	s.Contains(resp.Error, "name is required")
}

// TestCreate_DuplicateName tests creating a domain with duplicate name
func (s *DomainHandlerTestSuite) TestCreate_DuplicateName() {
	// Arrange
	body := `{"name": "example.com"}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Domain")).
		Return(repository.ErrDuplicateEntry)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusConflict, rec.Code)

	resp, err := parseErrorResponse(rec)
	s.NoError(err)
	s.False(resp.Success)
	s.Contains(resp.Error, "already exists")
}

// TestCreate_InvalidJSON tests creating a domain with invalid JSON
func (s *DomainHandlerTestSuite) TestCreate_InvalidJSON() {
	// Arrange
	body := `{invalid json}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestCreate_WithIsActive tests creating a domain with is_active field
func (s *DomainHandlerTestSuite) TestCreate_WithIsActive() {
	// Arrange
	body := `{"name": "example.com", "is_active": false}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	s.mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Name == "example.com" && !d.IsActive
	})).
		Run(func(args mock.Arguments) {
			domain := args.Get(1).(*models.Domain)
			domain.ID = 1
		}).
		Return(nil)

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)
}

// TestCreate_InternalError tests creating a domain when repository returns error
func (s *DomainHandlerTestSuite) TestCreate_InternalError() {
	// Arrange
	body := `{"name": "example.com"}`
	c, rec := s.createContext(http.MethodPost, "/api/domains", body)

	s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Domain")).
		Return(errors.New("database error"))

	// Act
	err := s.handler.Create(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}


// ==================== Get Tests ====================

// TestGet_ValidID tests getting a domain with valid ID
func (s *DomainHandlerTestSuite) TestGet_ValidID() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	c, rec := s.createContext(http.MethodGet, "/api/domains/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	resp, err := parseAPIResponse(rec)
	s.NoError(err)
	s.True(resp.Success)
}

// TestGet_NonExistentID tests getting a domain with non-existent ID
func (s *DomainHandlerTestSuite) TestGet_NonExistentID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/domains/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)

	resp, err := parseErrorResponse(rec)
	s.NoError(err)
	s.False(resp.Success)
	s.Contains(resp.Error, "not found")
}

// TestGet_InvalidID tests getting a domain with invalid ID format
func (s *DomainHandlerTestSuite) TestGet_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/domains/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestGet_InternalError tests getting a domain when repository returns error
func (s *DomainHandlerTestSuite) TestGet_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/domains/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== List Tests ====================

// TestList_Success tests listing all domains
func (s *DomainHandlerTestSuite) TestList_Success() {
	// Arrange
	domains := []models.Domain{
		*s.createTestDomain(1, "example.com", true),
		*s.createTestDomain(2, "test.com", true),
	}
	c, rec := s.createContext(http.MethodGet, "/api/domains", "")

	s.mockRepo.On("List", mock.Anything, false).Return(domains, nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	resp, err := parseAPIResponse(rec)
	s.NoError(err)
	s.True(resp.Success)
}

// TestList_ActiveOnly tests listing only active domains
func (s *DomainHandlerTestSuite) TestList_ActiveOnly() {
	// Arrange
	domains := []models.Domain{
		*s.createTestDomain(1, "example.com", true),
	}
	c, rec := s.createContext(http.MethodGet, "/api/domains?active_only=true", "")

	s.mockRepo.On("List", mock.Anything, true).Return(domains, nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

// TestList_EmptyResult tests listing domains when none exist
func (s *DomainHandlerTestSuite) TestList_EmptyResult() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/domains", "")

	s.mockRepo.On("List", mock.Anything, false).Return([]models.Domain{}, nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

// TestList_InternalError tests listing domains when repository returns error
func (s *DomainHandlerTestSuite) TestList_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/domains", "")

	s.mockRepo.On("List", mock.Anything, false).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== Update Tests ====================

// TestUpdate_ValidData tests updating a domain with valid data
func (s *DomainHandlerTestSuite) TestUpdate_ValidData() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"name": "updated.com"}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Domain")).Return(nil)

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	resp, err := parseAPIResponse(rec)
	s.NoError(err)
	s.True(resp.Success)
}

// TestUpdate_NonExistentID tests updating a non-existent domain
func (s *DomainHandlerTestSuite) TestUpdate_NonExistentID() {
	// Arrange
	body := `{"name": "updated.com"}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/999", body)
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestUpdate_DuplicateName tests updating a domain with duplicate name
func (s *DomainHandlerTestSuite) TestUpdate_DuplicateName() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"name": "existing.com"}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Domain")).Return(repository.ErrDuplicateEntry)

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusConflict, rec.Code)
}

// TestUpdate_InvalidID tests updating a domain with invalid ID format
func (s *DomainHandlerTestSuite) TestUpdate_InvalidID() {
	// Arrange
	body := `{"name": "updated.com"}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/invalid", body)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestUpdate_InvalidJSON tests updating a domain with invalid JSON
func (s *DomainHandlerTestSuite) TestUpdate_InvalidJSON() {
	// Arrange
	body := `{invalid json}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestUpdate_IsActiveOnly tests updating only is_active field
func (s *DomainHandlerTestSuite) TestUpdate_IsActiveOnly() {
	// Arrange
	domain := s.createTestDomain(1, "example.com", true)
	body := `{"is_active": false}`
	c, rec := s.createContext(http.MethodPut, "/api/domains/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("GetByID", mock.Anything, uint(1)).Return(domain, nil)
	s.mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Name == "example.com" && !d.IsActive
	})).Return(nil)

	// Act
	err := s.handler.Update(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

// ==================== Delete Tests ====================

// TestDelete_ValidID tests deleting a domain with valid ID
func (s *DomainHandlerTestSuite) TestDelete_ValidID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/domains/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNoContent, rec.Code)
}

// TestDelete_NonExistentID tests deleting a non-existent domain
func (s *DomainHandlerTestSuite) TestDelete_NonExistentID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/domains/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockRepo.On("Delete", mock.Anything, uint(999)).Return(repository.ErrNotFound)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestDelete_InvalidID tests deleting a domain with invalid ID format
func (s *DomainHandlerTestSuite) TestDelete_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/domains/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestDelete_InternalError tests deleting a domain when repository returns error
func (s *DomainHandlerTestSuite) TestDelete_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/domains/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockRepo.On("Delete", mock.Anything, uint(1)).Return(errors.New("database error"))

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}
