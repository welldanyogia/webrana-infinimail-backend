package models

import (
	"time"
)

// Message represents an email message received by a mailbox
type Message struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MailboxID   uint      `gorm:"not null;index" json:"mailbox_id"`
	SenderEmail string    `gorm:"not null;size:255" json:"sender_email"`
	SenderName  string    `gorm:"size:255" json:"sender_name,omitempty"`
	Subject     string    `json:"subject,omitempty"`
	Snippet     string    `gorm:"size:255" json:"snippet,omitempty"`
	BodyText    string    `json:"body_text,omitempty"`
	BodyHTML    string    `json:"body_html,omitempty"`
	IsRead      bool      `gorm:"default:false" json:"is_read"`
	ReceivedAt  time.Time `gorm:"autoCreateTime" json:"received_at"`

	// Relationships
	Mailbox     Mailbox      `gorm:"foreignKey:MailboxID;constraint:OnDelete:CASCADE" json:"-"`
	Attachments []Attachment `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"attachments,omitempty"`
}

// TableName returns the table name for Message
func (Message) TableName() string {
	return "messages"
}

// MessageListItem is a lightweight version for list views
type MessageListItem struct {
	ID              uint      `json:"id"`
	MailboxID       uint      `json:"mailbox_id"`
	SenderEmail     string    `json:"sender_email"`
	SenderName      string    `json:"sender_name,omitempty"`
	Subject         string    `json:"subject,omitempty"`
	Snippet         string    `json:"snippet,omitempty"`
	IsRead          bool      `json:"is_read"`
	ReceivedAt      time.Time `json:"received_at"`
	AttachmentCount int       `json:"attachment_count"`
}
