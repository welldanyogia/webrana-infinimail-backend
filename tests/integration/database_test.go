//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseIntegrationTestSuite tests database operations with real PostgreSQL
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	container      testcontainers.Container
	db             *gorm.DB
	domainRepo     repository.DomainRepository
	mailboxRepo    repository.MailboxRepository
	messageRepo    repository.MessageRepository
	attachmentRepo repository.AttachmentRepository
}

// SetupSuite starts PostgreSQL container and initializes database
func (s *DatabaseIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "infinimail_test",
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

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=infinimail_test sslmode=disable",
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
}

// TearDownSuite stops the PostgreSQL container
func (s *DatabaseIntegrationTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

// SetupTest cleans up data before each test
func (s *DatabaseIntegrationTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE attachments, messages, mailboxes, domains RESTART IDENTITY CASCADE")
}

// TestDatabaseIntegrationTestSuite runs the test suite
func TestDatabaseIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}

// ==================== Domain CRUD Tests ====================

func (s *DatabaseIntegrationTestSuite) TestDomain_Create() {
	ctx := context.Background()

	domain := &models.Domain{Name: "example.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)

	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), domain.ID)
	assert.NotZero(s.T(), domain.CreatedAt)
	assert.NotZero(s.T(), domain.UpdatedAt)
}

func (s *DatabaseIntegrationTestSuite) TestDomain_GetByID() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "getbyid.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Get by ID
	retrieved, err := s.domainRepo.GetByID(ctx, domain.ID)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), domain.ID, retrieved.ID)
	assert.Equal(s.T(), "getbyid.com", retrieved.Name)
}

func (s *DatabaseIntegrationTestSuite) TestDomain_GetByName() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "getbyname.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Get by name
	retrieved, err := s.domainRepo.GetByName(ctx, "getbyname.com")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), domain.ID, retrieved.ID)
}

func (s *DatabaseIntegrationTestSuite) TestDomain_List() {
	ctx := context.Background()

	// Create domains
	domains := []*models.Domain{
		{Name: "domain1.com", IsActive: true},
		{Name: "domain2.com", IsActive: false},
		{Name: "domain3.com", IsActive: true},
	}
	for _, d := range domains {
		err := s.domainRepo.Create(ctx, d)
		require.NoError(s.T(), err)
	}

	// List all
	all, err := s.domainRepo.List(ctx, false)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), all, 3)

	// List active only
	active, err := s.domainRepo.List(ctx, true)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), active, 2)
}

func (s *DatabaseIntegrationTestSuite) TestDomain_Update() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "original.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Update
	domain.Name = "updated.com"
	domain.IsActive = false
	err = s.domainRepo.Update(ctx, domain)
	assert.NoError(s.T(), err)

	// Verify
	retrieved, err := s.domainRepo.GetByID(ctx, domain.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "updated.com", retrieved.Name)
	assert.False(s.T(), retrieved.IsActive)
}

