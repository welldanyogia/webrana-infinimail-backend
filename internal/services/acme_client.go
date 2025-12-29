package services

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
)

// ACMEDirectoryURL constants for Let's Encrypt
const (
	LetsEncryptProduction = "https://acme-v02.api.letsencrypt.org/directory"
	LetsEncryptStaging    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// CertificateBundle contains the issued certificate and private key
type CertificateBundle struct {
	Certificate []byte    // PEM-encoded certificate
	PrivateKey  []byte    // PEM-encoded private key
	Chain       []byte    // PEM-encoded certificate chain
	ExpiresAt   time.Time // Certificate expiration time
}

// DNSChallengeInfo contains information for DNS-01 challenge
type DNSChallengeInfo struct {
	Domain    string `json:"domain"`
	Token     string `json:"token"`
	KeyAuth   string `json:"key_auth"`
	TXTRecord string `json:"txt_record"` // The value to put in _acme-challenge.domain TXT record
}

// ACMEClientConfig holds configuration for the ACME client
type ACMEClientConfig struct {
	DirectoryURL string // ACME directory URL (production or staging)
	Email        string // Contact email for account registration
	Staging      bool   // Whether to use staging environment
}

// ACMEClient defines the interface for ACME operations
type ACMEClient interface {
	// RegisterAccount registers or retrieves an existing ACME account
	RegisterAccount(ctx context.Context) error

	// GetDNSChallenge initiates an order and returns DNS-01 challenge info
	GetDNSChallenge(ctx context.Context, domain string) (*DNSChallengeInfo, error)

	// CompleteDNSChallenge completes the DNS-01 challenge after TXT record is set
	CompleteDNSChallenge(ctx context.Context, domain string) error

	// RequestCertificate requests a certificate for the given domains
	RequestCertificate(ctx context.Context, domains []string) (*CertificateBundle, error)

	// GetAccountKey returns the account private key (for persistence)
	GetAccountKey() crypto.PrivateKey
}


// acmeClient implements ACMEClient interface
type acmeClient struct {
	config     ACMEClientConfig
	client     *acme.Client
	accountKey *ecdsa.PrivateKey
	account    *acme.Account

	// Store active orders and challenges for multi-step flow
	activeOrders     map[string]*acme.Order
	activeChallenges map[string]*acme.Challenge
}

// NewACMEClient creates a new ACME client instance
func NewACMEClient(config ACMEClientConfig) (ACMEClient, error) {
	// Generate account key if not provided
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate account key: %w", err)
	}

	// Determine directory URL
	directoryURL := config.DirectoryURL
	if directoryURL == "" {
		if config.Staging {
			directoryURL = LetsEncryptStaging
		} else {
			directoryURL = LetsEncryptProduction
		}
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directoryURL,
	}

	return &acmeClient{
		config:           config,
		client:           client,
		accountKey:       accountKey,
		activeOrders:     make(map[string]*acme.Order),
		activeChallenges: make(map[string]*acme.Challenge),
	}, nil
}

// NewACMEClientWithKey creates a new ACME client with an existing account key
func NewACMEClientWithKey(config ACMEClientConfig, accountKey *ecdsa.PrivateKey) (ACMEClient, error) {
	// Determine directory URL
	directoryURL := config.DirectoryURL
	if directoryURL == "" {
		if config.Staging {
			directoryURL = LetsEncryptStaging
		} else {
			directoryURL = LetsEncryptProduction
		}
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directoryURL,
	}

	return &acmeClient{
		config:           config,
		client:           client,
		accountKey:       accountKey,
		activeOrders:     make(map[string]*acme.Order),
		activeChallenges: make(map[string]*acme.Challenge),
	}, nil
}

// RegisterAccount registers or retrieves an existing ACME account
func (c *acmeClient) RegisterAccount(ctx context.Context) error {
	// Build contact list
	var contact []string
	if c.config.Email != "" {
		contact = []string{"mailto:" + c.config.Email}
	}

	// Try to register new account
	account := &acme.Account{
		Contact: contact,
	}

	registeredAccount, err := c.client.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		// Check if account already exists
		if acmeErr, ok := err.(*acme.Error); ok && acmeErr.StatusCode == 409 {
			// Account exists, retrieve it
			registeredAccount, err = c.client.GetReg(ctx, "")
			if err != nil {
				return fmt.Errorf("failed to retrieve existing account: %w", err)
			}
		} else {
			return fmt.Errorf("failed to register account: %w", err)
		}
	}

	c.account = registeredAccount
	return nil
}


