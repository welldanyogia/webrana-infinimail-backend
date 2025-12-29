//go:build integration

package integration

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/smtp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SMTPIntegrationTestSuite tests SMTP server with real database
type SMTPIntegrationTestSuite struct {
	suite.Suite
	container    testcontainers.Container
	db           *gorm.DB
	smtpServer   *smtp.Server
	smtpAddr     string
	domainRepo   repository.DomainRepository
	mailboxRepo  repository.MailboxRepository
	messageRepo  repository.MessageRepository
}

// SetupSuite starts PostgreSQL container and SMTP server
func (s *SMTPIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "infinimail_smtp_test",
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

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=infinimail_smtp_test sslmode=disable",
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

// TearDownSuite stops SMTP server and PostgreSQL container
func (s *SMTPIntegrationTestSuite) TearDownSuite() {
	if s.smtpServer != nil {
		s.smtpServer.Close()
	}
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

// SetupTest cleans up data before each test
func (s *SMTPIntegrationTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE attachments, messages, mailboxes, domains RESTART IDENTITY CASCADE")
}

// TestSMTPIntegrationTestSuite runs the test suite
func TestSMTPIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	suite.Run(t, new(SMTPIntegrationTestSuite))
}

// Helper function to connect to SMTP server
func (s *SMTPIntegrationTestSuite) connectSMTP() (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", s.smtpAddr, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	reader := bufio.NewReader(conn)
	return conn, reader, nil
}

// Helper function to read SMTP response
func readResponse(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Helper function to send SMTP command
func sendCommand(conn net.Conn, cmd string) error {
	_, err := conn.Write([]byte(cmd + "\r\n"))
	return err
}

// ==================== Connection Tests ====================

func (s *SMTPIntegrationTestSuite) TestSMTP_AcceptsConnection() {
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	response, err := readResponse(reader)
	require.NoError(s.T(), err)

	assert.True(s.T(), strings.HasPrefix(response, "220"))
}

func (s *SMTPIntegrationTestSuite) TestSMTP_Banner() {
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	response, err := readResponse(reader)
	require.NoError(s.T(), err)

	assert.Contains(s.T(), response, "220")
	assert.Contains(s.T(), response, "ESMTP")
}

func (s *SMTPIntegrationTestSuite) TestSMTP_EHLO() {
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// Send EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)

	// Read response (may be multi-line)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)

	assert.True(s.T(), strings.HasPrefix(response, "250"))
}

// ==================== RCPT TO Tests ====================

func (s *SMTPIntegrationTestSuite) TestSMTP_RCPT_ValidDomain() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "valid-domain.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Connect and send commands
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = sendCommand(conn, "MAIL FROM:<sender@example.com>")
	require.NoError(s.T(), err)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "250"))

	// RCPT TO with valid domain
	err = sendCommand(conn, "RCPT TO:<user@valid-domain.com>")
	require.NoError(s.T(), err)
	response, err = readResponse(reader)
	require.NoError(s.T(), err)

	assert.True(s.T(), strings.HasPrefix(response, "250"))
}

func (s *SMTPIntegrationTestSuite) TestSMTP_RCPT_InvalidDomain() {
	// Connect and send commands
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = sendCommand(conn, "MAIL FROM:<sender@example.com>")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO with invalid domain
	err = sendCommand(conn, "RCPT TO:<user@nonexistent-domain.com>")
	require.NoError(s.T(), err)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)

	// Should reject with 550
	assert.True(s.T(), strings.HasPrefix(response, "550") || strings.HasPrefix(response, "551"))
}

// ==================== Email Delivery Tests ====================

func (s *SMTPIntegrationTestSuite) TestSMTP_DeliverEmail() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "delivery-test.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Connect and send email
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = sendCommand(conn, "MAIL FROM:<sender@example.com>")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO
	err = sendCommand(conn, "RCPT TO:<testuser@delivery-test.com>")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// DATA
	err = sendCommand(conn, "DATA")
	require.NoError(s.T(), err)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "354"))

	// Send email content
	emailContent := `From: sender@example.com
To: testuser@delivery-test.com
Subject: Test Email

This is a test email body.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)

	// Read response
	response, err = readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "250"))

	// QUIT
	err = sendCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for message to be stored
	time.Sleep(100 * time.Millisecond)

	// Verify mailbox was created (auto-provisioning)
	mailbox, err := s.mailboxRepo.GetByAddress(ctx, "testuser@delivery-test.com")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox)

	// Verify message was stored
	messages, total, err := s.messageRepo.ListByMailbox(ctx, mailbox.ID, 10, 0)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), total)
	assert.Len(s.T(), messages, 1)
	assert.Equal(s.T(), "Test Email", messages[0].Subject)
}

func (s *SMTPIntegrationTestSuite) TestSMTP_AutoProvisioning_CreatesMailbox() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "auto-provision.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Verify mailbox doesn't exist
	_, err = s.mailboxRepo.GetByAddress(ctx, "newuser@auto-provision.com")
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)

	// Connect and send email
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = sendCommand(conn, "MAIL FROM:<sender@example.com>")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO (should auto-provision)
	err = sendCommand(conn, "RCPT TO:<newuser@auto-provision.com>")
	require.NoError(s.T(), err)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "250"))

	// DATA
	err = sendCommand(conn, "DATA")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// Send email content
	emailContent := `From: sender@example.com
To: newuser@auto-provision.com
Subject: Auto Provision Test

Test body.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// QUIT
	err = sendCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify mailbox was created
	mailbox, err := s.mailboxRepo.GetByAddress(ctx, "newuser@auto-provision.com")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox)
	assert.Equal(s.T(), "newuser", mailbox.LocalPart)
}

// ==================== Multiple Recipients Tests ====================

func (s *SMTPIntegrationTestSuite) TestSMTP_MultipleRecipients() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "multi-rcpt.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Connect and send email to multiple recipients
	conn, reader, err := s.connectSMTP()
	require.NoError(s.T(), err)
	defer conn.Close()

	// Read banner
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// EHLO
	err = sendCommand(conn, "EHLO localhost")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// MAIL FROM
	err = sendCommand(conn, "MAIL FROM:<sender@example.com>")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// RCPT TO - first recipient
	err = sendCommand(conn, "RCPT TO:<user1@multi-rcpt.com>")
	require.NoError(s.T(), err)
	response, err := readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "250"))

	// RCPT TO - second recipient
	err = sendCommand(conn, "RCPT TO:<user2@multi-rcpt.com>")
	require.NoError(s.T(), err)
	response, err = readResponse(reader)
	require.NoError(s.T(), err)
	assert.True(s.T(), strings.HasPrefix(response, "250"))

	// DATA
	err = sendCommand(conn, "DATA")
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// Send email content
	emailContent := `From: sender@example.com
To: user1@multi-rcpt.com, user2@multi-rcpt.com
Subject: Multi Recipient Test

Test body.
.`
	_, err = conn.Write([]byte(emailContent + "\r\n"))
	require.NoError(s.T(), err)
	_, err = readResponse(reader)
	require.NoError(s.T(), err)

	// QUIT
	err = sendCommand(conn, "QUIT")
	require.NoError(s.T(), err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify both mailboxes were created
	mailbox1, err := s.mailboxRepo.GetByAddress(ctx, "user1@multi-rcpt.com")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox1)

	mailbox2, err := s.mailboxRepo.GetByAddress(ctx, "user2@multi-rcpt.com")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), mailbox2)
}
