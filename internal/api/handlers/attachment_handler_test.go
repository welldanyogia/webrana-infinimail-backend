package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// AttachmentHandlerTestSuite is the test suite for AttachmentHandler
type AttachmentHandlerTestSuite struct {
	suite.Suite
	echo               *echo.Echo
	handler            *AttachmentHandler
	mockAttachmentRepo *mocks.MockAttachmentRepository
	mockMessageRepo    *mocks.MockMessageRepository
	mockFileStorage    *mocks.MockFileStorage
}

// SetupTest runs before each test
func (s *AttachmentHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.mockAttachmentRepo = new(mocks.MockAttachmentRepository)
	s.mockMessageRepo = new(mocks.MockMessageRepository)
	s.mockFileStorage = new(mocks.MockFileStorage)
	s.handler = NewAttachmentHandler(s.mockAttachmentRepo, s.mockMessageRepo, s.mockFileStorage)
}

// TearDownTest runs after each test
func (s *AttachmentHandlerTestSuite) TearDownTest() {
	s.mockAttachmentRepo.AssertExpectations(s.T())
	s.mockMessageRepo.AssertExpectations(s.T())
	s.mockFileStorage.AssertExpectations(s.T())
}

// TestAttachmentHandlerTestSuite runs the test suite
func TestAttachmentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AttachmentHandlerTestSuite))
}

// Helper function to create a test context
func (s *AttachmentHandlerTestSuite) createContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	return c, rec
}

// Helper function to create a test message
func (s *AttachmentHandlerTestSuite) createTestMessage(id uint, mailboxID uint) *models.Message {
	now := time.Now()
	return &models.Message{
		ID:          id,
		MailboxID:   mailboxID,
		SenderEmail: "sender@external.com",
		SenderName:  "Test Sender",
		Subject:     "Test Subject",
		ReceivedAt:  now,
	}
}

// Helper function to create a test attachment
func (s *AttachmentHandlerTestSuite) createTestAttachment(id uint, messageID uint) *models.Attachment {
	return &models.Attachment{
		ID:          id,
		MessageID:   messageID,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		FilePath:    "/attachments/abc123.pdf",
		SizeBytes:   1024,
	}
}

