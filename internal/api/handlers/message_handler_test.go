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

// MessageHandlerTestSuite is the test suite for MessageHandler
type MessageHandlerTestSuite struct {
	suite.Suite
	echo            *echo.Echo
	handler         *MessageHandler
	mockMessageRepo *mocks.MockMessageRepository
	mockMailboxRepo *mocks.MockMailboxRepository
}

// SetupTest runs before each test
func (s *MessageHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.mockMessageRepo = new(mocks.MockMessageRepository)
	s.mockMailboxRepo = new(mocks.MockMailboxRepository)
	s.handler = NewMessageHandler(s.mockMessageRepo, s.mockMailboxRepo)
}

// TearDownTest runs after each test
func (s *MessageHandlerTestSuite) TearDownTest() {
	s.mockMessageRepo.AssertExpectations(s.T())
	s.mockMailboxRepo.AssertExpectations(s.T())
}

// TestMessageHandlerTestSuite runs the test suite
func TestMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MessageHandlerTestSuite))
}

// Helper function to create a test context
func (s *MessageHandlerTestSuite) createContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	return c, rec
}

// Helper function to create a test mailbox
func (s *MessageHandlerTestSuite) createTestMailbox(id uint) *models.Mailbox {
	now := time.Now()
	return &models.Mailbox{
		ID:          id,
		LocalPart:   "user",
		DomainID:    1,
		FullAddress: "user@example.com",
		CreatedAt:   now,
	}
}

// Helper function to create a test message
func (s *MessageHandlerTestSuite) createTestMessage(id uint, mailboxID uint, isRead bool) *models.Message {
	now := time.Now()
	return &models.Message{
		ID:          id,
		MailboxID:   mailboxID,
		SenderEmail: "sender@external.com",
		SenderName:  "Test Sender",
		Subject:     "Test Subject",
		Snippet:     "This is a test email...",
		BodyText:    "This is a test email body.",
		BodyHTML:    "<p>This is a test email body.</p>",
		IsRead:      isRead,
		ReceivedAt:  now,
	}
}

// Helper function to create a test message list item
func (s *MessageHandlerTestSuite) createTestMessageListItem(id uint, mailboxID uint, isRead bool) models.MessageListItem {
	now := time.Now()
	return models.MessageListItem{
		ID:              id,
		MailboxID:       mailboxID,
		SenderEmail:     "sender@external.com",
		SenderName:      "Test Sender",
		Subject:         "Test Subject",
		Snippet:         "This is a test email...",
		IsRead:          isRead,
		ReceivedAt:      now,
		AttachmentCount: 0,
	}
}

// ==================== List Tests ====================

