package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// DNSRecord represents a single DNS record for setup guide
type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"`
	TTL      int    `json:"ttl"`
}

// DNSGuide contains all DNS records needed for domain setup
type DNSGuide struct {
	MXRecord          DNSRecord `json:"mx_record"`
	ARecord           DNSRecord `json:"a_record"`
	TXTRecord         DNSRecord `json:"txt_record"`
	ACMEChallengeInfo string    `json:"acme_challenge_info,omitempty"` // Info about ACME challenge (set during cert generation)
	SMTPHost          string    `json:"smtp_host"`
	ServerIP          string    `json:"server_ip"`
}

// DomainManagerConfig holds configuration for the domain manager service
type DomainManagerConfig struct {
	SMTPHostname string
	ServerIP     string
}

// DomainManagerService defines the interface for domain lifecycle management
type DomainManagerService interface {
	// CreateDomain creates a new domain with pending_dns status and generates challenge token
	CreateDomain(ctx context.Context, name string) (*models.Domain, error)

	// UpdateStatus updates domain status with optional error message
	UpdateStatus(ctx context.Context, domainID uint, status models.DomainStatus, errorMsg string) error

	// GetDNSGuide returns DNS records to configure for a domain
	GetDNSGuide(ctx context.Context, domainID uint) (*DNSGuide, error)

	// ActivateDomain sets domain to active status after certificate is issued
	ActivateDomain(ctx context.Context, domainID uint) error

	// GetDomain retrieves a domain by ID
	GetDomain(ctx context.Context, domainID uint) (*models.Domain, error)

	// GenerateChallengeForLegacyDomain generates a DNS challenge token for legacy domains
	// that were created before the Auto SSL feature was implemented
	GenerateChallengeForLegacyDomain(ctx context.Context, domainID uint) error
}

// domainManagerService implements DomainManagerService
type domainManagerService struct {
	repo   repository.DomainRepository
	config DomainManagerConfig
}

// NewDomainManagerService creates a new DomainManagerService instance
func NewDomainManagerService(repo repository.DomainRepository, config DomainManagerConfig) DomainManagerService {
	return &domainManagerService{
		repo:   repo,
		config: config,
	}
}

// CreateDomain creates a new domain with pending_dns status and generates a unique challenge token
func (s *domainManagerService) CreateDomain(ctx context.Context, name string) (*models.Domain, error) {
	// Validate domain name
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	// Generate unique DNS challenge token
	challengeToken, err := generateChallengeToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge token: %w", err)
	}

	// Create domain with pending_dns status
	domain := &models.Domain{
		Name:         name,
		IsActive:     false,
		Status:       models.StatusPendingDNS,
		DNSChallenge: challengeToken,
	}

	if err := s.repo.Create(ctx, domain); err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}

	return domain, nil
}

// UpdateStatus updates the domain status with optional error message
func (s *domainManagerService) UpdateStatus(ctx context.Context, domainID uint, status models.DomainStatus, errorMsg string) error {
	// Validate status
	if !status.IsValid() {
		return fmt.Errorf("invalid domain status: %s", status)
	}

	// Get existing domain
	domain, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Update status and error message
	domain.Status = status
	domain.ErrorMessage = errorMsg

	// Clear error message if status is not failed
	if status != models.StatusFailed {
		domain.ErrorMessage = ""
	}

	if err := s.repo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to update domain status: %w", err)
	}

	return nil
}

// GetDNSGuide returns the DNS records needed for domain setup
func (s *domainManagerService) GetDNSGuide(ctx context.Context, domainID uint) (*DNSGuide, error) {
	// Get domain
	domain, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	// Extract parent domain for MX and TXT records
	// e.g., if domain.Name is "mail.example.com", parentDomain is "example.com"
	parentDomain := getParentDomainName(domain.Name)
	
	// Get mail hostname for A record
	// e.g., if domain.Name is "example.com", mailHostname is "mail.example.com"
	mailHostname := getMailHostnameName(domain.Name)

	// Generate DNS guide based on configuration
	guide := &DNSGuide{
		SMTPHost: s.config.SMTPHostname,
		ServerIP: s.config.ServerIP,
		MXRecord: DNSRecord{
			Type:     "MX",
			Name:     parentDomain,
			Value:    s.config.SMTPHostname,
			Priority: 10,
			TTL:      3600,
		},
		ARecord: DNSRecord{
			Type:  "A",
			Name:  mailHostname,
			Value: s.config.ServerIP,
			TTL:   3600,
		},
		TXTRecord: DNSRecord{
			Type:  "TXT",
			Name:  fmt.Sprintf("_infinimail.%s", parentDomain),
			Value: fmt.Sprintf("infinimail-verify=%s", domain.DNSChallenge),
			TTL:   3600,
		},
	}

	return guide, nil
}

// getParentDomainName extracts the parent domain from a domain name
// e.g., "mail.example.com" -> "example.com", "example.com" -> "example.com"
func getParentDomainName(domainName string) string {
	// If domain starts with "mail.", extract the parent domain
	if strings.HasPrefix(strings.ToLower(domainName), "mail.") {
		return domainName[5:] // Remove "mail." prefix
	}
	return domainName
}

// getMailHostnameName returns the mail hostname for a domain
// e.g., "example.com" -> "mail.example.com", "mail.example.com" -> "mail.example.com"
func getMailHostnameName(domainName string) string {
	// If domain already starts with "mail.", return as-is
	if strings.HasPrefix(strings.ToLower(domainName), "mail.") {
		return domainName
	}
	return fmt.Sprintf("mail.%s", domainName)
}

// ActivateDomain sets the domain to active status
func (s *domainManagerService) ActivateDomain(ctx context.Context, domainID uint) error {
	// Get existing domain
	domain, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Verify domain is in certificate_issued status
	if domain.Status != models.StatusCertificateIssued {
		return fmt.Errorf("domain must be in certificate_issued status to activate, current status: %s", domain.Status)
	}

	// Update to active status
	domain.Status = models.StatusActive
	domain.IsActive = true
	domain.ErrorMessage = ""

	if err := s.repo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to activate domain: %w", err)
	}

	return nil
}

// GetDomain retrieves a domain by ID
func (s *domainManagerService) GetDomain(ctx context.Context, domainID uint) (*models.Domain, error) {
	return s.repo.GetByID(ctx, domainID)
}

// GenerateChallengeForLegacyDomain generates a DNS challenge token for legacy domains
// that were created before the Auto SSL feature was implemented
func (s *domainManagerService) GenerateChallengeForLegacyDomain(ctx context.Context, domainID uint) error {
	// Get existing domain
	domain, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Only generate if challenge is empty
	if domain.DNSChallenge != "" {
		return nil // Already has a challenge token
	}

	// Generate unique DNS challenge token
	challengeToken, err := generateChallengeToken()
	if err != nil {
		return fmt.Errorf("failed to generate challenge token: %w", err)
	}

	// Update domain with challenge token and set status to pending_dns
	domain.DNSChallenge = challengeToken
	domain.Status = models.StatusPendingDNS

	if err := s.repo.Update(ctx, domain); err != nil {
		return fmt.Errorf("failed to update domain with challenge token: %w", err)
	}

	return nil
}

// generateChallengeToken generates a unique 32-character hex token for DNS verification
func generateChallengeToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
