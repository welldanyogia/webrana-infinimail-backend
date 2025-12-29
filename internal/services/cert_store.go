package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// CertificateStore manages certificates for SNI (Server Name Indication) support
// It provides an in-memory cache of TLS certificates that can be dynamically
// loaded and reloaded without server restart
type CertificateStore interface {
	// LoadCertificate loads a certificate from file for a specific domain
	LoadCertificate(domainName string) (*tls.Certificate, error)

	// AddCertificate adds a certificate to the in-memory store
	AddCertificate(domainName string, cert *tls.Certificate) error

	// RemoveCertificate removes a certificate from the in-memory store
	RemoveCertificate(domainName string) error

	// GetCertificateFunc returns a function suitable for tls.Config.GetCertificate
	// This function selects the appropriate certificate based on SNI hostname
	GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error)

	// ReloadAll reloads all certificates from disk based on database records
	ReloadAll(ctx context.Context) error

	// GetCertificate retrieves a certificate from the in-memory store
	GetCertificate(domainName string) (*tls.Certificate, error)

	// SetDefaultCertificate sets the default certificate to use when no SNI match is found
	SetDefaultCertificate(cert *tls.Certificate)

	// GetDefaultCertificate returns the default certificate
	GetDefaultCertificate() *tls.Certificate

	// Count returns the number of certificates in the store
	Count() int

	// LoadAndAddCertificate loads a certificate from disk and adds it to the store
	// This is used for hot reload after a new certificate is generated
	LoadAndAddCertificate(domainName string) error

	// RegisterReloadCallback registers a callback function to be called after certificates are reloaded
	RegisterReloadCallback(callback func())
}

// CertificateReloadNotifier is an interface for components that need to be notified
// when certificates are reloaded (e.g., SMTP server)
type CertificateReloadNotifier interface {
	// OnCertificateReload is called when certificates have been reloaded
	OnCertificateReload()
}

// certificateStore implements CertificateStore interface
type certificateStore struct {
	mu              sync.RWMutex
	certificates    map[string]*tls.Certificate // domain name -> certificate
	defaultCert     *tls.Certificate
	certStorage     CertStorage
	certRepo        repository.CertificateRepository
	reloadCallbacks []func()
}

// CertificateStoreConfig holds configuration for the certificate store
type CertificateStoreConfig struct {
	CertStorage CertStorage
	CertRepo    repository.CertificateRepository
}

// NewCertificateStore creates a new CertificateStore instance
func NewCertificateStore(config CertificateStoreConfig) (CertificateStore, error) {
	if config.CertStorage == nil {
		return nil, fmt.Errorf("cert storage cannot be nil")
	}

	return &certificateStore{
		certificates:    make(map[string]*tls.Certificate),
		certStorage:     config.CertStorage,
		certRepo:        config.CertRepo,
		reloadCallbacks: make([]func(), 0),
	}, nil
}


// LoadCertificate loads a certificate from file for a specific domain
// It reads the certificate and key files from the configured storage path
// and parses them into a tls.Certificate
func (s *certificateStore) LoadCertificate(domainName string) (*tls.Certificate, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	// Check if certificate exists in storage
	if !s.certStorage.CertificateExists(domainName) {
		return nil, fmt.Errorf("certificate not found for domain: %s", domainName)
	}

	// Get file paths
	certPath := s.certStorage.GetCertificatePath(domainName)
	keyPath := s.certStorage.GetKeyPath(domainName)

	// Load certificate from files
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate for domain %s: %w", domainName, err)
	}

	return &cert, nil
}

// AddCertificate adds a certificate to the in-memory store
// This allows the certificate to be served via SNI
func (s *certificateStore) AddCertificate(domainName string, cert *tls.Certificate) error {
	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.certificates[domainName] = cert
	return nil
}

// RemoveCertificate removes a certificate from the in-memory store
func (s *certificateStore) RemoveCertificate(domainName string) error {
	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.certificates, domainName)
	return nil
}

