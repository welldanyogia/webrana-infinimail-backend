package fixtures

import (
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
)

// DomainBuilder creates test Domain instances with fluent API
type DomainBuilder struct {
	domain models.Domain
}

// NewDomainBuilder creates a new DomainBuilder with sensible defaults
func NewDomainBuilder() *DomainBuilder {
	now := time.Now()
	return &DomainBuilder{
		domain: models.Domain{
			ID:        1,
			Name:      "example.com",
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// WithID sets the domain ID
func (b *DomainBuilder) WithID(id uint) *DomainBuilder {
	b.domain.ID = id
	return b
}

// WithName sets the domain name
func (b *DomainBuilder) WithName(name string) *DomainBuilder {
	b.domain.Name = name
	return b
}

// WithActive sets the domain active status
func (b *DomainBuilder) WithActive(active bool) *DomainBuilder {
	b.domain.IsActive = active
	return b
}

// WithCreatedAt sets the created timestamp
func (b *DomainBuilder) WithCreatedAt(t time.Time) *DomainBuilder {
	b.domain.CreatedAt = t
	return b
}

// WithUpdatedAt sets the updated timestamp
func (b *DomainBuilder) WithUpdatedAt(t time.Time) *DomainBuilder {
	b.domain.UpdatedAt = t
	return b
}

// Build returns the constructed Domain
func (b *DomainBuilder) Build() *models.Domain {
	return &b.domain
}

// BuildValue returns the constructed Domain as a value (not pointer)
func (b *DomainBuilder) BuildValue() models.Domain {
	return b.domain
}

// MailboxBuilder creates test Mailbox instances with fluent API
type MailboxBuilder struct {
	mailbox models.Mailbox
}

// NewMailboxBuilder creates a new MailboxBuilder with sensible defaults
func NewMailboxBuilder() *MailboxBuilder {
	now := time.Now()
	return &MailboxBuilder{
		mailbox: models.Mailbox{
			ID:          1,
			LocalPart:   "user",
			DomainID:    1,
			FullAddress: "user@example.com",
			CreatedAt:   now,
		},
	}
}

// WithID sets the mailbox ID
func (b *MailboxBuilder) WithID(id uint) *MailboxBuilder {
	b.mailbox.ID = id
	return b
}

// WithLocalPart sets the local part of the email address
func (b *MailboxBuilder) WithLocalPart(localPart string) *MailboxBuilder {
	b.mailbox.LocalPart = localPart
	return b
}

// WithDomainID sets the domain ID
func (b *MailboxBuilder) WithDomainID(domainID uint) *MailboxBuilder {
	b.mailbox.DomainID = domainID
	return b
}

// WithFullAddress sets the full email address
func (b *MailboxBuilder) WithFullAddress(address string) *MailboxBuilder {
	b.mailbox.FullAddress = address
	return b
}

// WithCreatedAt sets the created timestamp
func (b *MailboxBuilder) WithCreatedAt(t time.Time) *MailboxBuilder {
	b.mailbox.CreatedAt = t
	return b
}

// WithLastAccessedAt sets the last accessed timestamp
func (b *MailboxBuilder) WithLastAccessedAt(t *time.Time) *MailboxBuilder {
	b.mailbox.LastAccessedAt = t
	return b
}

// WithDomain sets the associated domain
func (b *MailboxBuilder) WithDomain(domain models.Domain) *MailboxBuilder {
	b.mailbox.Domain = domain
	return b
}

// Build returns the constructed Mailbox
func (b *MailboxBuilder) Build() *models.Mailbox {
	return &b.mailbox
}

// BuildValue returns the constructed Mailbox as a value (not pointer)
func (b *MailboxBuilder) BuildValue() models.Mailbox {
	return b.mailbox
}


// MessageBuilder creates test Message instances with fluent API
type MessageBuilder struct {
	message models.Message
}

// NewMessageBuilder creates a new MessageBuilder with sensible defaults
func NewMessageBuilder() *MessageBuilder {
	now := time.Now()
	return &MessageBuilder{
		message: models.Message{
			ID:          1,
			MailboxID:   1,
			SenderEmail: "sender@external.com",
			SenderName:  "Test Sender",
			Subject:     "Test Subject",
			Snippet:     "This is a test email...",
			BodyText:    "This is a test email body.",
			BodyHTML:    "<p>This is a test email body.</p>",
			IsRead:      false,
			ReceivedAt:  now,
		},
	}
}

// WithID sets the message ID
func (b *MessageBuilder) WithID(id uint) *MessageBuilder {
	b.message.ID = id
	return b
}

// WithMailboxID sets the mailbox ID
func (b *MessageBuilder) WithMailboxID(mailboxID uint) *MessageBuilder {
	b.message.MailboxID = mailboxID
	return b
}

// WithSender sets the sender email and name
func (b *MessageBuilder) WithSender(email, name string) *MessageBuilder {
	b.message.SenderEmail = email
	b.message.SenderName = name
	return b
}

// WithSenderEmail sets only the sender email
func (b *MessageBuilder) WithSenderEmail(email string) *MessageBuilder {
	b.message.SenderEmail = email
	return b
}

// WithSenderName sets only the sender name
func (b *MessageBuilder) WithSenderName(name string) *MessageBuilder {
	b.message.SenderName = name
	return b
}

// WithSubject sets the message subject
func (b *MessageBuilder) WithSubject(subject string) *MessageBuilder {
	b.message.Subject = subject
	return b
}

// WithSnippet sets the message snippet
func (b *MessageBuilder) WithSnippet(snippet string) *MessageBuilder {
	b.message.Snippet = snippet
	return b
}

// WithBody sets both text and HTML body
func (b *MessageBuilder) WithBody(text, html string) *MessageBuilder {
	b.message.BodyText = text
	b.message.BodyHTML = html
	return b
}

// WithBodyText sets only the text body
func (b *MessageBuilder) WithBodyText(text string) *MessageBuilder {
	b.message.BodyText = text
	return b
}

// WithBodyHTML sets only the HTML body
func (b *MessageBuilder) WithBodyHTML(html string) *MessageBuilder {
	b.message.BodyHTML = html
	return b
}

// WithRead sets the read status
func (b *MessageBuilder) WithRead(isRead bool) *MessageBuilder {
	b.message.IsRead = isRead
	return b
}

// WithReceivedAt sets the received timestamp
func (b *MessageBuilder) WithReceivedAt(t time.Time) *MessageBuilder {
	b.message.ReceivedAt = t
	return b
}

// WithAttachments sets the message attachments
func (b *MessageBuilder) WithAttachments(attachments []models.Attachment) *MessageBuilder {
	b.message.Attachments = attachments
	return b
}

// WithMailbox sets the associated mailbox
func (b *MessageBuilder) WithMailbox(mailbox models.Mailbox) *MessageBuilder {
	b.message.Mailbox = mailbox
	return b
}

// Build returns the constructed Message
func (b *MessageBuilder) Build() *models.Message {
	return &b.message
}

// BuildValue returns the constructed Message as a value (not pointer)
func (b *MessageBuilder) BuildValue() models.Message {
	return b.message
}

// AttachmentBuilder creates test Attachment instances with fluent API
type AttachmentBuilder struct {
	attachment models.Attachment
}

// NewAttachmentBuilder creates a new AttachmentBuilder with sensible defaults
func NewAttachmentBuilder() *AttachmentBuilder {
	return &AttachmentBuilder{
		attachment: models.Attachment{
			ID:          1,
			MessageID:   1,
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			FilePath:    "/attachments/abc123.pdf",
			SizeBytes:   1024,
		},
	}
}

// WithID sets the attachment ID
func (b *AttachmentBuilder) WithID(id uint) *AttachmentBuilder {
	b.attachment.ID = id
	return b
}

// WithMessageID sets the message ID
func (b *AttachmentBuilder) WithMessageID(messageID uint) *AttachmentBuilder {
	b.attachment.MessageID = messageID
	return b
}

// WithFilename sets the attachment filename
func (b *AttachmentBuilder) WithFilename(filename string) *AttachmentBuilder {
	b.attachment.Filename = filename
	return b
}

// WithContentType sets the content type
func (b *AttachmentBuilder) WithContentType(contentType string) *AttachmentBuilder {
	b.attachment.ContentType = contentType
	return b
}

// WithFilePath sets the file path
func (b *AttachmentBuilder) WithFilePath(filePath string) *AttachmentBuilder {
	b.attachment.FilePath = filePath
	return b
}

// WithSize sets the file size in bytes
func (b *AttachmentBuilder) WithSize(size int64) *AttachmentBuilder {
	b.attachment.SizeBytes = size
	return b
}

// WithMessage sets the associated message
func (b *AttachmentBuilder) WithMessage(message models.Message) *AttachmentBuilder {
	b.attachment.Message = message
	return b
}

// Build returns the constructed Attachment
func (b *AttachmentBuilder) Build() *models.Attachment {
	return &b.attachment
}

// BuildValue returns the constructed Attachment as a value (not pointer)
func (b *AttachmentBuilder) BuildValue() models.Attachment {
	return b.attachment
}

// Helper functions for creating multiple test entities

// CreateDomains creates a slice of domains with sequential IDs
func CreateDomains(count int) []models.Domain {
	domains := make([]models.Domain, count)
	for i := 0; i < count; i++ {
		domains[i] = NewDomainBuilder().
			WithID(uint(i + 1)).
			WithName(generateDomainName(i)).
			BuildValue()
	}
	return domains
}

// CreateMailboxes creates a slice of mailboxes for a given domain
func CreateMailboxes(domainID uint, count int) []models.Mailbox {
	mailboxes := make([]models.Mailbox, count)
	for i := 0; i < count; i++ {
		localPart := generateLocalPart(i)
		mailboxes[i] = NewMailboxBuilder().
			WithID(uint(i + 1)).
			WithLocalPart(localPart).
			WithDomainID(domainID).
			WithFullAddress(localPart + "@example.com").
			BuildValue()
	}
	return mailboxes
}

// CreateMessages creates a slice of messages for a given mailbox
func CreateMessages(mailboxID uint, count int) []models.Message {
	messages := make([]models.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = NewMessageBuilder().
			WithID(uint(i + 1)).
			WithMailboxID(mailboxID).
			WithSubject(generateSubject(i)).
			WithReceivedAt(time.Now().Add(-time.Duration(i) * time.Hour)).
			BuildValue()
	}
	return messages
}

// Helper functions for generating test data
func generateDomainName(index int) string {
	names := []string{"example.com", "test.com", "mail.org", "inbox.net", "demo.io"}
	return names[index%len(names)]
}

func generateLocalPart(index int) string {
	parts := []string{"user", "admin", "info", "support", "contact"}
	if index < len(parts) {
		return parts[index]
	}
	return parts[index%len(parts)] + string(rune('0'+index/len(parts)))
}

func generateSubject(index int) string {
	subjects := []string{
		"Welcome to our service",
		"Your order confirmation",
		"Important update",
		"Newsletter",
		"Account notification",
	}
	return subjects[index%len(subjects)]
}