func (s *DatabaseIntegrationTestSuite) TestDomain_Delete() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "todelete.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Delete
	err = s.domainRepo.Delete(ctx, domain.ID)
	assert.NoError(s.T(), err)

	// Verify
	_, err = s.domainRepo.GetByID(ctx, domain.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

// ==================== Unique Constraint Tests ====================

func (s *DatabaseIntegrationTestSuite) TestDomain_UniqueConstraint() {
	ctx := context.Background()

	// Create first domain
	domain1 := &models.Domain{Name: "unique.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain1)
	require.NoError(s.T(), err)

	// Try to create duplicate
	domain2 := &models.Domain{Name: "unique.com", IsActive: true}
	err = s.domainRepo.Create(ctx, domain2)

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, repository.ErrDuplicateEntry)
}

func (s *DatabaseIntegrationTestSuite) TestMailbox_UniqueConstraint() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "mailbox-unique.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Create first mailbox
	mailbox1 := &models.Mailbox{
		LocalPart:   "user",
		DomainID:    domain.ID,
		FullAddress: "user@mailbox-unique.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox1)
	require.NoError(s.T(), err)

	// Try to create duplicate
	mailbox2 := &models.Mailbox{
		LocalPart:   "user",
		DomainID:    domain.ID,
		FullAddress: "user@mailbox-unique.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox2)

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, repository.ErrDuplicateEntry)
}

// ==================== Mailbox CRUD Tests ====================

func (s *DatabaseIntegrationTestSuite) TestMailbox_CRUD() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "mailbox-crud.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Create mailbox
	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@mailbox-crud.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), mailbox.ID)

	// Get by ID
	retrieved, err := s.mailboxRepo.GetByID(ctx, mailbox.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "test@mailbox-crud.com", retrieved.FullAddress)

	// Get by address
	retrieved, err = s.mailboxRepo.GetByAddress(ctx, "test@mailbox-crud.com")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), mailbox.ID, retrieved.ID)

	// Delete
	err = s.mailboxRepo.Delete(ctx, mailbox.ID)
	assert.NoError(s.T(), err)

	// Verify deletion
	_, err = s.mailboxRepo.GetByID(ctx, mailbox.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

func (s *DatabaseIntegrationTestSuite) TestMailbox_GetOrCreate() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "getorcreate.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// First call creates
	mailbox1, created1, err := s.mailboxRepo.GetOrCreate(ctx, "newuser", domain.ID, "getorcreate.com")
	assert.NoError(s.T(), err)
	assert.True(s.T(), created1)
	assert.NotZero(s.T(), mailbox1.ID)

	// Second call returns existing
	mailbox2, created2, err := s.mailboxRepo.GetOrCreate(ctx, "newuser", domain.ID, "getorcreate.com")
	assert.NoError(s.T(), err)
	assert.False(s.T(), created2)
	assert.Equal(s.T(), mailbox1.ID, mailbox2.ID)
}

// ==================== Message CRUD Tests ====================

func (s *DatabaseIntegrationTestSuite) TestMessage_CRUD() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "message-crud.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@message-crud.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Create message
	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		SenderName:  "Test Sender",
		Subject:     "Test Subject",
		BodyText:    "Test body",
		IsRead:      false,
	}
	err = s.messageRepo.Create(ctx, message)
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), message.ID)

	// Get by ID
	retrieved, err := s.messageRepo.GetByID(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "Test Subject", retrieved.Subject)
	assert.False(s.T(), retrieved.IsRead)

	// Mark as read
	err = s.messageRepo.MarkAsRead(ctx, message.ID)
	assert.NoError(s.T(), err)

	// Verify read status
	retrieved, err = s.messageRepo.GetByID(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), retrieved.IsRead)

	// Delete
	err = s.messageRepo.Delete(ctx, message.ID)
	assert.NoError(s.T(), err)

	// Verify deletion
	_, err = s.messageRepo.GetByID(ctx, message.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)
}

func (s *DatabaseIntegrationTestSuite) TestMessage_WithAttachments() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "attachments.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@attachments.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Create message with attachments
	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "With Attachments",
	}
	attachments := []models.Attachment{
		{Filename: "doc1.pdf", ContentType: "application/pdf", FilePath: "/path/doc1.pdf", SizeBytes: 1024},
		{Filename: "image.png", ContentType: "image/png", FilePath: "/path/image.png", SizeBytes: 2048},
	}
	err = s.messageRepo.CreateWithAttachments(ctx, message, attachments)
	assert.NoError(s.T(), err)

	// Get message with attachments
	retrieved, err := s.messageRepo.GetByID(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), retrieved.Attachments, 2)
}

// ==================== Cascade Delete Tests ====================

func (s *DatabaseIntegrationTestSuite) TestCascadeDelete_DomainToMailbox() {
	ctx := context.Background()

	// Create domain
	domain := &models.Domain{Name: "cascade-domain.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	// Create mailboxes
	for i := 0; i < 3; i++ {
		mailbox := &models.Mailbox{
			LocalPart:   fmt.Sprintf("user%d", i),
			DomainID:    domain.ID,
			FullAddress: fmt.Sprintf("user%d@cascade-domain.com", i),
		}
		err = s.mailboxRepo.Create(ctx, mailbox)
		require.NoError(s.T(), err)
	}

	// Verify mailboxes exist
	mailboxes, _, err := s.mailboxRepo.ListByDomain(ctx, domain.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), mailboxes, 3)

	// Delete domain
	err = s.domainRepo.Delete(ctx, domain.ID)
	assert.NoError(s.T(), err)

	// Verify mailboxes are deleted
	mailboxes, _, err = s.mailboxRepo.ListByDomain(ctx, domain.ID, 10, 0)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), mailboxes)
}