// GetDNSChallenge initiates an order and returns DNS-01 challenge info
func (c *acmeClient) GetDNSChallenge(ctx context.Context, domain string) (*DNSChallengeInfo, error) {
	// Ensure account is registered
	if c.account == nil {
		if err := c.RegisterAccount(ctx); err != nil {
			return nil, fmt.Errorf("failed to register account: %w", err)
		}
	}

	// Create authorization identifiers
	identifiers := []acme.AuthzID{
		{Type: "dns", Value: domain},
	}

	// Create new order
	order, err := c.client.AuthorizeOrder(ctx, identifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Store order for later use
	c.activeOrders[domain] = order

	// Get authorization URL (first one for single domain)
	if len(order.AuthzURLs) == 0 {
		return nil, fmt.Errorf("no authorization URLs in order")
	}

	// Get authorization
	authz, err := c.client.GetAuthorization(ctx, order.AuthzURLs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization: %w", err)
	}

	// Find DNS-01 challenge
	var dnsChallenge *acme.Challenge
	for _, ch := range authz.Challenges {
		if ch.Type == "dns-01" {
			dnsChallenge = ch
			break
		}
	}

	if dnsChallenge == nil {
		return nil, fmt.Errorf("DNS-01 challenge not available for domain %s", domain)
	}

	// Store challenge for later
	c.activeChallenges[domain] = dnsChallenge

	// Compute DNS TXT record value
	txtRecord, err := c.client.DNS01ChallengeRecord(dnsChallenge.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to compute DNS challenge record: %w", err)
	}

	return &DNSChallengeInfo{
		Domain:    domain,
		Token:     dnsChallenge.Token,
		KeyAuth:   txtRecord, // This is the key authorization hash
		TXTRecord: txtRecord, // Value to put in _acme-challenge.domain
	}, nil
}

// CompleteDNSChallenge completes the DNS-01 challenge after TXT record is set
func (c *acmeClient) CompleteDNSChallenge(ctx context.Context, domain string) error {
	logger := GetACMELogger()

	// Get stored challenge
	challenge, ok := c.activeChallenges[domain]
	if !ok {
		return fmt.Errorf("no active challenge found for domain %s", domain)
	}

	// Get stored order
	order, ok := c.activeOrders[domain]
	if !ok {
		return fmt.Errorf("no active order found for domain %s", domain)
	}

	// Compute expected TXT record value for verification
	expectedTXT, err := c.client.DNS01ChallengeRecord(challenge.Token)
	if err != nil {
		return fmt.Errorf("failed to compute expected TXT record: %w", err)
	}

	log.Printf("[ACME] Starting DNS-01 challenge for domain: %s", domain)
	logger.LogInfo(domain, "challenge_start", "Starting DNS-01 challenge")
	logger.LogDebug(domain, "txt_record", fmt.Sprintf("Expected TXT record: _acme-challenge.%s = %s", domain, expectedTXT), map[string]string{
		"record_name":  "_acme-challenge." + domain,
		"record_value": expectedTXT,
	})

	// Pre-check: Verify TXT record exists before accepting challenge
	// This helps catch DNS propagation issues early
	txtFound, txtValues := c.verifyDNSTXTRecord(domain, expectedTXT)
	if !txtFound {
		log.Printf("[ACME] WARNING: TXT record not found or doesn't match. Found values: %v", txtValues)
		logger.LogWarning(domain, "dns_precheck", "TXT record not found or doesn't match", map[string]interface{}{
			"expected":     expectedTXT,
			"found_values": txtValues,
		})
		log.Printf("[ACME] Waiting for DNS propagation...")
	} else {
		log.Printf("[ACME] TXT record verified successfully")
		logger.LogInfo(domain, "dns_precheck", "TXT record verified successfully before propagation delay")
	}

	// Wait for DNS propagation before accepting challenge
	// Let's Encrypt validates from multiple vantage points globally
	propagationDelay := 90 * time.Second // 90 seconds for DNS propagation
	log.Printf("[ACME] Waiting %v for DNS propagation...", propagationDelay)
	logger.LogInfo(domain, "dns_propagation", fmt.Sprintf("Waiting %v for DNS propagation", propagationDelay))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(propagationDelay):
		// Continue after propagation delay
	}

	// Re-verify TXT record after waiting
	txtFound, txtValues = c.verifyDNSTXTRecord(domain, expectedTXT)
	if !txtFound {
		errMsg := fmt.Sprintf("DNS TXT record verification failed after waiting. Expected _acme-challenge.%s with value %s, but found: %v", domain, expectedTXT, txtValues)
		logger.LogError(domain, "dns_verify_failed", errMsg, fmt.Errorf("TXT record not found"))
		return fmt.Errorf("%s. Please ensure the TXT record is properly configured and has propagated", errMsg)
	}
	log.Printf("[ACME] TXT record verified after propagation delay")
	logger.LogInfo(domain, "dns_verified", "TXT record verified after propagation delay")

	// Accept the challenge - this tells Let's Encrypt to start validation
	log.Printf("[ACME] Accepting challenge...")
	logger.LogInfo(domain, "challenge_accept", "Accepting challenge with Let's Encrypt")
	_, err = c.client.Accept(ctx, challenge)
	if err != nil {
		logger.LogError(domain, "challenge_accept_failed", "Failed to accept challenge", err)
		return fmt.Errorf("failed to accept challenge: %w", err)
	}
	log.Printf("[ACME] Challenge accepted, waiting for Let's Encrypt validation...")
	logger.LogInfo(domain, "challenge_accepted", "Challenge accepted, waiting for Let's Encrypt validation")

	// Poll authorization status to get detailed error information
	// This is more informative than just waiting for order
	authzURL := order.AuthzURLs[0]
	authzCtx, authzCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer authzCancel()

	// Poll for authorization status with detailed error reporting
	pollInterval := 5 * time.Second
	maxAttempts := 60 // 5 minutes max
	for attempt := 0; attempt < maxAttempts; attempt++ {
		authz, err := c.client.GetAuthorization(authzCtx, authzURL)
		if err != nil {
			log.Printf("[ACME] Error getting authorization status: %v", err)
			logger.LogError(domain, "authz_poll_error", "Error getting authorization status", err)
			return fmt.Errorf("failed to get authorization status: %w", err)
		}

		log.Printf("[ACME] Authorization status: %s (attempt %d/%d)", authz.Status, attempt+1, maxAttempts)
		logger.LogDebug(domain, "authz_poll", fmt.Sprintf("Authorization status: %s (attempt %d/%d)", authz.Status, attempt+1, maxAttempts), map[string]interface{}{
			"status":  authz.Status,
			"attempt": attempt + 1,
		})

		switch authz.Status {
		case acme.StatusValid:
			log.Printf("[ACME] Authorization validated successfully!")
			logger.LogInfo(domain, "authz_valid", "Authorization validated successfully by Let's Encrypt")
			// Authorization is valid, now wait for order to be ready
			orderCtx, orderCancel := context.WithTimeout(ctx, 2*time.Minute)
			defer orderCancel()
			_, err = c.client.WaitOrder(orderCtx, order.URI)
			if err != nil {
				logger.LogError(domain, "order_wait_failed", "Failed waiting for order after authorization", err)
				return fmt.Errorf("failed waiting for order after authorization: %w", err)
			}
			logger.LogInfo(domain, "order_ready", "Order is ready for certificate finalization")
			return nil

		case acme.StatusInvalid:
			// Get detailed error from challenge
			errDetails := c.getAuthorizationErrorDetails(authz)
			log.Printf("[ACME] Authorization INVALID: %s", errDetails)
			logger.LogError(domain, "authz_invalid", "Authorization failed", fmt.Errorf(errDetails))
			return fmt.Errorf("ACME authorization failed: %s", errDetails)

		case acme.StatusPending, acme.StatusProcessing:
			// Still processing, continue polling
			log.Printf("[ACME] Authorization still processing, waiting %v...", pollInterval)
			select {
			case <-authzCtx.Done():
				logger.LogError(domain, "authz_timeout", "Timeout waiting for authorization", authzCtx.Err())
				return fmt.Errorf("timeout waiting for authorization: %w", authzCtx.Err())
			case <-time.After(pollInterval):
				continue
			}

		case acme.StatusDeactivated, acme.StatusExpired, acme.StatusRevoked:
			errMsg := fmt.Sprintf("authorization is %s, cannot proceed", authz.Status)
			logger.LogError(domain, "authz_invalid_status", errMsg, fmt.Errorf(errMsg))
			return fmt.Errorf(errMsg)

		default:
			log.Printf("[ACME] Unknown authorization status: %s", authz.Status)
			logger.LogWarning(domain, "authz_unknown_status", fmt.Sprintf("Unknown authorization status: %s", authz.Status), nil)
		}
	}

	errMsg := fmt.Sprintf("timeout: authorization did not complete within %d attempts", maxAttempts)
	logger.LogError(domain, "authz_max_attempts", errMsg, fmt.Errorf(errMsg))
	return fmt.Errorf(errMsg)
}

