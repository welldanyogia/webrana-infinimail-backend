//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/handlers"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// APIIntegrationTestSuite tests API handlers with real database
type APIIntegrationTestSuite struct {
	suite.Suite
	container         testcontainers.Container
	db                *gorm.DB
	echo              *echo.Echo
	domainHandler     *handlers.DomainHandler
	mailboxHandler    *handlers.MailboxHandler
	messageHandler    *handlers.MessageHandler
	attachmentHandler *handlers.AttachmentHandler
	domainRepo        repository.DomainRepository
	mailboxRepo       repository.MailboxRepository
	messageRepo       repository.MessageRepository
	attachmentRepo    repository.AttachmentRepository
}

// SetupSuite starts PostgreSQL container and initializes API handlers
func (s *APIIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "infinimail_api_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.container = container

	// Get connection details
	host, err := container.Host(ctx)
	require.NoError(s.T(), err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(s.T(), err)

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=infinimail_api_test sslmode=disable",
		host, port.Port())

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)
	s.db = db

	// Run migrations
	err = db.AutoMigrate(&models.Domain{}, &models.DomainCertificate{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
	require.NoError(s.T(), err)

	// Initialize repositories
	s.domainRepo = repository.NewDomainRepository(db)
	s.mailboxRepo = repository.NewMailboxRepository(db)
	s.messageRepo = repository.NewMessageRepository(db)
	s.attachmentRepo = repository.NewAttachmentRepository(db, nil)

	// Initialize handlers
	s.domainHandler = handlers.NewDomainHandler(s.domainRepo)
	s.mailboxHandler = handlers.NewMailboxHandler(s.mailboxRepo, s.domainRepo, s.messageRepo)
	s.messageHandler = handlers.NewMessageHandler(s.messageRepo, s.mailboxRepo)
	s.attachmentHandler = handlers.NewAttachmentHandler(s.attachmentRepo, nil)

	// Setup Echo
	s.echo = echo.New()
}

// TearDownSuite stops the PostgreSQL container
func (s *APIIntegrationTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

// SetupTest cleans up data before each test
func (s *APIIntegrationTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE attachments, messages, mailboxes, domains RESTART IDENTITY CASCADE")
}

// TestAPIIntegrationTestSuite runs the test suite
func TestAPIIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	suite.Run(t, new(APIIntegrationTestSuite))
}

// ==================== Domain API Tests ====================

func (s *APIIntegrationTestSuite) TestDomainAPI_Create() {
	// Arrange
	body := map[string]interface{}{"name": "api-test.com"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err := s.domainHandler.Create(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	var resp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *APIIntegrationTestSuite) TestDomainAPI_Get() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "get-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/domains/"+fmt.Sprint(domain.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	// Act
	err = s.domainHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *APIIntegrationTestSuite) TestDomainAPI_List() {
	ctx := context.Background()

	// Create domains
	for i := 0; i < 3; i++ {
		domain := &models.Domain{Name: fmt.Sprintf("list%d.com", i), IsActive: true}
		err := s.domainRepo.Create(ctx, domain)
		require.NoError(s.T(), err)
	}

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err := s.domainHandler.List(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *APIIntegrationTestSuite) TestDomainAPI_Update() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "update-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	body := map[string]interface{}{"name": "updated.com", "is_active": false}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/domains/"+fmt.Sprint(domain.ID), bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	// Act
	err = s.domainHandler.Update(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify update
	updated, err := s.domainRepo.GetByID(ctx, domain.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "updated.com", updated.Name)
	assert.False(s.T(), updated.IsActive)
}

func (s *APIIntegrationTestSuite) TestDomainAPI_Delete() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "delete-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/api/domains/"+fmt.Sprint(domain.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	// Act
	err = s.domainHandler.Delete(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, rec.Code)

	// Verify deletion
	_, err = s.domainRepo.GetByID(ctx, domain.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

// ==================== Mailbox API Tests ====================

func (s *APIIntegrationTestSuite) TestMailboxAPI_Create() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "mailbox-api.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	body := map[string]interface{}{
		"local_part": "testuser",
		"domain_id":  domain.ID,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/mailboxes", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err = s.mailboxHandler.Create(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	var resp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *APIIntegrationTestSuite) TestMailboxAPI_CreateRandom() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "random-api.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	body := map[string]interface{}{"domain_id": domain.ID}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/mailboxes/random", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err = s.mailboxHandler.CreateRandom(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)
}

func (s *APIIntegrationTestSuite) TestMailboxAPI_Get() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "get-mailbox.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@get-mailbox.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+fmt.Sprint(mailbox.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	// Act
	err = s.mailboxHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

func (s *APIIntegrationTestSuite) TestMailboxAPI_List() {
	ctx := context.Background()

	// Create domain and mailboxes
	domain := &models.Domain{Name: "list-mailbox.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	for i := 0; i < 3; i++ {
		mailbox := &models.Mailbox{
			LocalPart:   fmt.Sprintf("user%d", i),
			DomainID:    domain.ID,
			FullAddress: fmt.Sprintf("user%d@list-mailbox.com", i),
		}
		err = s.mailboxRepo.Create(ctx, mailbox)
		require.NoError(s.T(), err)
	}

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/mailboxes?domain_id="+fmt.Sprint(domain.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.QueryParams().Set("domain_id", fmt.Sprint(domain.ID))

	// Act
	err = s.mailboxHandler.List(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

func (s *APIIntegrationTestSuite) TestMailboxAPI_Delete() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "delete-mailbox.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@delete-mailbox.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/api/mailboxes/"+fmt.Sprint(mailbox.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	// Act
	err = s.mailboxHandler.Delete(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, rec.Code)
}

// ==================== Message API Tests ====================

func (s *APIIntegrationTestSuite) TestMessageAPI_List() {
	ctx := context.Background()

	// Create domain, mailbox, and messages
	domain := &models.Domain{Name: "message-api.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@message-api.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	for i := 0; i < 3; i++ {
		message := &models.Message{
			MailboxID:   mailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     fmt.Sprintf("Message %d", i),
		}
		err = s.messageRepo.Create(ctx, message)
		require.NoError(s.T(), err)
	}

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+fmt.Sprint(mailbox.ID)+"/messages", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("mailbox_id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	// Act
	err = s.messageHandler.List(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

func (s *APIIntegrationTestSuite) TestMessageAPI_Get() {
	ctx := context.Background()

	// Create domain, mailbox, and message
	domain := &models.Domain{Name: "get-message.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@get-message.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Test Message",
		BodyText:    "Test body",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	// Act
	err = s.messageHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify auto-mark as read
	updated, err := s.messageRepo.GetByID(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), updated.IsRead)
}

func (s *APIIntegrationTestSuite) TestMessageAPI_MarkAsRead() {
	ctx := context.Background()

	// Create domain, mailbox, and message
	domain := &models.Domain{Name: "mark-read.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@mark-read.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Unread Message",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodPatch, "/api/messages/"+fmt.Sprint(message.ID)+"/read", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	// Act
	err = s.messageHandler.MarkAsRead(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify
	updated, err := s.messageRepo.GetByID(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), updated.IsRead)
}

func (s *APIIntegrationTestSuite) TestMessageAPI_Delete() {
	ctx := context.Background()

	// Create domain, mailbox, and message
	domain := &models.Domain{Name: "delete-message.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@delete-message.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "To Delete",
	}
	err = s.messageRepo.Create(ctx, message)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	// Act
	err = s.messageHandler.Delete(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, rec.Code)
}

// ==================== Attachment API Tests ====================

func (s *APIIntegrationTestSuite) TestAttachmentAPI_List() {
	ctx := context.Background()

	// Create domain, mailbox, message with attachments
	domain := &models.Domain{Name: "attachment-api.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@attachment-api.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", FilePath: "/path/doc.pdf", SizeBytes: 1024},
	}
	err = s.messageRepo.CreateWithAttachments(ctx, message, attachments)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/messages/"+fmt.Sprint(message.ID)+"/attachments", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("message_id")
	c.SetParamValues(fmt.Sprint(message.ID))

	// Act
	err = s.attachmentHandler.List(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

func (s *APIIntegrationTestSuite) TestAttachmentAPI_Get() {
	ctx := context.Background()

	// Create domain, mailbox, message with attachment
	domain := &models.Domain{Name: "get-attachment.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@get-attachment.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachment",
	}
	attachments := []models.Attachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", FilePath: "/path/doc.pdf", SizeBytes: 1024},
	}
	err = s.messageRepo.CreateWithAttachments(ctx, message, attachments)
	require.NoError(s.T(), err)

	// Get attachment ID
	atts, err := s.attachmentRepo.ListByMessage(ctx, message.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), atts, 1)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/attachments/"+fmt.Sprint(atts[0].ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(atts[0].ID))

	// Act
	err = s.attachmentHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

// ==================== Health Check Tests ====================

func (s *APIIntegrationTestSuite) TestHealthAPI_Check() {
	healthHandler := handlers.NewHealthHandler(s.db)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err := healthHandler.Health(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

func (s *APIIntegrationTestSuite) TestHealthAPI_Ready() {
	healthHandler := handlers.NewHealthHandler(s.db)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Act
	err := healthHandler.Ready(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

// ==================== JSON Response Format Tests ====================

func (s *APIIntegrationTestSuite) TestAPI_ResponseFormat_Success() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "response-format.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/domains/"+fmt.Sprint(domain.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	// Act
	err = s.domainHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)

	// Verify response format
	assert.Contains(s.T(), resp, "success")
	assert.Contains(s.T(), resp, "data")
	assert.Equal(s.T(), true, resp["success"])
}

func (s *APIIntegrationTestSuite) TestAPI_ResponseFormat_NotFound() {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/domains/99999", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	// Act
	err := s.domainHandler.Get(c)

	// Assert
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)

	// Verify error response format
	assert.Contains(s.T(), resp, "success")
	assert.Contains(s.T(), resp, "error")
	assert.Equal(s.T(), false, resp["success"])
}
