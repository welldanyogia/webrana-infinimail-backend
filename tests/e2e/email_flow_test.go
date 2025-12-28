//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/welldanyogia/webrana-infinimail-backend/internal/smtp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// E2ETestSuite tests complete email flow from SMTP to API
type E2ETestSuite struct {
	suite.Suite
	container      testcontainers.Container
	db             *gorm.DB
	echo           *echo.Echo
	smtpServer     *smtp.Server
	smtpAddr       string
	domainRepo     repository.DomainRepository
	mailboxRepo    repository.MailboxRepository
	messageRepo    repository.MessageRepository
	attachmentRepo repository.AttachmentRepository
	domainHandler  *handlers.DomainHandler
	mailboxHandler *handlers.MailboxHandler
	messageHandler *handlers.MessageHandler
}

// SetupSuite starts PostgreSQL container, SMTP server, and API handlers
func (s *E2ETestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "infinimail_e2e_test",
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

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=infinimail_e2e_test sslmode=disable",
		host, port.Port())

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)
	s.db = db

	// Run migrations
	err = db.AutoMigrate(&models.Domain{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
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

	// Setup Echo
	s.echo = echo.New()

	// Start SMTP server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err)
	s.smtpAddr = listener.Addr().String()
	listener.Close()

	// Create SMTP server
	s.smtpServer = smtp.NewServer(
		s.domainRepo,
		s.mailboxRepo,
		s.messageRepo,
		nil, // No file storage for tests
		nil, // No websocket hub for tests
		true, // Auto-provisioning enabled
	)

	// Start SMTP server in background
	go func() {
		s.smtpServer.ListenAndServe(s.smtpAddr)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
}

// TearDownSuite stops all services
func (s *E2ETestSuite) TearDownSuite() {
	if s.smtpServer != nil {
		s.smtpServer.Close()
	}
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

// SetupTest cleans up data before each test
func (s *E2ETestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE attachments, messages, mailboxes, domains RESTART IDENTITY CASCADE")
}

// TestE2ETestSuite runs the test suite
func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	suite.Run(t, new(E2ETestSuite))
}

// Helper functions
func (s *E2ETestSuite) connectSMTP() (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", s.smtpAddr, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	reader := bufio.NewReader(conn)
	return conn, reader, nil
}

func (s *E2ETestSuite) readSMTPResponse(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (s *E2ETestSuite) sendSMTPCommand(conn net.Conn, cmd string) error {
	_, err := conn.Write([]byte(cmd + "\r\n"))
	return err
}

// ==================== Complete Email Flow Tests ====================

func (s *E2ETestSuite) TestE2E_CompleteEmailFlow() {
	ctx := context.Background()

	// Step 1: Create domain via API
	domainBody := map[string]interface{}{"name": "e2e-test.com"}
	jsonBody, _ := json.Marshal(domainBody)

	req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := s.domainHandler.Create(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	var domainResp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &domainResp)
	require.NoError(s.T(), err)

	// Step 2: Send email via SMTP
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = s.sendSMTPCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = s.sendSMTPCommand(conn, "MAIL FROM:<sender@external.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO
	err = s.sendSMTPCommand(conn, "RCPT TO:<testuser@e2e-test.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// DATA
	err = s.sendSMTPCommand(conn, "DATA")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// Send email content
	emailContent := `From: sender@external.com
To: testuser@e2e-test.com
Subject: E2E Test Email

This is an end-to-end test email.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// QUIT
	err = s.sendSMTPCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Step 3: Verify mailbox was created via API
	mailbox, err := s.mailboxRepo.GetByAddress(ctx, "testuser@e2e-test.com")
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox)

	// Step 4: List messages via API
	req = httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+fmt.Sprint(mailbox.ID)+"/messages", nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("mailbox_id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	err = s.messageHandler.List(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Step 5: Get message and verify content
	messages, _, err := s.messageRepo.ListByMailbox(ctx, mailbox.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), messages, 1)
	assert.Equal(s.T(), "E2E Test Email", messages[0].Subject)
	assert.False(s.T(), messages[0].IsRead)

	// Step 6: Read message via API (should mark as read)
	message, err := s.messageRepo.GetByID(ctx, messages[0].ID)
	require.NoError(s.T(), err)

	req = httptest.NewRequest(http.MethodGet, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	err = s.messageHandler.Get(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Step 7: Verify message is now read
	message, err = s.messageRepo.GetByID(ctx, message.ID)
	require.NoError(s.T(), err)
	assert.True(s.T(), message.IsRead)
}

func (s *E2ETestSuite) TestE2E_DomainManagementWorkflow() {
	ctx := context.Background()

	// Step 1: Create domain
	domainBody := map[string]interface{}{"name": "workflow-test.com"}
	jsonBody, _ := json.Marshal(domainBody)

	req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := s.domainHandler.Create(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	// Get domain ID
	domain, err := s.domainRepo.GetByName(ctx, "workflow-test.com")
	require.NoError(s.T(), err)

	// Step 2: List domains
	req = httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)

	err = s.domainHandler.List(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Step 3: Update domain (deactivate)
	updateBody := map[string]interface{}{"name": "workflow-test.com", "is_active": false}
	jsonBody, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPut, "/api/domains/"+fmt.Sprint(domain.ID), bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	err = s.domainHandler.Update(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify update
	domain, err = s.domainRepo.GetByID(ctx, domain.ID)
	require.NoError(s.T(), err)
	assert.False(s.T(), domain.IsActive)

	// Step 4: Delete domain
	req = httptest.NewRequest(http.MethodDelete, "/api/domains/"+fmt.Sprint(domain.ID), nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(domain.ID))

	err = s.domainHandler.Delete(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, rec.Code)

	// Verify deletion
	_, err = s.domainRepo.GetByID(ctx, domain.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

func (s *E2ETestSuite) TestE2E_MailboxCreationAndEmailReceiving() {
	ctx := context.Background()

	// Step 1: Create domain
	domain := &models.Domain{Name: "mailbox-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Step 2: Create mailbox via API
	mailboxBody := map[string]interface{}{
		"local_part": "inbox",
		"domain_id":  domain.ID,
	}
	jsonBody, _ := json.Marshal(mailboxBody)

	req := httptest.NewRequest(http.MethodPost, "/api/mailboxes", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err = s.mailboxHandler.Create(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	// Verify mailbox was created
	mailbox, err := s.mailboxRepo.GetByAddress(ctx, "inbox@mailbox-test.com")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "inbox", mailbox.LocalPart)

	// Step 3: Send email via SMTP to the created mailbox
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = s.sendSMTPCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = s.sendSMTPCommand(conn, "MAIL FROM:<external@sender.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO
	err = s.sendSMTPCommand(conn, "RCPT TO:<inbox@mailbox-test.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// DATA
	err = s.sendSMTPCommand(conn, "DATA")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// Send email content
	emailContent := `From: external@sender.com
To: inbox@mailbox-test.com
Subject: Mailbox Test Email

Testing mailbox email receiving.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// QUIT
	err = s.sendSMTPCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Step 4: Verify message was received
	messages, total, err := s.messageRepo.ListByMailbox(ctx, mailbox.ID, 10, 0)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), total)
	require.Len(s.T(), messages, 1)
	assert.Equal(s.T(), "Mailbox Test Email", messages[0].Subject)

	// Step 5: Get mailbox via API and verify unread count
	req = httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+fmt.Sprint(mailbox.ID), nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	err = s.mailboxHandler.Get(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var mailboxResp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &mailboxResp)
	require.NoError(s.T(), err)
	assert.True(s.T(), mailboxResp.Success)
}

func (s *E2ETestSuite) TestE2E_MessageReadingAndMarkAsRead() {
	ctx := context.Background()

	// Setup: Create domain, mailbox, and message
	domain := &models.Domain{Name: "read-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "reader",
		DomainID:    domain.ID,
		FullAddress: "reader@read-test.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@external.com",
		SenderName:  "Test Sender",
		Subject:     "Read Test Message",
		Snippet:     "This is a test message for reading...",
		BodyText:    "This is a test message for reading functionality.",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message)
	require.NoError(s.T(), err)

	// Step 1: List messages - should show unread
	req := httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+fmt.Sprint(mailbox.ID)+"/messages", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("mailbox_id")
	c.SetParamValues(fmt.Sprint(mailbox.ID))

	err = s.messageHandler.List(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Step 2: Get message - should auto mark as read
	req = httptest.NewRequest(http.MethodGet, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	err = s.messageHandler.Get(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify message is now read
	updatedMessage, err := s.messageRepo.GetByID(ctx, message.ID)
	require.NoError(s.T(), err)
	assert.True(s.T(), updatedMessage.IsRead)

	// Step 3: Create another unread message
	message2 := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "another@external.com",
		Subject:     "Another Test Message",
		BodyText:    "Another test message body.",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message2)
	require.NoError(s.T(), err)

	// Step 4: Mark as read via API
	req = httptest.NewRequest(http.MethodPatch, "/api/messages/"+fmt.Sprint(message2.ID)+"/read", nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message2.ID))

	err = s.messageHandler.MarkAsRead(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Verify message2 is now read
	updatedMessage2, err := s.messageRepo.GetByID(ctx, message2.ID)
	require.NoError(s.T(), err)
	assert.True(s.T(), updatedMessage2.IsRead)

	// Step 5: Delete message
	req = httptest.NewRequest(http.MethodDelete, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec = httptest.NewRecorder()
	c = s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	err = s.messageHandler.Delete(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, rec.Code)

	// Verify message is deleted
	_, err = s.messageRepo.GetByID(ctx, message.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

func (s *E2ETestSuite) TestE2E_AttachmentDownloadFlow() {
	ctx := context.Background()

	// Setup: Create domain, mailbox, message with attachment
	domain := &models.Domain{Name: "attachment-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "attachments",
		DomainID:    domain.ID,
		FullAddress: "attachments@attachment-test.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@external.com",
		Subject:     "Email with Attachment",
		BodyText:    "Please see attached file.",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message)
	require.NoError(s.T(), err)

	// Create attachment record (without actual file for this test)
	attachment := &models.Attachment{
		MessageID:   message.ID,
		Filename:    "test-document.pdf",
		ContentType: "application/pdf",
		FilePath:    "/tmp/test-document.pdf",
		SizeBytes:   1024,
	}
	err = s.attachmentRepo.Create(ctx, attachment)
	require.NoError(s.T(), err)

	// Step 1: Get message with attachments
	req := httptest.NewRequest(http.MethodGet, "/api/messages/"+fmt.Sprint(message.ID), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(message.ID))

	err = s.messageHandler.Get(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	// Step 2: List attachments for message
	attachments, err := s.attachmentRepo.ListByMessage(ctx, message.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), attachments, 1)
	assert.Equal(s.T(), "test-document.pdf", attachments[0].Filename)
	assert.Equal(s.T(), "application/pdf", attachments[0].ContentType)
	assert.Equal(s.T(), int64(1024), attachments[0].SizeBytes)

	// Step 3: Get attachment metadata
	fetchedAttachment, err := s.attachmentRepo.GetByID(ctx, attachment.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), attachment.Filename, fetchedAttachment.Filename)
	assert.Equal(s.T(), attachment.ContentType, fetchedAttachment.ContentType)
}

func (s *E2ETestSuite) TestE2E_MultipleRecipientsEmail() {
	ctx := context.Background()

	// Setup: Create domain
	domain := &models.Domain{Name: "multi-rcpt.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Send email via SMTP to multiple recipients
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = s.sendSMTPCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = s.sendSMTPCommand(conn, "MAIL FROM:<sender@external.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO - first recipient
	err = s.sendSMTPCommand(conn, "RCPT TO:<user1@multi-rcpt.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO - second recipient
	err = s.sendSMTPCommand(conn, "RCPT TO:<user2@multi-rcpt.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// DATA
	err = s.sendSMTPCommand(conn, "DATA")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// Send email content
	emailContent := `From: sender@external.com
To: user1@multi-rcpt.com, user2@multi-rcpt.com
Subject: Multi-Recipient Test

This email is sent to multiple recipients.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// QUIT
	err = s.sendSMTPCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify both mailboxes were created
	mailbox1, err := s.mailboxRepo.GetByAddress(ctx, "user1@multi-rcpt.com")
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox1)

	mailbox2, err := s.mailboxRepo.GetByAddress(ctx, "user2@multi-rcpt.com")
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox2)

	// Verify both mailboxes received the message
	messages1, _, err := s.messageRepo.ListByMailbox(ctx, mailbox1.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), messages1, 1)
	assert.Equal(s.T(), "Multi-Recipient Test", messages1[0].Subject)

	messages2, _, err := s.messageRepo.ListByMailbox(ctx, mailbox2.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), messages2, 1)
	assert.Equal(s.T(), "Multi-Recipient Test", messages2[0].Subject)
}

func (s *E2ETestSuite) TestE2E_SMTPRejectsInvalidDomain() {
	// Try to send email to non-existent domain
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = s.sendSMTPCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = s.sendSMTPCommand(conn, "MAIL FROM:<sender@external.com>")
	require.NoError(s.T(), err)
	_, err = s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO - non-existent domain should be rejected
	err = s.sendSMTPCommand(conn, "RCPT TO:<user@nonexistent-domain.com>")
	require.NoError(s.T(), err)
	response, err := s.readSMTPResponse(reader)
	require.NoError(s.T(), err)

	// Should get 550 error
	assert.True(s.T(), strings.HasPrefix(response, "550"), "Expected 550 error for non-existent domain, got: %s", response)
}

func (s *E2ETestSuite) TestE2E_RandomMailboxCreation() {
	ctx := context.Background()

	// Setup: Create domain
	domain := &models.Domain{Name: "random-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Create random mailbox via API
	randomBody := map[string]interface{}{
		"domain_id": domain.ID,
	}
	jsonBody, _ := json.Marshal(randomBody)

	req := httptest.NewRequest(http.MethodPost, "/api/mailboxes/random", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err = s.mailboxHandler.CreateRandom(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	// Verify response
	var mailboxResp response.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &mailboxResp)
	require.NoError(s.T(), err)
	assert.True(s.T(), mailboxResp.Success)

	// Verify mailbox was created with random local part
	mailboxes, _, err := s.mailboxRepo.ListByDomain(ctx, domain.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), mailboxes, 1)
	assert.Len(s.T(), mailboxes[0].LocalPart, 8) // Random local part should be 8 chars
}
