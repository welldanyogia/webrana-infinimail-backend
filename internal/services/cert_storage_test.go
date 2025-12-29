package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sample PEM data for testing
var testCertPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpegPjMCMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yMzAxMDEwMDAwMDBaFw0yNDAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96HtiXjnpL5GR8xJ9jH
mVzLpHGg5WMPfLVf8nnLiUso7xHGz7qRQqPYxqNqFKOGKPqFTfLSMfGYNdGzPfPT
AgMBAAGjUzBRMB0GA1UdDgQWBBQK8So4YPYC7PnfLNC3DLHSF5LkpjAfBgNVHSME
GDAWgBQK8So4YPYC7PnfLNC3DLHSF5LkpjAPBgNVHRMBAf8EBTADAQH/MA0GCSqG
SIb3DQEBCwUAA0EAhHv5zHnvPfELf0s0XT5YgMPe7TEhwpPvF/qFRxeDvFYE8hGH
bcqFIC0HAewvPmVYGwLvUA3yj+BcGHsY9lNvZg==
-----END CERTIFICATE-----`)

var testKeyPEM = []byte(`-----BEGIN EC PRIVATE KEY-----
MHQCAQEEIBYr17jQJdd+8xhgvCi0sP1YPMfXZ+sDU/ODB0vPmpaeoAcGBSuBBAAK
oUQDQgAEjr8gLxu/vkrpFqST9tKYQCqvGb+hFClZFnhzWgTfua5tCnfpvCwLCSxj
pjata8Qffnzqgz8HV00BVp/7Y+WuIA==
-----END EC PRIVATE KEY-----`)

var testChainPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpegPjMDMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yMzAxMDEwMDAwMDBaFw0yNDAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96HtiXjnpL5GR8xJ9jH
mVzLpHGg5WMPfLVf8nnLiUso7xHGz7qRQqPYxqNqFKOGKPqFTfLSMfGYNdGzPfPT
AgMBAAGjUzBRMB0GA1UdDgQWBBQK8So4YPYC7PnfLNC3DLHSF5LkpjAfBgNVHSME
GDAWgBQK8So4YPYC7PnfLNC3DLHSF5LkpjAPBgNVHRMBAf8EBTADAQH/MA0GCSqG
SIb3DQEBCwUAA0EAhHv5zHnvPfELf0s0XT5YgMPe7TEhwpPvF/qFRxeDvFYE8hGH
bcqFIC0HAewvPmVYGwLvUA3yj+BcGHsY9lNvZg==
-----END CERTIFICATE-----`)

func TestNewCertStorage(t *testing.T) {
	tests := []struct {
		name    string
		config  CertStorageConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CertStorageConfig{
				BasePath: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "empty base path",
			config: CertStorageConfig{
				BasePath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewCertStorage(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, storage)
		})
	}
}

func TestCertStorage_SaveCertificate(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	bundle := &CertificateBundle{
		Certificate: testCertPEM,
		PrivateKey:  testKeyPEM,
		Chain:       testChainPEM,
		ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}

	stored, err := storage.SaveCertificate("example.com", bundle)
	require.NoError(t, err)
	assert.NotNil(t, stored)
	assert.Equal(t, "example.com", stored.DomainName)
	assert.NotEmpty(t, stored.CertPath)
	assert.NotEmpty(t, stored.KeyPath)

	// Verify files exist
	assert.FileExists(t, stored.CertPath)
	assert.FileExists(t, stored.KeyPath)

	// Verify files are readable (permissions vary by OS)
	certInfo, err := os.Stat(stored.CertPath)
	require.NoError(t, err)
	assert.True(t, certInfo.Mode().IsRegular())

	keyInfo, err := os.Stat(stored.KeyPath)
	require.NoError(t, err)
	assert.True(t, keyInfo.Mode().IsRegular())
}


