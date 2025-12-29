package services

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CertStorageConfig holds configuration for certificate storage
type CertStorageConfig struct {
	BasePath string // Base directory for certificate storage (e.g., /certs)
}

// StoredCertificate contains metadata about a stored certificate
type StoredCertificate struct {
	DomainName string    `json:"domain_name"`
	CertPath   string    `json:"cert_path"`
	KeyPath    string    `json:"key_path"`
	ChainPath  string    `json:"chain_path,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
	IssuedAt   time.Time `json:"issued_at"`
}

// CertStorage defines the interface for certificate storage operations
type CertStorage interface {
	// SaveCertificate saves a certificate bundle to disk
	SaveCertificate(domainName string, bundle *CertificateBundle) (*StoredCertificate, error)

	// LoadCertificate loads certificate and key from disk
	LoadCertificate(domainName string) (*CertificateBundle, error)

	// DeleteCertificate removes certificate files from disk
	DeleteCertificate(domainName string) error

	// CertificateExists checks if certificate files exist for a domain
	CertificateExists(domainName string) bool

	// GetCertificatePath returns the path to the certificate file
	GetCertificatePath(domainName string) string

	// GetKeyPath returns the path to the private key file
	GetKeyPath(domainName string) string
}

// certStorage implements CertStorage interface
type certStorage struct {
	config CertStorageConfig
}

// NewCertStorage creates a new certificate storage instance
func NewCertStorage(config CertStorageConfig) (CertStorage, error) {
	if config.BasePath == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	// Ensure base directory exists
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &certStorage{
		config: config,
	}, nil
}


// SaveCertificate saves a certificate bundle to disk with restricted permissions
func (s *certStorage) SaveCertificate(domainName string, bundle *CertificateBundle) (*StoredCertificate, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}
	if bundle == nil {
		return nil, fmt.Errorf("certificate bundle cannot be nil")
	}
	if len(bundle.Certificate) == 0 {
		return nil, fmt.Errorf("certificate data cannot be empty")
	}
	if len(bundle.PrivateKey) == 0 {
		return nil, fmt.Errorf("private key data cannot be empty")
	}

	// Create domain directory: /certs/{domain}/
	domainDir := filepath.Join(s.config.BasePath, domainName)
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create domain directory: %w", err)
	}

	// Define file paths
	certPath := filepath.Join(domainDir, "fullchain.pem")
	keyPath := filepath.Join(domainDir, "privkey.pem")

	// Write certificate file with restricted permissions (0600)
	if err := writeFileSecure(certPath, bundle.Certificate); err != nil {
		return nil, fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key file with restricted permissions (0600)
	if err := writeFileSecure(keyPath, bundle.PrivateKey); err != nil {
		// Clean up certificate file on failure
		os.Remove(certPath)
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	// Write chain file if present
	var chainPath string
	if len(bundle.Chain) > 0 {
		chainPath = filepath.Join(domainDir, "chain.pem")
		if err := writeFileSecure(chainPath, bundle.Chain); err != nil {
			// Clean up on failure
			os.Remove(certPath)
			os.Remove(keyPath)
			return nil, fmt.Errorf("failed to write chain: %w", err)
		}
	}

	// Parse certificate to get issued date
	issuedAt := time.Now()
	if block, _ := pem.Decode(bundle.Certificate); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			issuedAt = cert.NotBefore
		}
	}

	return &StoredCertificate{
		DomainName: domainName,
		CertPath:   certPath,
		KeyPath:    keyPath,
		ChainPath:  chainPath,
		ExpiresAt:  bundle.ExpiresAt,
		IssuedAt:   issuedAt,
	}, nil
}

// LoadCertificate loads certificate and key from disk
func (s *certStorage) LoadCertificate(domainName string) (*CertificateBundle, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	certPath := s.GetCertificatePath(domainName)
	keyPath := s.GetKeyPath(domainName)

	// Read certificate
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	// Read private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Read chain if exists
	chainPath := filepath.Join(s.config.BasePath, domainName, "chain.pem")
	var chainData []byte
	if _, err := os.Stat(chainPath); err == nil {
		chainData, _ = os.ReadFile(chainPath)
	}

	// Parse certificate to get expiry
	var expiresAt time.Time
	if block, _ := pem.Decode(certData); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiresAt = cert.NotAfter
		}
	}

	return &CertificateBundle{
		Certificate: certData,
		PrivateKey:  keyData,
		Chain:       chainData,
		ExpiresAt:   expiresAt,
	}, nil
}

// DeleteCertificate removes certificate files from disk
func (s *certStorage) DeleteCertificate(domainName string) error {
	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	domainDir := filepath.Join(s.config.BasePath, domainName)

	// Check if directory exists
	if _, err := os.Stat(domainDir); os.IsNotExist(err) {
		return nil // Already deleted
	}

	// Remove entire domain directory
	if err := os.RemoveAll(domainDir); err != nil {
		return fmt.Errorf("failed to delete certificate directory: %w", err)
	}

	return nil
}

// CertificateExists checks if certificate files exist for a domain
func (s *certStorage) CertificateExists(domainName string) bool {
	certPath := s.GetCertificatePath(domainName)
	keyPath := s.GetKeyPath(domainName)

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

// GetCertificatePath returns the path to the certificate file
func (s *certStorage) GetCertificatePath(domainName string) string {
	return filepath.Join(s.config.BasePath, domainName, "fullchain.pem")
}

// GetKeyPath returns the path to the private key file
func (s *certStorage) GetKeyPath(domainName string) string {
	return filepath.Join(s.config.BasePath, domainName, "privkey.pem")
}

// writeFileSecure writes data to a file with restricted permissions (0600)
func writeFileSecure(path string, data []byte) error {
	// Write file with restricted permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	// Ensure permissions are set correctly (in case umask affected them)
	return os.Chmod(path, 0600)
}
