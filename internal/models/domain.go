package models

import (
	"time"
)

// Domain represents an email domain that the system handles
type Domain struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null;size:255" json:"name"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Mailboxes []Mailbox `gorm:"foreignKey:DomainID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName returns the table name for Domain
func (Domain) TableName() string {
	return "domains"
}
