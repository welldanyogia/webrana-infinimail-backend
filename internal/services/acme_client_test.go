package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewACMEClient(t *testing.T) {
	tests := []struct {
		name    string
		config  ACMEClientConfig
		wantErr bool
	}{
		{
			name: "staging environment",
			config: ACMEClientConfig{
				Email:   "test@example.com",
				Staging: true,
			},
			wantErr: false,
		},
		{
			name: "production environment",
			config: ACMEClientConfig{
				Email:   "test@example.com",
				Staging: false,
			},
			wantErr: false,
		},
		{
			name: "custom directory URL",
			config: ACMEClientConfig{
				DirectoryURL: "https://custom-acme.example.com/directory",
				Email:        "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "empty email",
			config: ACMEClientConfig{
				Staging: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewACMEClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.NotNil(t, client.GetAccountKey())
		})
	}
}

func TestNewACMEClientWithKey(t *testing.T) {
	// Generate a test key
	testKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	config := ACMEClientConfig{
		Email:   "test@example.com",
		Staging: true,
	}

	client, err := NewACMEClientWithKey(config, testKey)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify the key is the same
	returnedKey := client.GetAccountKey()
	assert.Equal(t, testKey, returnedKey)
}

func TestACMEDirectoryURLConstants(t *testing.T) {
	assert.Equal(t, "https://acme-v02.api.letsencrypt.org/directory", LetsEncryptProduction)
	assert.Equal(t, "https://acme-staging-v02.api.letsencrypt.org/directory", LetsEncryptStaging)
}

func TestCreateCSR(t *testing.T) {
	// Generate a test key
	testKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name    string
		domains []string
		wantErr bool
	}{
		{
			name:    "single domain",
			domains: []string{"example.com"},
			wantErr: false,
		},
		{
			name:    "multiple domains (SAN)",
			domains: []string{"example.com", "mail.example.com", "www.example.com"},
			wantErr: false,
		},
		{
			name:    "empty domains",
			domains: []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csr, err := createCSR(testKey, tt.domains)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, csr)
		})
	}
}

func TestCertificateBundleFields(t *testing.T) {
	bundle := &CertificateBundle{
		Certificate: []byte("cert-data"),
		PrivateKey:  []byte("key-data"),
		Chain:       []byte("chain-data"),
	}

	assert.Equal(t, []byte("cert-data"), bundle.Certificate)
	assert.Equal(t, []byte("key-data"), bundle.PrivateKey)
	assert.Equal(t, []byte("chain-data"), bundle.Chain)
}

func TestDNSChallengeInfoFields(t *testing.T) {
	info := &DNSChallengeInfo{
		Domain:    "example.com",
		Token:     "test-token",
		KeyAuth:   "key-auth-value",
		TXTRecord: "txt-record-value",
	}

	assert.Equal(t, "example.com", info.Domain)
	assert.Equal(t, "test-token", info.Token)
	assert.Equal(t, "key-auth-value", info.KeyAuth)
	assert.Equal(t, "txt-record-value", info.TXTRecord)
}
