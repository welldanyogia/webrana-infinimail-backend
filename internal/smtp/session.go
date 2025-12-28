package smtp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/websocket"
)

// Session implements the go-smtp Session interface
type Session struct {
	backend    *Backend
	from       string
	recipients []string
}

// NewSession creates a new SMTP session
func NewSession(backend *Backend) *Session {
	return &Session{
		backend:    backend,
		recipients: make([]string, 0),
	}
}

// AuthPlain handles PLAIN authentication (not required for receiving)
func (s *Session) AuthPlain(username, password string) error {
	// No authentication required for receiving emails
	return nil
}

// Mail handles the MAIL FROM command
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	if s.backend.logger != nil {
		s.backend.logger.Debug("MAIL FROM", slog.String("from", from))
	}
	return nil
}

// Rcpt handles the RCPT TO command
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// Parse recipient address
	localPart, domainName, err := parseEmailAddress(to)
	if err != nil {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "Invalid recipient address",
		}
	}

	// Check if domain exists and is active
	ctx := context.Background()
	domain, err := s.backend.domainRepo.GetByName(ctx, domainName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &smtp.SMTPError{
				Code:         550,
				EnhancedCode: smtp.EnhancedCode{5, 1, 1},
				Message:      "Domain not found",
			}
		}
		return &smtp.SMTPError{
			Code:         451,
			EnhancedCode: smtp.EnhancedCode{4, 3, 0},
			Message:      "Temporary error",
		}
	}

	if !domain.IsActive {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "Domain is not active",
		}
	}

	// If auto-provisioning is disabled, check if mailbox exists
	if !s.backend.autoProvision {
		_, err := s.backend.mailboxRepo.GetByAddress(ctx, to)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return &smtp.SMTPError{
					Code:         550,
					EnhancedCode: smtp.EnhancedCode{5, 1, 1},
					Message:      "Mailbox not found",
				}
			}
			return &smtp.SMTPError{
				Code:         451,
				EnhancedCode: smtp.EnhancedCode{4, 3, 0},
				Message:      "Temporary error",
			}
		}
	}

	s.recipients = append(s.recipients, to)
	if s.backend.logger != nil {
		s.backend.logger.Debug("RCPT TO", slog.String("to", to), slog.String("local_part", localPart))
	}
	return nil
}

// Data handles the DATA command - receives the email content
func (s *Session) Data(r io.Reader) error {
	if len(s.recipients) == 0 {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "No recipients specified",
		}
	}

	// Parse the email
	parsedEmail, err := ParseEmail(r)
	if err != nil {
		if s.backend.logger != nil {
			s.backend.logger.Error("failed to parse email", slog.Any("error", err))
		}
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 6, 0},
			Message:      "Failed to parse email",
		}
	}

	// Override sender from envelope if not in headers
	if parsedEmail.SenderEmail == "" {
		parsedEmail.SenderEmail = s.from
	}

	ctx := context.Background()

	// Process for each recipient
	for _, recipient := range s.recipients {
		if err := s.processEmail(ctx, recipient, parsedEmail); err != nil {
			if s.backend.logger != nil {
				s.backend.logger.Error("failed to process email",
					slog.String("recipient", recipient),
					slog.Any("error", err))
			}
			// Continue processing other recipients
		}
	}

	if s.backend.logger != nil {
		s.backend.logger.Info("email received",
			slog.String("from", s.from),
			slog.Int("recipients", len(s.recipients)),
			slog.String("subject", parsedEmail.Subject))
	}

	return nil
}

// processEmail stores the email for a single recipient
func (s *Session) processEmail(ctx context.Context, recipient string, email *ParsedEmail) error {
	localPart, domainName, err := parseEmailAddress(recipient)
	if err != nil {
		return err
	}

	// Get domain
	domain, err := s.backend.domainRepo.GetByName(ctx, domainName)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Get or create mailbox
	mailbox, created, err := s.backend.mailboxRepo.GetOrCreate(ctx, localPart, domain.ID, domain.Name)
	if err != nil {
		return fmt.Errorf("failed to get/create mailbox: %w", err)
	}

	if created && s.backend.logger != nil {
		s.backend.logger.Info("auto-provisioned mailbox", slog.String("address", mailbox.FullAddress))
	}

	// Create message
	message := &models.Message{
		MailboxID:   mailbox.ID,
		SenderEmail: email.SenderEmail,
		SenderName:  email.SenderName,
		Subject:     email.Subject,
		Snippet:     email.Snippet,
		BodyText:    email.BodyText,
		BodyHTML:    email.BodyHTML,
		IsRead:      false,
	}

	// Store attachments
	var attachments []models.Attachment
	for _, att := range email.Attachments {
		// Save file to storage
		filePath, err := s.backend.fileStorage.Save(att.Filename, att.Content)
		if err != nil {
			if s.backend.logger != nil {
				s.backend.logger.Error("failed to save attachment",
					slog.String("filename", att.Filename),
					slog.Any("error", err))
			}
			continue
		}

		attachments = append(attachments, models.Attachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			FilePath:    filePath,
			SizeBytes:   att.Size,
		})
	}

	// Create message with attachments
	if err := s.backend.messageRepo.CreateWithAttachments(ctx, message, attachments); err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Notify WebSocket subscribers
	if s.backend.wsHub != nil {
		s.backend.wsHub.BroadcastNewMessage(mailbox.ID, &websocket.NewMessagePayload{
			ID:          message.ID,
			SenderEmail: message.SenderEmail,
			SenderName:  message.SenderName,
			Subject:     message.Subject,
			ReceivedAt:  message.ReceivedAt.Format(time.RFC3339),
		})
	}

	return nil
}

// Reset resets the session state
func (s *Session) Reset() {
	s.from = ""
	s.recipients = make([]string, 0)
}

// Logout handles the end of the session
func (s *Session) Logout() error {
	return nil
}

// parseEmailAddress parses an email address into local part and domain
func parseEmailAddress(address string) (localPart, domain string, err error) {
	// Remove angle brackets if present
	address = strings.TrimPrefix(address, "<")
	address = strings.TrimSuffix(address, ">")
	address = strings.TrimSpace(address)

	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email address: %s", address)
	}

	localPart = strings.ToLower(parts[0])
	domain = strings.ToLower(parts[1])

	if localPart == "" || domain == "" {
		return "", "", fmt.Errorf("invalid email address: %s", address)
	}

	return localPart, domain, nil
}
