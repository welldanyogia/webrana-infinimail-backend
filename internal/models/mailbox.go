package models

import (
	"time"
)

// Mailbox represents an email address within a domain
type Mailbox struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	LocalPart      string     `gorm:"not null;size:255" json:"local_part"`
	DomainID       uint       `gorm:"not null;index" json:"domain_id"`
	FullAddress    string     `gorm:"uniqueIndex;not null;size:255" json:"full_address"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`

	// Relationships
	Domain   Domain    `gorm:"foreignKey:DomainID;constraint:OnDelete:CASCADE" json:"-"`
	Messages []Message `gorm:"foreignKey:MailboxID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName returns the table name for Mailbox
func (Mailbox) TableName() string {
	return "mailboxes"
}

// MailboxWithUnreadCount is used for API responses that include unread count
type MailboxWithUnreadCount struct {
	Mailbox
	UnreadCount int64 `json:"unread_count"`
}