func TestCertStorage_SaveCertificate_Errors(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	tests := []struct {
		name       string
		domainName string
		bundle     *CertificateBundle
		wantErr    bool
	}{
		{
			name:       "empty domain name",
			domainName: "",
			bundle: &CertificateBundle{
				Certificate: testCertPEM,
				PrivateKey:  testKeyPEM,
			},
			wantErr: true,
		},
		{
			name:       "nil bundle",
			domainName: "example.com",
			bundle:     nil,
			wantErr:    true,
		},
		{
			name:       "empty certificate",
			domainName: "example.com",
			bundle: &CertificateBundle{
				Certificate: []byte{},
				PrivateKey:  testKeyPEM,
			},
			wantErr: true,
		},
		{
			name:       "empty private key",
			domainName: "example.com",
			bundle: &CertificateBundle{
				Certificate: testCertPEM,
				PrivateKey:  []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.SaveCertificate(tt.domainName, tt.bundle)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCertStorage_LoadCertificate(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	// Save a certificate first
	bundle := &CertificateBundle{
		Certificate: testCertPEM,
		PrivateKey:  testKeyPEM,
		Chain:       testChainPEM,
		ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}

	_, err = storage.SaveCertificate("example.com", bundle)
	require.NoError(t, err)

	// Load the certificate
	loaded, err := storage.LoadCertificate("example.com")
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, testCertPEM, loaded.Certificate)
	assert.Equal(t, testKeyPEM, loaded.PrivateKey)
	assert.Equal(t, testChainPEM, loaded.Chain)
}

func TestCertStorage_LoadCertificate_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	_, err = storage.LoadCertificate("nonexistent.com")
	assert.Error(t, err)
}

func TestCertStorage_DeleteCertificate(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	// Save a certificate first
	bundle := &CertificateBundle{
		Certificate: testCertPEM,
		PrivateKey:  testKeyPEM,
		ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}

	stored, err := storage.SaveCertificate("example.com", bundle)
	require.NoError(t, err)

	// Verify files exist
	assert.FileExists(t, stored.CertPath)
	assert.FileExists(t, stored.KeyPath)

	// Delete the certificate
	err = storage.DeleteCertificate("example.com")
	require.NoError(t, err)

	// Verify files are deleted
	assert.NoFileExists(t, stored.CertPath)
	assert.NoFileExists(t, stored.KeyPath)
}

func TestCertStorage_DeleteCertificate_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	// Should not error when deleting non-existent certificate
	err = storage.DeleteCertificate("nonexistent.com")
	assert.NoError(t, err)
}

func TestCertStorage_CertificateExists(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	// Should not exist initially
	assert.False(t, storage.CertificateExists("example.com"))

	// Save a certificate
	bundle := &CertificateBundle{
		Certificate: testCertPEM,
		PrivateKey:  testKeyPEM,
		ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}

	_, err = storage.SaveCertificate("example.com", bundle)
	require.NoError(t, err)

	// Should exist now
	assert.True(t, storage.CertificateExists("example.com"))

	// Delete and verify
	err = storage.DeleteCertificate("example.com")
	require.NoError(t, err)
	assert.False(t, storage.CertificateExists("example.com"))
}

func TestCertStorage_GetPaths(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	certPath := storage.GetCertificatePath("example.com")
	keyPath := storage.GetKeyPath("example.com")

	assert.Equal(t, filepath.Join(tempDir, "example.com", "fullchain.pem"), certPath)
	assert.Equal(t, filepath.Join(tempDir, "example.com", "privkey.pem"), keyPath)
}

func TestCertStorage_DirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewCertStorage(CertStorageConfig{BasePath: tempDir})
	require.NoError(t, err)

	bundle := &CertificateBundle{
		Certificate: testCertPEM,
		PrivateKey:  testKeyPEM,
		ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}

	_, err = storage.SaveCertificate("example.com", bundle)
	require.NoError(t, err)

	// Verify directory structure: /certs/{domain}/
	domainDir := filepath.Join(tempDir, "example.com")
	assert.DirExists(t, domainDir)

	// Verify domain directory exists and is a directory
	dirInfo, err := os.Stat(domainDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}
