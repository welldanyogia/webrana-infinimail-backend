package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// DNSVerificationResult contains the results of DNS verification
type DNSVerificationResult struct {
	MXVerified  bool     `json:"mx_verified"`
	AVerified   bool     `json:"a_verified"`
	TXTVerified bool     `json:"txt_verified"`
	AllVerified bool     `json:"all_verified"`
	Errors      []string `json:"errors,omitempty"`
}

// DNSVerifierConfig holds configuration for the DNS verifier service
type DNSVerifierConfig struct {
	SMTPHostname   string
	ServerIP       string
	MaxRetries     int
	RetryDelay     time.Duration
	LookupTimeout  time.Duration
}

// DefaultDNSVerifierConfig returns default configuration for DNS verifier
func DefaultDNSVerifierConfig() DNSVerifierConfig {
	return DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "127.0.0.1",
		MaxRetries:    3,
		RetryDelay:    5 * time.Second,
		LookupTimeout: 10 * time.Second,
	}
}

// DNSVerifierService defines the interface for DNS verification
type DNSVerifierService interface {
	// VerifyDNS checks all required DNS records for a domain
	VerifyDNS(ctx context.Context, domain *models.Domain) (*DNSVerificationResult, error)

	// VerifyMXRecord checks if MX record points to the expected SMTP hostname
	VerifyMXRecord(ctx context.Context, domainName, expectedHost string) (bool, error)

	// VerifyARecord checks if A record resolves to the expected IP
	VerifyARecord(ctx context.Context, hostname, expectedIP string) (bool, error)

	// VerifyTXTRecord checks if TXT record contains the challenge token
	VerifyTXTRecord(ctx context.Context, domainName, challengeToken string) (bool, error)
}

// DNSResolver interface for DNS lookups (allows mocking in tests)
type DNSResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// defaultDNSResolver implements DNSResolver using net package
type defaultDNSResolver struct {
	resolver *net.Resolver
}

func newDefaultDNSResolver(timeout time.Duration) *defaultDNSResolver {
	return &defaultDNSResolver{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: timeout,
				}
				return d.DialContext(ctx, network, address)
			},
		},
	}
}

func (r *defaultDNSResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return r.resolver.LookupMX(ctx, name)
}

func (r *defaultDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.resolver.LookupHost(ctx, host)
}

func (r *defaultDNSResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.resolver.LookupTXT(ctx, name)
}

// dnsVerifierService implements DNSVerifierService
type dnsVerifierService struct {
	repo     repository.DomainRepository
	config   DNSVerifierConfig
	resolver DNSResolver
}

// NewDNSVerifierService creates a new DNSVerifierService instance
func NewDNSVerifierService(repo repository.DomainRepository, config DNSVerifierConfig) DNSVerifierService {
	return &dnsVerifierService{
		repo:     repo,
		config:   config,
		resolver: newDefaultDNSResolver(config.LookupTimeout),
	}
}

// NewDNSVerifierServiceWithResolver creates a new DNSVerifierService with custom resolver (for testing)
func NewDNSVerifierServiceWithResolver(repo repository.DomainRepository, config DNSVerifierConfig, resolver DNSResolver) DNSVerifierService {
	return &dnsVerifierService{
		repo:     repo,
		config:   config,
		resolver: resolver,
	}
}


// getParentDomain extracts the parent domain from a domain name
// e.g., "mail.example.com" -> "example.com", "example.com" -> "example.com"
func getParentDomain(domainName string) string {
	// If domain starts with "mail.", extract the parent domain
	if strings.HasPrefix(strings.ToLower(domainName), "mail.") {
		return domainName[5:] // Remove "mail." prefix
	}
	return domainName
}

// getMailHostname returns the mail hostname for a domain
// e.g., "example.com" -> "mail.example.com", "mail.example.com" -> "mail.example.com"
func getMailHostname(domainName string) string {
	// If domain already starts with "mail.", return as-is
	if strings.HasPrefix(strings.ToLower(domainName), "mail.") {
		return domainName
	}
	return fmt.Sprintf("mail.%s", domainName)
}

