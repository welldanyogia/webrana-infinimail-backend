package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// CertificateManagerConfig holds configuration for the certificate manager service
type CertificateManagerConfig struct {
	// RenewalDays is the number of days before expiry to trigger renewal
	RenewalDays int
	// CertStoragePath is the base path for certificate storage
	CertStoragePath string
}

// Certificate represents a certificate with its metadata
type Certificate struct {
	ID         uint      `json:"id"`
	DomainID   uint      `json:"domain_id"`
	DomainName string    `json:"domain_name"`
	CertPath   string    `json:"cert_path"`
	KeyPath    string    `json:"key_path"`
	ExpiresAt  time.Time `json:"expires_at"`
	IssuedAt   time.Time `json:"issued_at"`
	AutoRenew  bool      `json:"auto_renew"`
}

// CertificateManagerService defines the interface for certificate lifecycle management
type CertificateManagerService interface {
	// GenerateCertificate requests and stores a certificate for a domain using ACME
	GenerateCertificate(ctx context.Context, domain *models.Domain) (*Certificate, error)

	// GetCertificate retrieves certificate metadata for a domain
	GetCertificate(ctx context.Context, domainName string) (*Certificate, error)

	// GetCertificateByDomainID retrieves certificate metadata by domain ID
	GetCertificateByDomainID(ctx context.Context, domainID uint) (*Certificate, error)

	// RenewCertificate renews an existing certificate for a domain
	RenewCertificate(ctx context.Context, domainID uint) (*Certificate, error)

	// GetExpiringCertificates returns certificates expiring within the configured days
	GetExpiringCertificates(ctx context.Context, days int) ([]Certificate, error)

	// DeleteCertificate removes a certificate from storage and database
	DeleteCertificate(ctx context.Context, domainID uint) error

	// SetAutoRenew enables or disables auto-renewal for a certificate
	SetAutoRenew(ctx context.Context, domainID uint, autoRenew bool) error

	// SetCertificateStore sets the certificate store for hot reload notifications
	SetCertificateStore(certStore CertificateStore)
}


// certificateManagerService implements CertificateManagerService
type certificateManagerService struct {
	acmeClient    ACMEClient
	certStorage   CertStorage
	certRepo      repository.CertificateRepository
	domainRepo    repository.DomainRepository
	domainManager DomainManagerService
	certStore     CertificateStore // For hot reload notifications
	config        CertificateManagerConfig
}

// NewCertificateManagerService creates a new CertificateManagerService instance
func NewCertificateManagerService(
	acmeClient ACMEClient,
	certStorage CertStorage,
	certRepo repository.CertificateRepository,
	domainRepo repository.DomainRepository,
	domainManager DomainManagerService,
	config CertificateManagerConfig,
) CertificateManagerService {
	// Set default renewal days if not configured
	if config.RenewalDays <= 0 {
		config.RenewalDays = 30
	}

	return &certificateManagerService{
		acmeClient:    acmeClient,
		certStorage:   certStorage,
		certRepo:      certRepo,
		domainRepo:    domainRepo,
		domainManager: domainManager,
		config:        config,
	}
}