// verifyDNSTXTRecord checks if the expected TXT record exists
func (c *acmeClient) verifyDNSTXTRecord(domain, expectedValue string) (bool, []string) {
	challengeDomain := "_acme-challenge." + domain

	// Try multiple DNS servers for verification
	dnsServers := []string{
		"8.8.8.8:53",        // Google DNS
		"1.1.1.1:53",        // Cloudflare DNS
		"208.67.222.222:53", // OpenDNS
	}

	var allValues []string
	found := false

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
			log.Printf("[ACME] DNS lookup via %s failed: %v", server, err)
			continue
		}

		for _, txt := range txtRecords {
			allValues = append(allValues, txt)
			if strings.TrimSpace(txt) == strings.TrimSpace(expectedValue) {
				found = true
				log.Printf("[ACME] TXT record found via %s: %s", server, txt)
			}
		}
	}

	return found, allValues
}

// getAuthorizationErrorDetails extracts detailed error information from authorization
func (c *acmeClient) getAuthorizationErrorDetails(authz *acme.Authorization) string {
	var details []string

	for _, ch := range authz.Challenges {
		if ch.Status == acme.StatusInvalid && ch.Error != nil {
			// ch.Error is *acme.Error type
			if acmeErr, ok := ch.Error.(*acme.Error); ok {
				errMsg := fmt.Sprintf("Challenge %s failed: %s (type: %s)", ch.Type, acmeErr.Detail, acmeErr.ProblemType)
				if acmeErr.StatusCode != 0 {
					errMsg += fmt.Sprintf(" (HTTP %d)", acmeErr.StatusCode)
				}
				details = append(details, errMsg)
			} else {
				// Fallback for generic error
				details = append(details, fmt.Sprintf("Challenge %s failed: %v", ch.Type, ch.Error))
			}
		}
	}

	if len(details) == 0 {
		return "Authorization failed with no specific error details"
	}

	return strings.Join(details, "; ")
}


