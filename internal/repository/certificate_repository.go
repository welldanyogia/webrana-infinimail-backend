package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/gorm"
)

// CertificateRepository defines the interface for certificate data access
type CertificateRepository interface {
	// Create creates a new certificate record
	Create(ctx context.Context, cert *models.DomainCertificate) error

	// GetByID retrieves a certificate by its ID
	GetByID(ctx context.Context, id uint) (*models.DomainCertificate, error)

	// GetByDomainID retrieves a certificate by domain ID
	GetByDomainID(ctx context.Context, domainID uint) (*models.DomainCertificate, error)

	// GetByDomainName retrieves a certificate by domain name
	GetByDomainName(ctx context.Context, domainName string) (*models.DomainCertificate, error)

	// Update updates an existing certificate record
	Update(ctx context.Context, cert *models.DomainCertificate) error

	// Delete deletes a certificate by its ID
	Delete(ctx context.Context, id uint) error

	// DeleteByDomainID deletes a certificate by domain ID
	DeleteByDomainID(ctx context.Context, domainID uint) error

	// GetExpiringCertificates returns certificates expiring within the given number of days
	GetExpiringCertificates(ctx context.Context, days int) ([]models.DomainCertificate, error)

	// GetAllWithAutoRenew returns all certificates with auto-renew enabled
	GetAllWithAutoRenew(ctx context.Context) ([]models.DomainCertificate, error)

	// List returns all certificates
	List(ctx context.Context) ([]models.DomainCertificate, error)
}

// certificateRepository implements CertificateRepository using GORM
type certificateRepository struct {
	db *gorm.DB
}

// NewCertificateRepository creates a new CertificateRepository instance
func NewCertificateRepository(db *gorm.DB) CertificateRepository {
	return &certificateRepository{db: db}
}


// Create creates a new certificate record
func (r *certificateRepository) Create(ctx context.Context, cert *models.DomainCertificate) error {
	result := r.db.WithContext(ctx).Create(cert)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return fmt.Errorf("certificate for domain ID %d already exists: %w", cert.DomainID, ErrDuplicateEntry)
		}
		return fmt.Errorf("failed to create certificate: %w", result.Error)
	}
	return nil
}

// GetByID retrieves a certificate by its ID
func (r *certificateRepository) GetByID(ctx context.Context, id uint) (*models.DomainCertificate, error) {
	var cert models.DomainCertificate
	result := r.db.WithContext(ctx).First(&cert, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get certificate by ID: %w", result.Error)
	}
	return &cert, nil
}

// GetByDomainID retrieves a certificate by domain ID
func (r *certificateRepository) GetByDomainID(ctx context.Context, domainID uint) (*models.DomainCertificate, error) {
	var cert models.DomainCertificate
	result := r.db.WithContext(ctx).Where("domain_id = ?", domainID).First(&cert)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get certificate by domain ID: %w", result.Error)
	}
	return &cert, nil
}

// GetByDomainName retrieves a certificate by domain name
func (r *certificateRepository) GetByDomainName(ctx context.Context, domainName string) (*models.DomainCertificate, error) {
	var cert models.DomainCertificate
	result := r.db.WithContext(ctx).Where("domain_name = ?", domainName).First(&cert)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get certificate by domain name: %w", result.Error)
	}
	return &cert, nil
}

// Update updates an existing certificate record
func (r *certificateRepository) Update(ctx context.Context, cert *models.DomainCertificate) error {
	result := r.db.WithContext(ctx).Save(cert)
	if result.Error != nil {
		return fmt.Errorf("failed to update certificate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a certificate by its ID
func (r *certificateRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.DomainCertificate{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete certificate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteByDomainID deletes a certificate by domain ID
func (r *certificateRepository) DeleteByDomainID(ctx context.Context, domainID uint) error {
	result := r.db.WithContext(ctx).Where("domain_id = ?", domainID).Delete(&models.DomainCertificate{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete certificate by domain ID: %w", result.Error)
	}
	return nil
}

// GetExpiringCertificates returns certificates expiring within the given number of days
func (r *certificateRepository) GetExpiringCertificates(ctx context.Context, days int) ([]models.DomainCertificate, error) {
	var certs []models.DomainCertificate
	expiryThreshold := time.Now().AddDate(0, 0, days)

	result := r.db.WithContext(ctx).
		Where("expires_at <= ? AND expires_at > ?", expiryThreshold, time.Now()).
		Order("expires_at ASC").
		Find(&certs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get expiring certificates: %w", result.Error)
	}
	return certs, nil
}

// GetAllWithAutoRenew returns all certificates with auto-renew enabled
func (r *certificateRepository) GetAllWithAutoRenew(ctx context.Context) ([]models.DomainCertificate, error) {
	var certs []models.DomainCertificate
	result := r.db.WithContext(ctx).
		Where("auto_renew = ?", true).
		Order("expires_at ASC").
		Find(&certs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get auto-renew certificates: %w", result.Error)
	}
	return certs, nil
}

// List returns all certificates
func (r *certificateRepository) List(ctx context.Context) ([]models.DomainCertificate, error) {
	var certs []models.DomainCertificate
	result := r.db.WithContext(ctx).Order("domain_name ASC").Find(&certs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", result.Error)
	}
	return certs, nil
}