// GenerateCertificate requests and stores a certificate for a domain using ACME
// This is a two-step process:
// 1. First call: Returns ACME challenge info (user needs to add _acme-challenge TXT record)
// 2. Second call: Completes the challenge and generates the certificate
func (s *certificateManagerService) GenerateCertificate(ctx context.Context, domain *models.Domain) (*Certificate, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain cannot be nil")
	}

	// Verify domain is in dns_verified status
	if domain.Status != models.StatusDNSVerified {
		return nil, fmt.Errorf("domain must be in dns_verified status to generate certificate, current status: %s", domain.Status)
	}

	// Update domain status to pending_certificate
	if err := s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusPendingCertificate, ""); err != nil {
		return nil, fmt.Errorf("failed to update domain status to pending_certificate: %w", err)
	}

	// Prepare domains for certificate (main domain + mail subdomain for SAN)
	domains := []string{
		domain.Name,
		fmt.Sprintf("mail.%s", domain.Name),
	}

	// Get DNS challenge for the primary domain
	challengeInfo, err := s.acmeClient.GetDNSChallenge(ctx, domain.Name)
	if err != nil {
		// Update status to failed
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, fmt.Sprintf("ACME challenge failed: %v", err))
		return nil, fmt.Errorf("failed to get DNS challenge: %w", err)
	}

	// Store the ACME challenge info in domain error message temporarily
	// This allows the user to see what TXT record they need to add
	acmeChallengeMsg := fmt.Sprintf("ACME_CHALLENGE:_acme-challenge.%s=%s", domain.Name, challengeInfo.TXTRecord)
	if err := s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusPendingCertificate, acmeChallengeMsg); err != nil {
		return nil, fmt.Errorf("failed to store ACME challenge info: %w", err)
	}

	// Complete DNS challenge (Let's Encrypt will verify the _acme-challenge TXT record)
	if err := s.acmeClient.CompleteDNSChallenge(ctx, domain.Name); err != nil {
		// Update status to failed with helpful message
		errMsg := fmt.Sprintf("ACME DNS challenge failed. Please add TXT record: Name=_acme-challenge.%s Value=%s", domain.Name, challengeInfo.TXTRecord)
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, errMsg)
		return nil, fmt.Errorf("failed to complete DNS challenge: %w", err)
	}

	// Request certificate from ACME
	bundle, err := s.acmeClient.RequestCertificate(ctx, domains)
	if err != nil {
		// Update status to failed
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, fmt.Sprintf("Certificate request failed: %v", err))
		return nil, fmt.Errorf("failed to request certificate: %w", err)
	}

	// Save certificate to disk
	storedCert, err := s.certStorage.SaveCertificate(domain.Name, bundle)
	if err != nil {
		// Update status to failed
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, fmt.Sprintf("Certificate storage failed: %v", err))
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// Check if certificate already exists in database
	existingCert, err := s.certRepo.GetByDomainID(ctx, domain.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing certificate: %w", err)
	}

	// Create or update certificate record in database
	var dbCert *models.DomainCertificate
	if existingCert != nil {
		// Update existing certificate
		existingCert.CertPath = storedCert.CertPath
		existingCert.KeyPath = storedCert.KeyPath
		existingCert.ExpiresAt = storedCert.ExpiresAt
		existingCert.IssuedAt = storedCert.IssuedAt
		if err := s.certRepo.Update(ctx, existingCert); err != nil {
			return nil, fmt.Errorf("failed to update certificate record: %w", err)
		}
		dbCert = existingCert
	} else {
		// Create new certificate record
		dbCert = &models.DomainCertificate{
			DomainID:   domain.ID,
			DomainName: domain.Name,
			CertPath:   storedCert.CertPath,
			KeyPath:    storedCert.KeyPath,
			ExpiresAt:  storedCert.ExpiresAt,
			IssuedAt:   storedCert.IssuedAt,
			AutoRenew:  true,
		}
		if err := s.certRepo.Create(ctx, dbCert); err != nil {
			return nil, fmt.Errorf("failed to create certificate record: %w", err)
		}
	}

	// Update domain status to certificate_issued
	if err := s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusCertificateIssued, ""); err != nil {
		return nil, fmt.Errorf("failed to update domain status to certificate_issued: %w", err)
	}

	// Trigger hot reload to make the new certificate available immediately
	s.triggerHotReload(domain.Name)

	return modelToCertificate(dbCert), nil
}


// GetCertificate retrieves certificate metadata for a domain by name
func (s *certificateManagerService) GetCertificate(ctx context.Context, domainName string) (*Certificate, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	cert, err := s.certRepo.GetByDomainName(ctx, domainName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("certificate not found for domain: %s", domainName)
		}
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	return modelToCertificate(cert), nil
}

// GetCertificateByDomainID retrieves certificate metadata by domain ID
func (s *certificateManagerService) GetCertificateByDomainID(ctx context.Context, domainID uint) (*Certificate, error) {
	cert, err := s.certRepo.GetByDomainID(ctx, domainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("certificate not found for domain ID: %d", domainID)
		}
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	return modelToCertificate(cert), nil
}

