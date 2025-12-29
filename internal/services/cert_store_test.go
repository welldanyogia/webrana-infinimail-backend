package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertificate creates a self-signed certificate for testing
func generateTestCertificate(domain string) (*tls.Certificate, []byte, []byte, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   domain,
		},
		DNSNames:              []string{domain, "mail." + domain},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	// Create tls.Certificate
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, nil, err
	}

	return &tlsCert, certPEM, keyPEM, nil
}

func TestNewCertificateStore(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	t.Run("creates store with valid config", func(t *testing.T) {
		store, err := NewCertificateStore(CertificateStoreConfig{
			CertStorage: certStorage,
		})
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("returns error with nil cert storage", func(t *testing.T) {
		store, err := NewCertificateStore(CertificateStoreConfig{
			CertStorage: nil,
		})
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}


func TestCertificateStore_AddAndGetCertificate(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	// Generate test certificate
	cert, _, _, err := generateTestCertificate("example.com")
	require.NoError(t, err)

	t.Run("adds certificate successfully", func(t *testing.T) {
		err := store.AddCertificate("example.com", cert)
		assert.NoError(t, err)
	})

	t.Run("retrieves added certificate", func(t *testing.T) {
		retrieved, err := store.GetCertificate("example.com")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})

	t.Run("returns error for non-existent certificate", func(t *testing.T) {
		_, err := store.GetCertificate("nonexistent.com")
		assert.Error(t, err)
	})

	t.Run("returns error for empty domain name", func(t *testing.T) {
		err := store.AddCertificate("", cert)
		assert.Error(t, err)
	})

	t.Run("returns error for nil certificate", func(t *testing.T) {
		err := store.AddCertificate("test.com", nil)
		assert.Error(t, err)
	})
}

func TestCertificateStore_RemoveCertificate(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	// Generate and add test certificate
	cert, _, _, err := generateTestCertificate("example.com")
	require.NoError(t, err)
	require.NoError(t, store.AddCertificate("example.com", cert))

	t.Run("removes certificate successfully", func(t *testing.T) {
		err := store.RemoveCertificate("example.com")
		assert.NoError(t, err)

		// Verify certificate is removed
		_, err = store.GetCertificate("example.com")
		assert.Error(t, err)
	})

	t.Run("returns error for empty domain name", func(t *testing.T) {
		err := store.RemoveCertificate("")
		assert.Error(t, err)
	})
}

func TestCertificateStore_LoadCertificate(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	// Generate test certificate
	_, certPEM, keyPEM, err := generateTestCertificate("loadtest.com")
	require.NoError(t, err)

	// Save certificate to disk
	domainDir := filepath.Join(tempDir, "loadtest.com")
	require.NoError(t, os.MkdirAll(domainDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "fullchain.pem"), certPEM, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "privkey.pem"), keyPEM, 0600))

	t.Run("loads certificate from disk", func(t *testing.T) {
		cert, err := store.LoadCertificate("loadtest.com")
		assert.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("returns error for non-existent certificate", func(t *testing.T) {
		_, err := store.LoadCertificate("nonexistent.com")
		assert.Error(t, err)
	})

	t.Run("returns error for empty domain name", func(t *testing.T) {
		_, err := store.LoadCertificate("")
		assert.Error(t, err)
	})
}

func TestCertificateStore_GetCertificateFunc(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	// Generate test certificates
	cert1, _, _, err := generateTestCertificate("example.com")
	require.NoError(t, err)
	cert2, _, _, err := generateTestCertificate("another.com")
	require.NoError(t, err)
	defaultCert, _, _, err := generateTestCertificate("default.com")
	require.NoError(t, err)

	// Add certificates
	require.NoError(t, store.AddCertificate("example.com", cert1))
	require.NoError(t, store.AddCertificate("another.com", cert2))
	store.SetDefaultCertificate(defaultCert)

	getCertFunc := store.GetCertificateFunc()

	t.Run("returns correct certificate for exact domain match", func(t *testing.T) {
		hello := &tls.ClientHelloInfo{ServerName: "example.com"}
		cert, err := getCertFunc(hello)
		assert.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("returns parent domain certificate for subdomain", func(t *testing.T) {
		// Add certificate for parent domain
		require.NoError(t, store.AddCertificate("parent.com", cert1))

		hello := &tls.ClientHelloInfo{ServerName: "mail.parent.com"}
		cert, err := getCertFunc(hello)
		assert.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("returns default certificate when no match found", func(t *testing.T) {
		hello := &tls.ClientHelloInfo{ServerName: "unknown.com"}
		cert, err := getCertFunc(hello)
		assert.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("returns error when no certificate and no default", func(t *testing.T) {
		// Create store without default certificate
		store2, err := NewCertificateStore(CertificateStoreConfig{
			CertStorage: certStorage,
		})
		require.NoError(t, err)

		getCertFunc2 := store2.GetCertificateFunc()
		hello := &tls.ClientHelloInfo{ServerName: "unknown.com"}
		_, err = getCertFunc2(hello)
		assert.Error(t, err)
	})
}

func TestCertificateStore_DefaultCertificate(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	t.Run("initially has no default certificate", func(t *testing.T) {
		cert := store.GetDefaultCertificate()
		assert.Nil(t, cert)
	})

	t.Run("sets and gets default certificate", func(t *testing.T) {
		defaultCert, _, _, err := generateTestCertificate("default.com")
		require.NoError(t, err)

		store.SetDefaultCertificate(defaultCert)

		retrieved := store.GetDefaultCertificate()
		assert.NotNil(t, retrieved)
	})
}

func TestCertificateStore_Count(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
	})
	require.NoError(t, err)

	t.Run("initially has zero certificates", func(t *testing.T) {
		assert.Equal(t, 0, store.Count())
	})

	t.Run("count increases when certificates are added", func(t *testing.T) {
		cert, _, _, err := generateTestCertificate("count1.com")
		require.NoError(t, err)
		require.NoError(t, store.AddCertificate("count1.com", cert))

		assert.Equal(t, 1, store.Count())

		cert2, _, _, err := generateTestCertificate("count2.com")
		require.NoError(t, err)
		require.NoError(t, store.AddCertificate("count2.com", cert2))

		assert.Equal(t, 2, store.Count())
	})

	t.Run("count decreases when certificates are removed", func(t *testing.T) {
		require.NoError(t, store.RemoveCertificate("count1.com"))
		assert.Equal(t, 1, store.Count())
	})
}

func TestExtractParentDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mail.example.com", "example.com"},
		{"sub.domain.example.com", "domain.example.com"},
		{"example.com", "com"},
		{"localhost", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractParentDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCertificateStore_ReloadAll_WithoutRepo(t *testing.T) {
	tempDir := t.TempDir()

	certStorage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	// Create store without certificate repository
	store, err := NewCertificateStore(CertificateStoreConfig{
		CertStorage: certStorage,
		CertRepo:    nil,
	})
	require.NoError(t, err)

	t.Run("returns error when repository not configured", func(t *testing.T) {
		err := store.ReloadAll(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "certificate repository not configured")
	})
}