func (s *DatabaseIntegrationTestSuite) TestCascadeDelete_MailboxToMessage() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "cascade-mailbox.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@cascade-mailbox.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Create messages
	for i := 0; i < 3; i++ {
		message := &models.Message{
			MailboxID:   mailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     fmt.Sprintf("Message %d", i),
		}
		err = s.messageRepo.Create(ctx, message)
		require.NoError(s.T(), err)
	}

	// Verify messages exist
	messages, total, err := s.messageRepo.ListByMailbox(ctx, mailbox.ID, 10, 0)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(3), total)
	require.Len(s.T(), messages, 3)

	// Delete mailbox
	err = s.mailboxRepo.Delete(ctx, mailbox.ID)
	assert.NoError(s.T(), err)

	// Verify messages are deleted
	messages, total, err = s.messageRepo.ListByMailbox(ctx, mailbox.ID, 10, 0)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), total)
	assert.Empty(s.T(), messages)
}

func (s *DatabaseIntegrationTestSuite) TestCascadeDelete_MessageToAttachment() {
	ctx := context.Background()

	// Create domain, mailbox, and message with attachments
	domain := &models.Domain{Name: "cascade-message.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@cascade-message.com",
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

	// Verify attachments exist
	atts, err := s.attachmentRepo.ListByMessage(ctx, message.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), atts, 1)

	// Delete message
	err = s.messageRepo.Delete(ctx, message.ID)
	assert.NoError(s.T(), err)

	// Verify attachments are deleted
	atts, err = s.attachmentRepo.ListByMessage(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), atts)
}

func (s *DatabaseIntegrationTestSuite) TestCascadeDelete_FullChain() {
	ctx := context.Background()

	// Create full chain: domain -> mailbox -> message -> attachment
	domain := &models.Domain{Name: "full-cascade.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@full-cascade.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: "sender@example.com",
		Subject:     "Full Chain Test",
	}
	attachments := []models.Attachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", FilePath: "/path/doc.pdf", SizeBytes: 1024},
	}
	err = s.messageRepo.CreateWithAttachments(ctx, message, attachments)
	require.NoError(s.T(), err)

	// Delete domain (should cascade to all)
	err = s.domainRepo.Delete(ctx, domain.ID)
	assert.NoError(s.T(), err)

	// Verify all are deleted
	_, err = s.domainRepo.GetByID(ctx, domain.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)

	_, err = s.mailboxRepo.GetByID(ctx, mailbox.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)

	_, err = s.messageRepo.GetByID(ctx, message.ID)
	assert.ErrorIs(s.T(), err, repository.ErrNotFound)

	atts, err := s.attachmentRepo.ListByMessage(ctx, message.ID)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), atts)
}

// ==================== Unread Count Tests ====================

func (s *DatabaseIntegrationTestSuite) TestMailbox_UnreadCount() {
	ctx := context.Background()

	// Create domain and mailbox
	domain := &models.Domain{Name: "unread-count.com", IsActive: true}
	err := s.domainRepo.Create(ctx, domain)
	require.NoError(s.T(), err)

	mailbox := &models.Mailbox{
		LocalPart:   "test",
		DomainID:    domain.ID,
		FullAddress: "test@unread-count.com",
	}
	err = s.mailboxRepo.Create(ctx, mailbox)
	require.NoError(s.T(), err)

	// Create messages (3 unread, 2 read)
	for i := 0; i < 5; i++ {
		message := &models.Message{
			MailboxID:   mailbox.ID,
			SenderEmail: "sender@example.com",
			Subject:     fmt.Sprintf("Message %d", i),
			IsRead:      i < 2, // First 2 are read
		}
		err = s.messageRepo.Create(ctx, message)
		require.NoError(s.T(), err)
	}

	// Check unread count
	count, err := s.messageRepo.CountUnread(ctx, mailbox.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), count)

	// Check via ListByDomain
	mailboxes, _, err := s.mailboxRepo.ListByDomain(ctx, domain.ID, 10, 0)
	assert.NoError(s.T(), err)
	require.Len(s.T(), mailboxes, 1)
	assert.Equal(s.T(), int64(3), mailboxes[0].UnreadCount)
}