// VerifyDNS checks all required DNS records for a domain with retry mechanism
func (s *dnsVerifierService) VerifyDNS(ctx context.Context, domain *models.Domain) (*DNSVerificationResult, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain cannot be nil")
	}

	result := &DNSVerificationResult{
		Errors: make([]string, 0),
	}

	// Extract parent domain for MX and TXT lookups
	// e.g., if domain.Name is "mail.example.com", parentDomain is "example.com"
	parentDomain := getParentDomain(domain.Name)
	
	// Get mail hostname for A record lookup
	// e.g., if domain.Name is "example.com", mailHostname is "mail.example.com"
	// e.g., if domain.Name is "mail.example.com", mailHostname is "mail.example.com"
	mailHostname := getMailHostname(domain.Name)

	// Verify MX record (lookup on parent domain)
	mxVerified, err := s.verifyWithRetry(ctx, func(ctx context.Context) (bool, error) {
		return s.VerifyMXRecord(ctx, parentDomain, s.config.SMTPHostname)
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("MX verification failed: %v", err))
	}
	result.MXVerified = mxVerified

	// Verify A record for mail subdomain
	aVerified, err := s.verifyWithRetry(ctx, func(ctx context.Context) (bool, error) {
		return s.VerifyARecord(ctx, mailHostname, s.config.ServerIP)
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("A record verification failed: %v", err))
	}
	result.AVerified = aVerified

	// Verify TXT record for challenge (lookup on parent domain)
	txtVerified, err := s.verifyWithRetry(ctx, func(ctx context.Context) (bool, error) {
		return s.VerifyTXTRecord(ctx, parentDomain, domain.DNSChallenge)
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("TXT verification failed: %v", err))
	}
	result.TXTVerified = txtVerified

	// Set all verified flag
	result.AllVerified = result.MXVerified && result.AVerified && result.TXTVerified

	return result, nil
}

// verifyWithRetry executes a verification function with retry mechanism
func (s *dnsVerifierService) verifyWithRetry(ctx context.Context, verifyFunc func(context.Context) (bool, error)) (bool, error) {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		verified, err := verifyFunc(ctx)
		if err == nil && verified {
			return true, nil
		}

		if err != nil {
			lastErr = err
		}

		// Don't sleep on the last attempt
		if attempt < s.config.MaxRetries {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}
	}

	if lastErr != nil {
		return false, lastErr
	}
	return false, nil
}

// VerifyMXRecord checks if MX record points to the expected SMTP hostname
func (s *dnsVerifierService) VerifyMXRecord(ctx context.Context, domainName, expectedHost string) (bool, error) {
	if domainName == "" {
		return false, fmt.Errorf("domain name cannot be empty")
	}
	if expectedHost == "" {
		return false, fmt.Errorf("expected host cannot be empty")
	}

	// Normalize expected host (remove trailing dot if present)
	expectedHost = strings.TrimSuffix(strings.ToLower(expectedHost), ".")

	mxRecords, err := s.resolver.LookupMX(ctx, domainName)
	if err != nil {
		return false, fmt.Errorf("MX lookup failed for %s: %w", domainName, err)
	}

	if len(mxRecords) == 0 {
		return false, fmt.Errorf("no MX records found for %s", domainName)
	}

	// Check if any MX record matches the expected host
	for _, mx := range mxRecords {
		// Normalize MX host (remove trailing dot)
		mxHost := strings.TrimSuffix(strings.ToLower(mx.Host), ".")
		if mxHost == expectedHost {
			return true, nil
		}
	}

	return false, fmt.Errorf("MX record mismatch: expected %s, found %s", expectedHost, mxRecords[0].Host)
}

// VerifyARecord checks if A record resolves to the expected IP
func (s *dnsVerifierService) VerifyARecord(ctx context.Context, hostname, expectedIP string) (bool, error) {
	if hostname == "" {
		return false, fmt.Errorf("hostname cannot be empty")
	}
	if expectedIP == "" {
		return false, fmt.Errorf("expected IP cannot be empty")
	}

	// Normalize expected IP
	expectedIP = strings.TrimSpace(expectedIP)

	ips, err := s.resolver.LookupHost(ctx, hostname)
	if err != nil {
		return false, fmt.Errorf("A record lookup failed for %s: %w", hostname, err)
	}

	if len(ips) == 0 {
		return false, fmt.Errorf("no A records found for %s", hostname)
	}

	// Check if any IP matches the expected IP
	for _, ip := range ips {
		if strings.TrimSpace(ip) == expectedIP {
			return true, nil
		}
	}

	return false, fmt.Errorf("A record mismatch: expected %s, found %v", expectedIP, ips)
}

// VerifyTXTRecord checks if TXT record contains the challenge token
func (s *dnsVerifierService) VerifyTXTRecord(ctx context.Context, domainName, challengeToken string) (bool, error) {
	if domainName == "" {
		return false, fmt.Errorf("domain name cannot be empty")
	}
	if challengeToken == "" {
		return false, fmt.Errorf("challenge token cannot be empty")
	}

	// TXT record is at _infinimail.{domain}
	txtDomain := fmt.Sprintf("_infinimail.%s", domainName)
	expectedValue := fmt.Sprintf("infinimail-verify=%s", challengeToken)

	txtRecords, err := s.resolver.LookupTXT(ctx, txtDomain)
	if err != nil {
		return false, fmt.Errorf("TXT lookup failed for %s: %w", txtDomain, err)
	}

	if len(txtRecords) == 0 {
		return false, fmt.Errorf("no TXT records found for %s", txtDomain)
	}

	// Check if any TXT record matches the expected value
	for _, txt := range txtRecords {
		// TXT records may be split, so we check if the expected value is contained
		if strings.TrimSpace(txt) == expectedValue {
			return true, nil
		}
	}

	return false, fmt.Errorf("TXT record mismatch: expected %s at %s", expectedValue, txtDomain)
}
