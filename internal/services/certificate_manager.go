package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
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

// ACMEChallengeInfo contains information for the ACME DNS challenge (Manual DNS Verification)
type ACMEChallengeInfo struct {
	Domain          string    `json:"domain"`
	TXTRecordName   string    `json:"txt_record_name"`   // _acme-challenge.{domain}
	TXTRecordValue  string    `json:"txt_record_value"`  // The value to add
	ExpiresAt       time.Time `json:"expires_at"`        // Challenge expiration
	PropagationNote string    `json:"propagation_note"`  // Warning about DNS propagation
}

// ACMEDNSVerificationResult contains the result of DNS precheck
type ACMEDNSVerificationResult struct {
	Verified      bool     `json:"verified"`
	ExpectedValue string   `json:"expected_value"`
	FoundValues   []string `json:"found_values"`
	Message       string   `json:"message"`
	CanProceed    bool     `json:"can_proceed"`
}

// ACMEStatus represents the current state of ACME challenge
type ACMEStatus struct {
	Step            string             `json:"step"`                       // challenge_pending, dns_pending, dns_verified, validating, completed, failed
	ChallengeInfo   *ACMEChallengeInfo `json:"challenge_info,omitempty"`
	DNSVerified     bool               `json:"dns_verified"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	SuggestedAction string             `json:"suggested_action,omitempty"`
	LastUpdated     time.Time          `json:"last_updated"`
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

	// === Manual DNS Verification Methods ===

	// RequestACMEChallenge requests a new ACME challenge and stores it
	// Returns challenge info for user to add DNS record
	RequestACMEChallenge(ctx context.Context, domainID uint) (*ACMEChallengeInfo, error)

	// VerifyACMEDNS checks if the DNS TXT record is correctly configured
	// Does NOT submit to Let's Encrypt - just local DNS check
	VerifyACMEDNS(ctx context.Context, domainID uint) (*ACMEDNSVerificationResult, error)

	// SubmitACMEChallenge submits the challenge to Let's Encrypt for validation
	// Should only be called after VerifyACMEDNS returns success
	SubmitACMEChallenge(ctx context.Context, domainID uint) error

	// GetACMEStatus returns the current ACME challenge status
	GetACMEStatus(ctx context.Context, domainID uint) (*ACMEStatus, error)

	// ClearACMEChallenge clears stored challenge data
	ClearACMEChallenge(ctx context.Context, domainID uint) error
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
// This method uses the Manual DNS Verification flow:
// - It requires the domain to have acme_dns_verified = true (set by VerifyACMEDNS)
// - It uses stored challenge data instead of requesting a new challenge
// - It does NOT automatically proceed - user must have verified DNS first
//
// For the new manual flow, use:
// 1. RequestACMEChallenge() - Get challenge info
// 2. VerifyACMEDNS() - Verify DNS record is set
// 3. SubmitACMEChallenge() - Submit to Let's Encrypt and generate certificate
//
// This method is kept for backward compatibility but now requires manual DNS verification.
func (s *certificateManagerService) GenerateCertificate(ctx context.Context, domain *models.Domain) (*Certificate, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain cannot be nil")
	}

	// Get ACME logger
	acmeLogger := GetACMELogger()
	acmeLogger.StartDomainLog(domain.Name)

	// For the new manual flow, domain must be in acme_challenge_ready status
	// This means DNS has been verified by the user
	if domain.Status != models.StatusACMEChallengeReady {
		// Check if domain is in dns_verified status (legacy flow attempt)
		if domain.Status == models.StatusDNSVerified {
			err := fmt.Errorf("please use the manual DNS verification flow: 1) Request ACME challenge, 2) Add DNS TXT record, 3) Verify DNS, 4) Submit challenge")
			acmeLogger.MarkFailed(domain.Name, err)
			return nil, err
		}
		err := fmt.Errorf("domain must be in acme_challenge_ready status (DNS verified) to generate certificate, current status: %s", domain.Status)
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, err
	}

	// Verify DNS has been verified (acme_dns_verified must be true)
	if !domain.ACMEDNSVerified {
		err := fmt.Errorf("DNS has not been verified. Please verify DNS first using VerifyACMEDNS before generating certificate")
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, err
	}

	// Verify we have stored challenge data
	if domain.ACMEChallengeToken == "" || domain.ACMEChallengeValue == "" {
		err := fmt.Errorf("no stored ACME challenge data found. Please request a new challenge using RequestACMEChallenge")
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, err
	}

	// Check if challenge has expired
	if domain.ACMEChallengeExpiresAt != nil && time.Now().After(*domain.ACMEChallengeExpiresAt) {
		err := fmt.Errorf("ACME challenge has expired. Please request a new challenge using RequestACMEChallenge")
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, err
	}

	log.Printf("[CertManager] Starting certificate generation for domain: %s (using stored challenge data)", domain.Name)
	acmeLogger.LogInfo(domain.Name, "init", "Starting certificate generation using stored challenge data (manual DNS verification flow)")

	// Update domain status to pending_certificate
	if err := s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusPendingCertificate, ""); err != nil {
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, fmt.Errorf("failed to update domain status to pending_certificate: %w", err)
	}

	// Prepare domains for certificate (main domain + mail subdomain for SAN)
	domains := []string{
		domain.Name,
		fmt.Sprintf("mail.%s", domain.Name),
	}
	log.Printf("[CertManager] Requesting certificate for domains: %v", domains)
	acmeLogger.LogInfo(domain.Name, "domains", fmt.Sprintf("Requesting certificate for domains: %v", domains))

	// Complete DNS challenge with Let's Encrypt (submit only, no automatic precheck/wait)
	// The DNS has already been verified by VerifyACMEDNS
	log.Printf("[CertManager] Submitting DNS challenge to Let's Encrypt...")
	acmeLogger.LogInfo(domain.Name, "complete_challenge", "Submitting DNS challenge to Let's Encrypt (DNS already verified)")
	if err := s.acmeClient.CompleteDNSChallenge(ctx, domain.Name); err != nil {
		// Update status to failed with helpful message
		errMsg := fmt.Sprintf("ACME DNS challenge failed: %v. Please verify the TXT record is still correctly configured and try again.", err)
		log.Printf("[CertManager] ERROR: %s", errMsg)
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, errMsg)
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, fmt.Errorf("failed to complete DNS challenge: %w", err)
	}

	log.Printf("[CertManager] DNS challenge completed successfully!")
	acmeLogger.LogInfo(domain.Name, "challenge_complete", "DNS challenge completed successfully")

	// Request certificate from ACME
	log.Printf("[CertManager] Requesting certificate from ACME server...")
	acmeLogger.LogInfo(domain.Name, "request_cert", "Requesting certificate from ACME server")
	bundle, err := s.acmeClient.RequestCertificate(ctx, domains)
	if err != nil {
		// Update status to failed
		errMsg := fmt.Sprintf("Certificate request failed: %v", err)
		log.Printf("[CertManager] ERROR: %s", errMsg)
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, errMsg)
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, fmt.Errorf("failed to request certificate: %w", err)
	}

	log.Printf("[CertManager] Certificate received! Expires at: %v", bundle.ExpiresAt)
	acmeLogger.LogInfo(domain.Name, "cert_received", fmt.Sprintf("Certificate received! Expires at: %v", bundle.ExpiresAt))

	// Save certificate to disk
	storedCert, err := s.certStorage.SaveCertificate(domain.Name, bundle)
	if err != nil {
		// Update status to failed
		s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusFailed, fmt.Sprintf("Certificate storage failed: %v", err))
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	log.Printf("[CertManager] Certificate saved to disk: %s", storedCert.CertPath)
	acmeLogger.LogInfo(domain.Name, "cert_saved", fmt.Sprintf("Certificate saved to disk: %s", storedCert.CertPath))

	// Check if certificate already exists in database
	existingCert, err := s.certRepo.GetByDomainID(ctx, domain.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		acmeLogger.MarkFailed(domain.Name, err)
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
			acmeLogger.MarkFailed(domain.Name, err)
			return nil, fmt.Errorf("failed to update certificate record: %w", err)
		}
		dbCert = existingCert
		acmeLogger.LogInfo(domain.Name, "db_updated", "Certificate record updated in database")
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
			acmeLogger.MarkFailed(domain.Name, err)
			return nil, fmt.Errorf("failed to create certificate record: %w", err)
		}
		acmeLogger.LogInfo(domain.Name, "db_created", "Certificate record created in database")
	}

	// Clear ACME challenge data after successful certificate generation
	if err := s.ClearACMEChallenge(ctx, domain.ID); err != nil {
		log.Printf("[CertManager] Warning: failed to clear ACME challenge data: %v", err)
	}

	// Update domain status to certificate_issued
	if err := s.domainManager.UpdateStatus(ctx, domain.ID, models.StatusCertificateIssued, ""); err != nil {
		acmeLogger.MarkFailed(domain.Name, err)
		return nil, fmt.Errorf("failed to update domain status to certificate_issued: %w", err)
	}

	// Trigger hot reload to make the new certificate available immediately
	s.triggerHotReload(domain.Name)

	log.Printf("[CertManager] Certificate generation completed successfully for domain: %s", domain.Name)
	acmeLogger.MarkSuccess(domain.Name)

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

// ============================================================================
// Manual DNS Verification Methods
// ============================================================================

// RequestACMEChallenge requests a new ACME challenge and stores it in the domain
// Returns challenge info for user to add DNS record
func (s *certificateManagerService) RequestACMEChallenge(ctx context.Context, domainID uint) (*ACMEChallengeInfo, error) {
	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	// Verify domain is in dns_verified status
	if domain.Status != models.StatusDNSVerified {
		return nil, fmt.Errorf("domain must be in dns_verified status to request ACME challenge, current status: %s", domain.Status)
	}

	// Get ACME logger
	acmeLogger := GetACMELogger()
	acmeLogger.StartDomainLog(domain.Name)

	log.Printf("[CertManager] Requesting ACME challenge for domain: %s", domain.Name)
	acmeLogger.LogInfo(domain.Name, "request_challenge", "Requesting ACME challenge from Let's Encrypt")

	// Request challenge from ACME client
	challengeInfo, err := s.acmeClient.GetDNSChallenge(ctx, domain.Name)
	if err != nil {
		acmeLogger.LogError(domain.Name, "challenge_failed", "Failed to get ACME challenge", err)
		return nil, fmt.Errorf("failed to get ACME challenge: %w", err)
	}

	// Calculate expiration time (ACME challenges typically expire in 7 days, but we use 24 hours for safety)
	expiresAt := time.Now().Add(24 * time.Hour)

	// Store challenge data in domain
	domain.ACMEChallengeToken = challengeInfo.Token
	domain.ACMEChallengeValue = challengeInfo.TXTRecord
	domain.ACMEChallengeExpiresAt = &expiresAt
	domain.ACMEDNSVerified = false
	domain.Status = models.StatusPendingACMEChallenge
	domain.ErrorMessage = ""

	if err := s.domainRepo.Update(ctx, domain); err != nil {
		acmeLogger.LogError(domain.Name, "store_challenge_failed", "Failed to store ACME challenge", err)
		return nil, fmt.Errorf("failed to store ACME challenge: %w", err)
	}

	log.Printf("[CertManager] ACME challenge stored for domain: %s, TXT record: _acme-challenge.%s = %s",
		domain.Name, domain.Name, challengeInfo.TXTRecord)
	acmeLogger.LogInfo(domain.Name, "challenge_stored", "ACME challenge stored successfully")

	return &ACMEChallengeInfo{
		Domain:          domain.Name,
		TXTRecordName:   fmt.Sprintf("_acme-challenge.%s", domain.Name),
		TXTRecordValue:  challengeInfo.TXTRecord,
		ExpiresAt:       expiresAt,
		PropagationNote: "DNS propagation may take up to 24 hours. Please wait a few minutes after adding the record before verifying.",
	}, nil
}

// VerifyACMEDNS checks if the DNS TXT record is correctly configured
// Does NOT submit to Let's Encrypt - just local DNS check
// If challenge has expired, it will auto-request a new challenge and inform the user
func (s *certificateManagerService) VerifyACMEDNS(ctx context.Context, domainID uint) (*ACMEDNSVerificationResult, error) {
	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	// Check if challenge exists
	if domain.ACMEChallengeToken == "" || domain.ACMEChallengeValue == "" {
		return &ACMEDNSVerificationResult{
			Verified:      false,
			ExpectedValue: "",
			FoundValues:   nil,
			Message:       "No active ACME challenge found. Please request a new challenge first.",
			CanProceed:    false,
		}, nil
	}

	// Check if challenge has expired - auto-renew if expired
	if domain.ACMEChallengeExpiresAt != nil && time.Now().After(*domain.ACMEChallengeExpiresAt) {
		log.Printf("[CertManager] ACME challenge expired for domain %s during verification, auto-requesting new challenge", domain.Name)

		// Reset domain status to dns_verified to allow requesting new challenge
		domain.Status = models.StatusDNSVerified
		if err := s.domainRepo.Update(ctx, domain); err != nil {
			return nil, fmt.Errorf("failed to reset domain status for challenge renewal: %w", err)
		}

		// Auto-request new challenge
		newChallenge, err := s.RequestACMEChallenge(ctx, domainID)
		if err != nil {
			return &ACMEDNSVerificationResult{
				Verified:      false,
				ExpectedValue: domain.ACMEChallengeValue,
				FoundValues:   nil,
				Message:       fmt.Sprintf("ACME challenge has expired and auto-renewal failed: %v. Please request a new challenge manually.", err),
				CanProceed:    false,
			}, nil
		}

		// Return result with new challenge info
		return &ACMEDNSVerificationResult{
			Verified:      false,
			ExpectedValue: newChallenge.TXTRecordValue,
			FoundValues:   nil,
			Message:       fmt.Sprintf("ACME challenge has expired. A new challenge has been automatically requested. Please update your DNS TXT record %s with the new value: %s", newChallenge.TXTRecordName, newChallenge.TXTRecordValue),
			CanProceed:    false,
		}, nil
	}

	// Perform DNS lookup for _acme-challenge.{domain}
	challengeDomain := fmt.Sprintf("_acme-challenge.%s", domain.Name)
	foundValues := s.lookupACMETXTRecord(challengeDomain)

	// Compare found values with expected value
	verified := false
	for _, val := range foundValues {
		if val == domain.ACMEChallengeValue {
			verified = true
			break
		}
	}

	result := &ACMEDNSVerificationResult{
		Verified:      verified,
		ExpectedValue: domain.ACMEChallengeValue,
		FoundValues:   foundValues,
		CanProceed:    verified,
	}

	if verified {
		result.Message = "DNS TXT record verified successfully!"

		// Update domain status and acme_dns_verified flag
		domain.ACMEDNSVerified = true
		domain.Status = models.StatusACMEChallengeReady
		if err := s.domainRepo.Update(ctx, domain); err != nil {
			return nil, fmt.Errorf("failed to update domain after DNS verification: %w", err)
		}

		log.Printf("[CertManager] ACME DNS verified for domain: %s", domain.Name)
	} else {
		// Provide detailed error message with expected and found values
		if len(foundValues) == 0 {
			result.Message = fmt.Sprintf("DNS TXT record not found. Please add a TXT record for '%s' with value: '%s'. DNS propagation may take up to 24 hours - please wait a few minutes after adding the record before verifying again.",
				challengeDomain, domain.ACMEChallengeValue)
		} else {
			result.Message = fmt.Sprintf("DNS TXT record value mismatch. Expected: '%s', Found: %v. Please update the TXT record '%s' with the correct value.",
				domain.ACMEChallengeValue, foundValues, challengeDomain)
		}
	}

	return result, nil
}

// lookupACMETXTRecord performs DNS lookup for ACME challenge TXT record
func (s *certificateManagerService) lookupACMETXTRecord(challengeDomain string) []string {
	// Try multiple DNS servers for verification
	dnsServers := []string{
		"8.8.8.8:53",        // Google DNS
		"1.1.1.1:53",        // Cloudflare DNS
		"208.67.222.222:53", // OpenDNS
	}

	var allValues []string
	seenValues := make(map[string]bool)

	for _, server := range dnsServers {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 10 * time.Second}
				return d.DialContext(ctx, "udp", server)
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		txtRecords, err := resolver.LookupTXT(ctx, challengeDomain)
		cancel()

		if err != nil {
			log.Printf("[CertManager] DNS lookup via %s failed: %v", server, err)
			continue
		}

		for _, txt := range txtRecords {
			trimmed := strings.TrimSpace(txt)
			if !seenValues[trimmed] {
				seenValues[trimmed] = true
				allValues = append(allValues, trimmed)
			}
		}
	}

	return allValues
}

// SubmitACMEChallenge submits the challenge to Let's Encrypt for validation
// Should only be called after VerifyACMEDNS returns success
func (s *certificateManagerService) SubmitACMEChallenge(ctx context.Context, domainID uint) error {
	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Verify domain is in acme_challenge_ready status
	if domain.Status != models.StatusACMEChallengeReady {
		return fmt.Errorf("domain must be in acme_challenge_ready status to submit ACME challenge, current status: %s. Please verify DNS first", domain.Status)
	}

	// Verify DNS has been verified
	if !domain.ACMEDNSVerified {
		return fmt.Errorf("DNS has not been verified. Please verify DNS first using VerifyACMEDNS")
	}

	// Get ACME logger
	acmeLogger := GetACMELogger()
	acmeLogger.LogInfo(domain.Name, "submit_challenge", "Submitting ACME challenge to Let's Encrypt")

	log.Printf("[CertManager] Submitting ACME challenge to Let's Encrypt for domain: %s", domain.Name)

	// Update status to pending_certificate
	domain.Status = models.StatusPendingCertificate
	if err := s.domainRepo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to update domain status: %w", err)
	}

	// Complete DNS challenge with Let's Encrypt (this submits and waits for validation)
	if err := s.acmeClient.CompleteDNSChallenge(ctx, domain.Name); err != nil {
		// Update status to failed with helpful message
		errMsg := fmt.Sprintf("ACME validation failed: %v. Please ensure the TXT record is correctly configured and try again.", err)
		domain.Status = models.StatusFailed
		domain.ErrorMessage = errMsg
		s.domainRepo.Update(ctx, domain)

		acmeLogger.LogError(domain.Name, "validation_failed", "ACME validation failed", err)
		return fmt.Errorf("ACME validation failed: %w", err)
	}

	log.Printf("[CertManager] ACME challenge validated successfully for domain: %s", domain.Name)
	acmeLogger.LogInfo(domain.Name, "validation_success", "ACME challenge validated successfully")

	// Proceed to certificate generation
	// Prepare domains for certificate (main domain + mail subdomain for SAN)
	domains := []string{
		domain.Name,
		fmt.Sprintf("mail.%s", domain.Name),
	}

	// Request certificate from ACME
	log.Printf("[CertManager] Requesting certificate from ACME server...")
	acmeLogger.LogInfo(domain.Name, "request_cert", "Requesting certificate from ACME server")
	bundle, err := s.acmeClient.RequestCertificate(ctx, domains)
	if err != nil {
		errMsg := fmt.Sprintf("Certificate request failed: %v", err)
		domain.Status = models.StatusFailed
		domain.ErrorMessage = errMsg
		s.domainRepo.Update(ctx, domain)

		acmeLogger.LogError(domain.Name, "cert_request_failed", "Certificate request failed", err)
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	log.Printf("[CertManager] Certificate received! Expires at: %v", bundle.ExpiresAt)
	acmeLogger.LogInfo(domain.Name, "cert_received", fmt.Sprintf("Certificate received! Expires at: %v", bundle.ExpiresAt))

	// Save certificate to disk
	storedCert, err := s.certStorage.SaveCertificate(domain.Name, bundle)
	if err != nil {
		domain.Status = models.StatusFailed
		domain.ErrorMessage = fmt.Sprintf("Certificate storage failed: %v", err)
		s.domainRepo.Update(ctx, domain)

		acmeLogger.LogError(domain.Name, "cert_storage_failed", "Certificate storage failed", err)
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	log.Printf("[CertManager] Certificate saved to disk: %s", storedCert.CertPath)
	acmeLogger.LogInfo(domain.Name, "cert_saved", fmt.Sprintf("Certificate saved to disk: %s", storedCert.CertPath))

	// Check if certificate already exists in database
	existingCert, err := s.certRepo.GetByDomainID(ctx, domain.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("failed to check existing certificate: %w", err)
	}

	// Create or update certificate record in database
	if existingCert != nil {
		existingCert.CertPath = storedCert.CertPath
		existingCert.KeyPath = storedCert.KeyPath
		existingCert.ExpiresAt = storedCert.ExpiresAt
		existingCert.IssuedAt = storedCert.IssuedAt
		if err := s.certRepo.Update(ctx, existingCert); err != nil {
			return fmt.Errorf("failed to update certificate record: %w", err)
		}
		acmeLogger.LogInfo(domain.Name, "db_updated", "Certificate record updated in database")
	} else {
		dbCert := &models.DomainCertificate{
			DomainID:   domain.ID,
			DomainName: domain.Name,
			CertPath:   storedCert.CertPath,
			KeyPath:    storedCert.KeyPath,
			ExpiresAt:  storedCert.ExpiresAt,
			IssuedAt:   storedCert.IssuedAt,
			AutoRenew:  true,
		}
		if err := s.certRepo.Create(ctx, dbCert); err != nil {
			return fmt.Errorf("failed to create certificate record: %w", err)
		}
		acmeLogger.LogInfo(domain.Name, "db_created", "Certificate record created in database")
	}

	// Clear ACME challenge data and update status
	if err := s.ClearACMEChallenge(ctx, domainID); err != nil {
		log.Printf("[CertManager] Warning: failed to clear ACME challenge data: %v", err)
	}

	// Update domain status to certificate_issued
	domain.Status = models.StatusCertificateIssued
	domain.ErrorMessage = ""
	if err := s.domainRepo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to update domain status to certificate_issued: %w", err)
	}

	// Trigger hot reload
	s.triggerHotReload(domain.Name)

	log.Printf("[CertManager] Certificate generation completed successfully for domain: %s", domain.Name)
	acmeLogger.MarkSuccess(domain.Name)

	return nil
}

// GetACMEStatus returns the current ACME challenge status
func (s *certificateManagerService) GetACMEStatus(ctx context.Context, domainID uint) (*ACMEStatus, error) {
	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	status := &ACMEStatus{
		DNSVerified: domain.ACMEDNSVerified,
		LastUpdated: domain.UpdatedAt,
	}

	// Determine step based on domain status and challenge state
	switch domain.Status {
	case models.StatusDNSVerified:
		status.Step = "ready_to_start"
		status.SuggestedAction = "Click 'Request Challenge' to start the ACME challenge process."

	case models.StatusPendingACMEChallenge:
		// Check if challenge has expired
		if domain.ACMEChallengeExpiresAt != nil && time.Now().After(*domain.ACMEChallengeExpiresAt) {
			// Auto-renew expired challenge
			log.Printf("[CertManager] ACME challenge expired for domain %s, auto-requesting new challenge", domain.Name)

			// Reset domain status to dns_verified to allow requesting new challenge
			domain.Status = models.StatusDNSVerified
			if err := s.domainRepo.Update(ctx, domain); err != nil {
				status.Step = "challenge_expired"
				status.ErrorMessage = "Challenge has expired and status reset failed. Please request a new challenge manually."
				status.SuggestedAction = "Click 'Request Challenge' to get a new challenge."
				return status, nil
			}

			newChallenge, err := s.RequestACMEChallenge(ctx, domainID)
			if err != nil {
				status.Step = "challenge_expired"
				status.ErrorMessage = fmt.Sprintf("Challenge has expired and auto-renewal failed: %v. Please request a new challenge.", err)
				status.SuggestedAction = "Click 'Request Challenge' to get a new challenge."
				return status, nil
			}
			status.ChallengeInfo = newChallenge
			status.Step = "dns_pending"
			status.SuggestedAction = "A new challenge has been automatically requested. Add the TXT record to your DNS provider, then click 'Verify DNS'."
		} else {
			status.Step = "dns_pending"
			if domain.ACMEChallengeValue != "" {
				expiresAt := time.Time{}
				if domain.ACMEChallengeExpiresAt != nil {
					expiresAt = *domain.ACMEChallengeExpiresAt
				}
				status.ChallengeInfo = &ACMEChallengeInfo{
					Domain:          domain.Name,
					TXTRecordName:   fmt.Sprintf("_acme-challenge.%s", domain.Name),
					TXTRecordValue:  domain.ACMEChallengeValue,
					ExpiresAt:       expiresAt,
					PropagationNote: "DNS propagation may take up to 24 hours. Please wait a few minutes after adding the record before verifying.",
				}
			}
			status.SuggestedAction = "Add the TXT record to your DNS provider, then click 'Verify DNS'."
		}

	case models.StatusACMEChallengeReady:
		status.Step = "dns_verified"
		status.SuggestedAction = "DNS verified! Click 'Continue' to submit the challenge to Let's Encrypt."

	case models.StatusPendingCertificate:
		status.Step = "validating"
		status.SuggestedAction = "Please wait while Let's Encrypt validates the challenge..."

	case models.StatusCertificateIssued:
		status.Step = "completed"
		status.SuggestedAction = "Certificate has been issued successfully!"

	case models.StatusActive:
		status.Step = "completed"
		status.SuggestedAction = "Domain is active and ready to receive email."

	case models.StatusFailed:
		status.Step = "failed"
		status.ErrorMessage = domain.ErrorMessage
		status.SuggestedAction = "Please check the error message and try again. You may need to request a new challenge."

	default:
		status.Step = "unknown"
		status.SuggestedAction = "Please start the domain setup process."
	}

	return status, nil
}

// ClearACMEChallenge clears stored challenge data from domain
func (s *certificateManagerService) ClearACMEChallenge(ctx context.Context, domainID uint) error {
	// Get domain
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Clear ACME challenge fields
	domain.ACMEChallengeToken = ""
	domain.ACMEChallengeValue = ""
	domain.ACMEChallengeExpiresAt = nil
	domain.ACMEDNSVerified = false

	if err := s.domainRepo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to clear ACME challenge data: %w", err)
	}

	log.Printf("[CertManager] ACME challenge data cleared for domain: %s", domain.Name)
	return nil
}
