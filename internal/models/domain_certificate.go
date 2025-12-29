package models

import (
	"time"
)

// DomainCertificate represents an SSL certificate for a domain
type DomainCertificate struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DomainID   uint      `gorm:"uniqueIndex;not null" json:"domain_id"`
	DomainName string    `gorm:"not null;size:255" json:"domain_name"`
	CertPath   string    `gorm:"not null;size:500" json:"cert_path"`
	KeyPath    string    `gorm:"not null;size:500" json:"key_path"`
	ExpiresAt  time.Time `gorm:"not null" json:"expires_at"`
	IssuedAt   time.Time `gorm:"not null" json:"issued_at"`
	AutoRenew  bool      `gorm:"default:true" json:"auto_renew"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Domain *Domain `gorm:"foreignKey:DomainID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName returns the table name for DomainCertificate
func (DomainCertificate) TableName() string {
	return "domain_certificates"
}

// IsExpired checks if the certificate has expired
func (c *DomainCertificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsExpiringSoon checks if the certificate is expiring within the given number of days
func (c *DomainCertificate) IsExpiringSoon(days int) bool {
	expiryThreshold := time.Now().AddDate(0, 0, days)
	return c.ExpiresAt.Before(expiryThreshold)
}

// DaysUntilExpiry returns the number of days until the certificate expires
func (c *DomainCertificate) DaysUntilExpiry() int {
	duration := time.Until(c.ExpiresAt)
	return int(duration.Hours() / 24)
}