// GetCertificateFunc returns a function suitable for tls.Config.GetCertificate
// This function implements SNI (Server Name Indication) certificate selection
// It returns the certificate matching the requested hostname, or the default
// certificate if no match is found
func (s *certificateStore) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		serverName := hello.ServerName

		// Try to find certificate for the exact domain
		if serverName != "" {
			s.mu.RLock()
			cert, ok := s.certificates[serverName]
			s.mu.RUnlock()

			if ok {
				return cert, nil
			}

			// Try to find certificate for parent domain (e.g., mail.example.com -> example.com)
			// This handles subdomains like mail.example.com using the example.com certificate
			parentDomain := extractParentDomain(serverName)
			if parentDomain != "" && parentDomain != serverName {
				s.mu.RLock()
				cert, ok = s.certificates[parentDomain]
				s.mu.RUnlock()

				if ok {
					return cert, nil
				}
			}
		}

		// Return default certificate if no match found
		s.mu.RLock()
		defaultCert := s.defaultCert
		s.mu.RUnlock()

		if defaultCert != nil {
			return defaultCert, nil
		}

		// No certificate available
		return nil, fmt.Errorf("no certificate found for server name: %s", serverName)
	}
}

// ReloadAll reloads all certificates from disk based on database records
// This allows hot-reloading certificates without server restart
func (s *certificateStore) ReloadAll(ctx context.Context) error {
	if s.certRepo == nil {
		return fmt.Errorf("certificate repository not configured")
	}

	// Get all certificates from database
	certs, err := s.certRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to get certificates from database: %w", err)
	}

	// Create new certificate map
	newCerts := make(map[string]*tls.Certificate)

	// Load each certificate from disk
	for _, dbCert := range certs {
		cert, err := s.LoadCertificate(dbCert.DomainName)
		if err != nil {
			// Log error but continue loading other certificates
			continue
		}
		newCerts[dbCert.DomainName] = cert

		// Also add mail subdomain mapping
		mailSubdomain := fmt.Sprintf("mail.%s", dbCert.DomainName)
		newCerts[mailSubdomain] = cert
	}

	// Atomically replace the certificate map
	s.mu.Lock()
	s.certificates = newCerts
	s.mu.Unlock()

	// Notify all registered callbacks
	s.notifyReloadCallbacks()

	return nil
}

// notifyReloadCallbacks calls all registered reload callbacks
func (s *certificateStore) notifyReloadCallbacks() {
	s.mu.RLock()
	callbacks := make([]func(), len(s.reloadCallbacks))
	copy(callbacks, s.reloadCallbacks)
	s.mu.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback()
		}
	}
}

// RegisterReloadCallback registers a callback function to be called after certificates are reloaded
func (s *certificateStore) RegisterReloadCallback(callback func()) {
	if callback == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadCallbacks = append(s.reloadCallbacks, callback)
}

// GetCertificate retrieves a certificate from the in-memory store
func (s *certificateStore) GetCertificate(domainName string) (*tls.Certificate, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cert, ok := s.certificates[domainName]
	if !ok {
		return nil, fmt.Errorf("certificate not found for domain: %s", domainName)
	}

	return cert, nil
}

// SetDefaultCertificate sets the default certificate to use when no SNI match is found
func (s *certificateStore) SetDefaultCertificate(cert *tls.Certificate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultCert = cert
}

// GetDefaultCertificate returns the default certificate
func (s *certificateStore) GetDefaultCertificate() *tls.Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultCert
}

// Count returns the number of certificates in the store
func (s *certificateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.certificates)
}

// extractParentDomain extracts the parent domain from a subdomain
// e.g., "mail.example.com" -> "example.com"
// Returns empty string if no parent domain can be extracted
func extractParentDomain(domain string) string {
	// Find the first dot
	for i := 0; i < len(domain); i++ {
		if domain[i] == '.' {
			// Return everything after the first dot
			if i+1 < len(domain) {
				return domain[i+1:]
			}
			break
		}
	}
	return ""
}

// LoadAndAddCertificate is a convenience method that loads a certificate from disk
// and adds it to the in-memory store in one operation
// This is used for hot reload after a new certificate is generated
func (s *certificateStore) LoadAndAddCertificate(domainName string) error {
	cert, err := s.LoadCertificate(domainName)
	if err != nil {
		return err
	}

	if err := s.AddCertificate(domainName, cert); err != nil {
		return err
	}

	// Also add mail subdomain mapping
	mailSubdomain := fmt.Sprintf("mail.%s", domainName)
	if err := s.AddCertificate(mailSubdomain, cert); err != nil {
		return err
	}

	// Notify all registered callbacks
	s.notifyReloadCallbacks()

	return nil
}