// TestList_Success tests listing messages for a mailbox
func (s *MessageHandlerTestSuite) TestList_Success() {
	// Arrange
	mailbox := s.createTestMailbox(1)
	messages := []models.MessageListItem{
		s.createTestMessageListItem(1, 1, false),
		s.createTestMessageListItem(2, 1, true),
	}
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/1/messages", "")
	c.SetParamNames("mailbox_id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(1)).Return(mailbox, nil)
	s.mockMessageRepo.On("ListByMailbox", mock.Anything, uint(1), 20, 0).Return(messages, int64(2), nil)

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

// TestList_WithPagination tests listing messages with pagination
func (s *MessageHandlerTestSuite) TestList_WithPagination() {
	// Arrange
	mailbox := s.createTestMailbox(1)
	messages := []models.MessageListItem{
		s.createTestMessageListItem(11, 1, false),
	}
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/1/messages?limit=10&offset=10", "")
	c.SetParamNames("mailbox_id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(1)).Return(mailbox, nil)
	s.mockMessageRepo.On("ListByMailbox", mock.Anything, uint(1), 10, 10).Return(messages, int64(15), nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.PaginatedResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
	s.Equal(10, resp.Meta.Limit)
	s.Equal(10, resp.Meta.Offset)
}

// TestList_MailboxNotFound tests listing messages for non-existent mailbox
func (s *MessageHandlerTestSuite) TestList_MailboxNotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/999/messages", "")
	c.SetParamNames("mailbox_id")
	c.SetParamValues("999")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestList_InvalidMailboxID tests listing messages with invalid mailbox ID
func (s *MessageHandlerTestSuite) TestList_InvalidMailboxID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/invalid/messages", "")
	c.SetParamNames("mailbox_id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestList_InternalError tests listing messages when repository returns error
func (s *MessageHandlerTestSuite) TestList_InternalError() {
	// Arrange
	mailbox := s.createTestMailbox(1)
	c, rec := s.createContext(http.MethodGet, "/api/mailboxes/1/messages", "")
	c.SetParamNames("mailbox_id")
	c.SetParamValues("1")

	s.mockMailboxRepo.On("GetByID", mock.Anything, uint(1)).Return(mailbox, nil)
	s.mockMessageRepo.On("ListByMailbox", mock.Anything, uint(1), 20, 0).Return(nil, int64(0), errors.New("database error"))

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}


// ==================== Get Tests ====================

// TestGet_Success tests getting a message with attachments
func (s *MessageHandlerTestSuite) TestGet_Success() {
	// Arrange
	message := s.createTestMessage(1, 1, true)
	message.Attachments = []models.Attachment{
		{
			ID:          1,
			MessageID:   1,
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			FilePath:    "/attachments/abc123.pdf",
			SizeBytes:   1024,
		},
	}
	c, rec := s.createContext(http.MethodGet, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestGet_AutoMarksAsRead tests that Get auto marks unread message as read
func (s *MessageHandlerTestSuite) TestGet_AutoMarksAsRead() {
	// Arrange
	message := s.createTestMessage(1, 1, false) // unread message
	c, rec := s.createContext(http.MethodGet, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)
	s.mockMessageRepo.On("MarkAsRead", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.mockMessageRepo.AssertCalled(s.T(), "MarkAsRead", mock.Anything, uint(1))
}

// TestGet_AlreadyRead tests that Get does not call MarkAsRead for already read message
func (s *MessageHandlerTestSuite) TestGet_AlreadyRead() {
	// Arrange
	message := s.createTestMessage(1, 1, true) // already read
	c, rec := s.createContext(http.MethodGet, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)
	// MarkAsRead should NOT be called

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.mockMessageRepo.AssertNotCalled(s.T(), "MarkAsRead", mock.Anything, mock.Anything)
}

// TestGet_NotFound tests getting a non-existent message
func (s *MessageHandlerTestSuite) TestGet_NotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/messages/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestGet_InvalidID tests getting a message with invalid ID format
func (s *MessageHandlerTestSuite) TestGet_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/messages/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestGet_InternalError tests getting a message when repository returns error
func (s *MessageHandlerTestSuite) TestGet_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== MarkAsRead Tests ====================

// TestMarkAsRead_Success tests marking a message as read
func (s *MessageHandlerTestSuite) TestMarkAsRead_Success() {
	// Arrange
	c, rec := s.createContext(http.MethodPatch, "/api/messages/1/read", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("MarkAsRead", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.MarkAsRead(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
	s.Contains(resp.Message, "marked as read")
}

// TestMarkAsRead_NotFound tests marking a non-existent message as read
func (s *MessageHandlerTestSuite) TestMarkAsRead_NotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodPatch, "/api/messages/999/read", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockMessageRepo.On("MarkAsRead", mock.Anything, uint(999)).Return(repository.ErrNotFound)

	// Act
	err := s.handler.MarkAsRead(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestMarkAsRead_InvalidID tests marking a message with invalid ID format
func (s *MessageHandlerTestSuite) TestMarkAsRead_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodPatch, "/api/messages/invalid/read", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.MarkAsRead(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestMarkAsRead_InternalError tests marking a message when repository returns error
func (s *MessageHandlerTestSuite) TestMarkAsRead_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodPatch, "/api/messages/1/read", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("MarkAsRead", mock.Anything, uint(1)).Return(errors.New("database error"))

	// Act
	err := s.handler.MarkAsRead(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== Delete Tests ====================

// TestDelete_Success tests deleting a message
func (s *MessageHandlerTestSuite) TestDelete_Success() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNoContent, rec.Code)
}

// TestDelete_NotFound tests deleting a non-existent message
func (s *MessageHandlerTestSuite) TestDelete_NotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/messages/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockMessageRepo.On("Delete", mock.Anything, uint(999)).Return(repository.ErrNotFound)

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestDelete_InvalidID tests deleting a message with invalid ID format
func (s *MessageHandlerTestSuite) TestDelete_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/messages/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestDelete_InternalError tests deleting a message when repository returns error
func (s *MessageHandlerTestSuite) TestDelete_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodDelete, "/api/messages/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("Delete", mock.Anything, uint(1)).Return(errors.New("database error"))

	// Act
	err := s.handler.Delete(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}
