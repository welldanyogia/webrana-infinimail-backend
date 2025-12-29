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
	// Get stored challenge
	challenge, ok := c.activeChallenges[domain]
	if !ok {
		return fmt.Errorf("no active challenge found for domain %s", domain)
	}

	// Accept the challenge
	_, err := c.client.Accept(ctx, challenge)
	if err != nil {
		return fmt.Errorf("failed to accept challenge: %w", err)
	}

	// Get stored order
	order, ok := c.activeOrders[domain]
	if !ok {
		return fmt.Errorf("no active order found for domain %s", domain)
	}

	// Wait for order to be ready
	_, err = c.client.WaitOrder(ctx, order.URI)
	if err != nil {
		return fmt.Errorf("failed waiting for order: %w", err)
	}

	return nil
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