// mockReadCloser is a helper type for mocking io.ReadCloser
type mockReadCloser struct {
	*bytes.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(data []byte) io.ReadCloser {
	return &mockReadCloser{bytes.NewReader(data)}
}

// ==================== List Tests ====================

// TestList_Success tests listing attachments for a message
func (s *AttachmentHandlerTestSuite) TestList_Success() {
	// Arrange
	message := s.createTestMessage(1, 1)
	attachments := []models.Attachment{
		*s.createTestAttachment(1, 1),
		{
			ID:          2,
			MessageID:   1,
			Filename:    "image.png",
			ContentType: "image/png",
			FilePath:    "/attachments/def456.png",
			SizeBytes:   2048,
		},
	}
	c, rec := s.createContext(http.MethodGet, "/api/messages/1/attachments", "")
	c.SetParamNames("message_id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)
	s.mockAttachmentRepo.On("ListByMessage", mock.Anything, uint(1)).Return(attachments, nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestList_MessageNotFound tests listing attachments for non-existent message
func (s *AttachmentHandlerTestSuite) TestList_MessageNotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/messages/999/attachments", "")
	c.SetParamNames("message_id")
	c.SetParamValues("999")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestList_InvalidMessageID tests listing attachments with invalid message ID
func (s *AttachmentHandlerTestSuite) TestList_InvalidMessageID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/messages/invalid/attachments", "")
	c.SetParamNames("message_id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestList_InternalError tests listing attachments when repository returns error
func (s *AttachmentHandlerTestSuite) TestList_InternalError() {
	// Arrange
	message := s.createTestMessage(1, 1)
	c, rec := s.createContext(http.MethodGet, "/api/messages/1/attachments", "")
	c.SetParamNames("message_id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)
	s.mockAttachmentRepo.On("ListByMessage", mock.Anything, uint(1)).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// TestList_EmptyResult tests listing attachments when none exist
func (s *AttachmentHandlerTestSuite) TestList_EmptyResult() {
	// Arrange
	message := s.createTestMessage(1, 1)
	c, rec := s.createContext(http.MethodGet, "/api/messages/1/attachments", "")
	c.SetParamNames("message_id")
	c.SetParamValues("1")

	s.mockMessageRepo.On("GetByID", mock.Anything, uint(1)).Return(message, nil)
	s.mockAttachmentRepo.On("ListByMessage", mock.Anything, uint(1)).Return([]models.Attachment{}, nil)

	// Act
	err := s.handler.List(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}


// ==================== Get Tests ====================

// TestGet_Success tests getting attachment metadata
func (s *AttachmentHandlerTestSuite) TestGet_Success() {
	// Arrange
	attachment := s.createTestAttachment(1, 1)
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(attachment, nil)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp response.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.True(resp.Success)
}

// TestGet_NotFound tests getting non-existent attachment
func (s *AttachmentHandlerTestSuite) TestGet_NotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestGet_InvalidID tests getting attachment with invalid ID format
func (s *AttachmentHandlerTestSuite) TestGet_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/invalid", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestGet_InternalError tests getting attachment when repository returns error
func (s *AttachmentHandlerTestSuite) TestGet_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.Get(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// ==================== Download Tests ====================

// TestDownload_Success tests downloading an attachment
func (s *AttachmentHandlerTestSuite) TestDownload_Success() {
	// Arrange
	attachment := s.createTestAttachment(1, 1)
	fileContent := []byte("PDF file content here")
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1/download", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(attachment, nil)
	s.mockFileStorage.On("Get", attachment.FilePath).Return(newMockReadCloser(fileContent), nil)

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/pdf", rec.Header().Get("Content-Type"))
	s.Contains(rec.Header().Get("Content-Disposition"), "document.pdf")
	s.Equal(string(fileContent), rec.Body.String())
}

// TestDownload_SetsContentDisposition tests that Download sets Content-Disposition header
func (s *AttachmentHandlerTestSuite) TestDownload_SetsContentDisposition() {
	// Arrange
	attachment := &models.Attachment{
		ID:          1,
		MessageID:   1,
		Filename:    "report.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		FilePath:    "/attachments/xyz789.xlsx",
		SizeBytes:   4096,
	}
	fileContent := []byte("Excel file content")
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1/download", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(attachment, nil)
	s.mockFileStorage.On("Get", attachment.FilePath).Return(newMockReadCloser(fileContent), nil)

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.Contains(rec.Header().Get("Content-Disposition"), `attachment; filename="report.xlsx"`)
}

// TestDownload_NotFound tests downloading non-existent attachment
func (s *AttachmentHandlerTestSuite) TestDownload_NotFound() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/999/download", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
}

// TestDownload_InvalidID tests downloading attachment with invalid ID format
func (s *AttachmentHandlerTestSuite) TestDownload_InvalidID() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/invalid/download", "")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// TestDownload_FileNotFound tests downloading when file is missing from storage
func (s *AttachmentHandlerTestSuite) TestDownload_FileNotFound() {
	// Arrange
	attachment := s.createTestAttachment(1, 1)
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1/download", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(attachment, nil)
	s.mockFileStorage.On("Get", attachment.FilePath).Return(nil, errors.New("file not found"))

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}

// TestDownload_CorrectContentType tests that Download returns correct Content-Type
func (s *AttachmentHandlerTestSuite) TestDownload_CorrectContentType() {
	// Arrange
	attachment := &models.Attachment{
		ID:          1,
		MessageID:   1,
		Filename:    "image.png",
		ContentType: "image/png",
		FilePath:    "/attachments/img123.png",
		SizeBytes:   2048,
	}
	fileContent := []byte("PNG image data")
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1/download", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(attachment, nil)
	s.mockFileStorage.On("Get", attachment.FilePath).Return(newMockReadCloser(fileContent), nil)

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
	s.Equal("image/png", rec.Header().Get("Content-Type"))
}

// TestDownload_InternalError tests downloading when repository returns error
func (s *AttachmentHandlerTestSuite) TestDownload_InternalError() {
	// Arrange
	c, rec := s.createContext(http.MethodGet, "/api/attachments/1/download", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	s.mockAttachmentRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, errors.New("database error"))

	// Act
	err := s.handler.Download(c)

	// Assert
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)
}
