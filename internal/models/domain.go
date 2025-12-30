package models

import (
	"time"
)

// DomainStatus represents the domain setup lifecycle status
type DomainStatus string

const (
	// StatusPendingDNS indicates domain is waiting for DNS configuration
	StatusPendingDNS DomainStatus = "pending_dns"
	// StatusDNSVerified indicates DNS records have been verified
	StatusDNSVerified DomainStatus = "dns_verified"
	// StatusPendingACMEChallenge indicates ACME challenge requested, waiting for user to add DNS record
	StatusPendingACMEChallenge DomainStatus = "pending_acme_challenge"
	// StatusACMEChallengeReady indicates DNS verified locally, ready to submit to Let's Encrypt
	StatusACMEChallengeReady DomainStatus = "acme_challenge_ready"
	// StatusPendingCertificate indicates certificate generation is in progress
	StatusPendingCertificate DomainStatus = "pending_certificate"
	// StatusCertificateIssued indicates certificate has been issued
	StatusCertificateIssued DomainStatus = "certificate_issued"
	// StatusActive indicates domain is fully active and ready to receive email
	StatusActive DomainStatus = "active"
	// StatusFailed indicates domain setup has failed
	StatusFailed DomainStatus = "failed"
)

// Domain represents an email domain that the system handles
type Domain struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	Name         string       `gorm:"uniqueIndex;not null;size:255" json:"name"`
	IsActive     bool         `gorm:"default:false" json:"is_active"`
	Status       DomainStatus `gorm:"type:varchar(50);default:'pending_dns'" json:"status"`
	DNSChallenge string       `gorm:"size:255" json:"dns_challenge,omitempty"`
	ErrorMessage string       `gorm:"size:1000" json:"error_message,omitempty"`

	// ACME Challenge fields for Manual DNS Verification flow
	ACMEChallengeToken     string     `gorm:"size:255" json:"acme_challenge_token,omitempty"`
	ACMEChallengeValue     string     `gorm:"size:255" json:"acme_challenge_value,omitempty"`
	ACMEChallengeExpiresAt *time.Time `json:"acme_challenge_expires_at,omitempty"`
	ACMEDNSVerified        bool       `gorm:"default:false" json:"acme_dns_verified"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Mailboxes   []Mailbox          `gorm:"foreignKey:DomainID;constraint:OnDelete:CASCADE" json:"-"`
	Certificate *DomainCertificate `gorm:"foreignKey:DomainID" json:"certificate,omitempty"`
}

// TableName returns the table name for Domain
func (Domain) TableName() string {
	return "domains"
}

// IsValidStatus checks if the given status is a valid DomainStatus
func (s DomainStatus) IsValid() bool {
	switch s {
	case StatusPendingDNS, StatusDNSVerified, StatusPendingACMEChallenge,
		StatusACMEChallengeReady, StatusPendingCertificate,
		StatusCertificateIssued, StatusActive, StatusFailed:
		return true
	}
	return false
}

// String returns the string representation of DomainStatus
func (s DomainStatus) String() string {
	return string(s)
}