// RequestCertificate requests a certificate for the given domains
func (c *acmeClient) RequestCertificate(ctx context.Context, domains []string) (*CertificateBundle, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("at least one domain is required")
	}

	// Ensure account is registered
	if c.account == nil {
		if err := c.RegisterAccount(ctx); err != nil {
			return nil, fmt.Errorf("failed to register account: %w", err)
		}
	}

	// Check if we have an active order for the primary domain
	primaryDomain := domains[0]
	order, ok := c.activeOrders[primaryDomain]
	if !ok {
		return nil, fmt.Errorf("no active order found for domain %s, call GetDNSChallenge first", primaryDomain)
	}

	// Wait for order to be ready (in case it's still processing)
	order, err := c.client.WaitOrder(ctx, order.URI)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for order: %w", err)
	}

	// Check order status
	if order.Status != acme.StatusReady {
		return nil, fmt.Errorf("order is not ready, status: %s", order.Status)
	}

	// Generate certificate private key
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate key: %w", err)
	}

	// Create CSR
	csr, err := createCSR(certKey, domains)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// Finalize order with CSR
	derCerts, _, err := c.client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize order: %w", err)
	}

	// Parse certificates to get expiry
	var expiresAt time.Time
	if len(derCerts) > 0 {
		cert, err := x509.ParseCertificate(derCerts[0])
		if err == nil {
			expiresAt = cert.NotAfter
		}
	}

	// Encode certificate chain to PEM
	var certPEM, chainPEM []byte
	for i, der := range derCerts {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		}
		if i == 0 {
			certPEM = pem.EncodeToMemory(block)
		} else {
			chainPEM = append(chainPEM, pem.EncodeToMemory(block)...)
		}
	}

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	// Clean up stored order and challenge
	delete(c.activeOrders, primaryDomain)
	delete(c.activeChallenges, primaryDomain)

	return &CertificateBundle{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		Chain:       chainPEM,
		ExpiresAt:   expiresAt,
	}, nil
}

// GetAccountKey returns the account private key
func (c *acmeClient) GetAccountKey() crypto.PrivateKey {
	return c.accountKey
}

// createCSR creates a Certificate Signing Request for the given domains
func createCSR(key *ecdsa.PrivateKey, domains []string) ([]byte, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("at least one domain is required")
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		DNSNames: domains,
	}

	return x509.CreateCertificateRequest(rand.Reader, template, key)
}