// RenewCertificate renews an existing certificate for a domain
func (s *certificateManagerService) RenewCertificate(ctx context.Context, domainID uint) (*Certificate, error) {
	// Get existing certificate
	existingCert, err := s.certRepo.GetByDomainID(ctx, domainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("no certificate found for domain ID: %d", domainID)
		}
		return nil, fmt.Errorf("failed to get existing certificate: %w", err)
	}

	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	// Prepare domains for certificate (main domain + mail subdomain for SAN)
	domains := []string{
		domain.Name,
		fmt.Sprintf("mail.%s", domain.Name),
	}

	// Get DNS challenge for the primary domain
	_, err = s.acmeClient.GetDNSChallenge(ctx, domain.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS challenge for renewal: %w", err)
	}

	// Complete DNS challenge
	if err := s.acmeClient.CompleteDNSChallenge(ctx, domain.Name); err != nil {
		return nil, fmt.Errorf("failed to complete DNS challenge for renewal: %w", err)
	}

	// Request new certificate from ACME
	bundle, err := s.acmeClient.RequestCertificate(ctx, domains)
	if err != nil {
		return nil, fmt.Errorf("failed to request renewal certificate: %w", err)
	}

	// Save new certificate to disk (overwrites existing)
	storedCert, err := s.certStorage.SaveCertificate(domain.Name, bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to save renewal certificate: %w", err)
	}

	// Update certificate record in database
	existingCert.CertPath = storedCert.CertPath
	existingCert.KeyPath = storedCert.KeyPath
	existingCert.ExpiresAt = storedCert.ExpiresAt
	existingCert.IssuedAt = storedCert.IssuedAt

	if err := s.certRepo.Update(ctx, existingCert); err != nil {
		return nil, fmt.Errorf("failed to update certificate record: %w", err)
	}

	// Trigger hot reload to make the renewed certificate available immediately
	s.triggerHotReload(domain.Name)

	return modelToCertificate(existingCert), nil
}

// GetExpiringCertificates returns certificates expiring within the given number of days
func (s *certificateManagerService) GetExpiringCertificates(ctx context.Context, days int) ([]Certificate, error) {
	if days <= 0 {
		days = s.config.RenewalDays
	}

	certs, err := s.certRepo.GetExpiringCertificates(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring certificates: %w", err)
	}

	result := make([]Certificate, len(certs))
	for i, cert := range certs {
		result[i] = *modelToCertificate(&cert)
	}

	return result, nil
}

// DeleteCertificate removes a certificate from storage and database
func (s *certificateManagerService) DeleteCertificate(ctx context.Context, domainID uint) error {
	// Get certificate to find domain name
	cert, err := s.certRepo.GetByDomainID(ctx, domainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Delete from disk
	if err := s.certStorage.DeleteCertificate(cert.DomainName); err != nil {
		return fmt.Errorf("failed to delete certificate from storage: %w", err)
	}

	// Delete from database
	if err := s.certRepo.DeleteByDomainID(ctx, domainID); err != nil {
		return fmt.Errorf("failed to delete certificate from database: %w", err)
	}

	return nil
}

// SetAutoRenew enables or disables auto-renewal for a certificate
func (s *certificateManagerService) SetAutoRenew(ctx context.Context, domainID uint, autoRenew bool) error {
	cert, err := s.certRepo.GetByDomainID(ctx, domainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("certificate not found for domain ID: %d", domainID)
		}
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	cert.AutoRenew = autoRenew
	if err := s.certRepo.Update(ctx, cert); err != nil {
		return fmt.Errorf("failed to update auto-renew setting: %w", err)
	}

	return nil
}

// SetCertificateStore sets the certificate store for hot reload notifications
func (s *certificateManagerService) SetCertificateStore(certStore CertificateStore) {
	s.certStore = certStore
}

// triggerHotReload triggers a hot reload of the certificate in the certificate store
func (s *certificateManagerService) triggerHotReload(domainName string) {
	if s.certStore == nil {
		return
	}
	// Load and add the new certificate to the store
	// This will also notify any registered callbacks (e.g., SMTP server)
	_ = s.certStore.LoadAndAddCertificate(domainName)
}

// modelToCertificate converts a DomainCertificate model to a Certificate
func modelToCertificate(model *models.DomainCertificate) *Certificate {
	if model == nil {
		return nil
	}
	return &Certificate{
		ID:         model.ID,
		DomainID:   model.DomainID,
		DomainName: model.DomainName,
		CertPath:   model.CertPath,
		KeyPath:    model.KeyPath,
		ExpiresAt:  model.ExpiresAt,
		IssuedAt:   model.IssuedAt,
		AutoRenew:  model.AutoRenew,
	}
}
