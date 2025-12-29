package services

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
)

// MockDNSResolver is a mock implementation of DNSResolver
type MockDNSResolver struct {
	mock.Mock
}

func (m *MockDNSResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*net.MX), args.Error(1)
}

func (m *MockDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	args := m.Called(ctx, host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDNSResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// Note: MockDomainRepository is defined in domain_manager_test.go

func TestVerifyMXRecord_Success(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{
		{Host: "mail.infinimail.local.", Pref: 10},
	}, nil)

	verified, err := service.VerifyMXRecord(context.Background(), "example.com", "mail.infinimail.local")

	assert.NoError(t, err)
	assert.True(t, verified)
	mockResolver.AssertExpectations(t)
}


func TestVerifyMXRecord_Mismatch(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns different host
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{
		{Host: "other.mail.server.", Pref: 10},
	}, nil)

	verified, err := service.VerifyMXRecord(context.Background(), "example.com", "mail.infinimail.local")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "MX record mismatch")
	mockResolver.AssertExpectations(t)
}

func TestVerifyMXRecord_NoRecords(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns empty slice
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{}, nil)

	verified, err := service.VerifyMXRecord(context.Background(), "example.com", "mail.infinimail.local")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "no MX records found")
	mockResolver.AssertExpectations(t)
}

func TestVerifyMXRecord_LookupError(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns error
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return(nil, errors.New("DNS timeout"))

	verified, err := service.VerifyMXRecord(context.Background(), "example.com", "mail.infinimail.local")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "MX lookup failed")
	mockResolver.AssertExpectations(t)
}

func TestVerifyARecord_Success(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{"192.168.1.1"}, nil)

	verified, err := service.VerifyARecord(context.Background(), "mail.example.com", "192.168.1.1")

	assert.NoError(t, err)
	assert.True(t, verified)
	mockResolver.AssertExpectations(t)
}

func TestVerifyARecord_Mismatch(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns different IP
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{"10.0.0.1"}, nil)

	verified, err := service.VerifyARecord(context.Background(), "mail.example.com", "192.168.1.1")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "A record mismatch")
	mockResolver.AssertExpectations(t)
}

func TestVerifyARecord_NoRecords(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns empty slice
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{}, nil)

	verified, err := service.VerifyARecord(context.Background(), "mail.example.com", "192.168.1.1")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "no A records found")
	mockResolver.AssertExpectations(t)
}

func TestVerifyTXTRecord_Success(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	verified, err := service.VerifyTXTRecord(context.Background(), "example.com", "abc123xyz")

	assert.NoError(t, err)
	assert.True(t, verified)
	mockResolver.AssertExpectations(t)
}

func TestVerifyTXTRecord_Mismatch(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns different token
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{"infinimail-verify=wrongtoken"}, nil)

	verified, err := service.VerifyTXTRecord(context.Background(), "example.com", "abc123xyz")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "TXT record mismatch")
	mockResolver.AssertExpectations(t)
}

func TestVerifyTXTRecord_NoRecords(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Setup mock - returns empty slice
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{}, nil)

	verified, err := service.VerifyTXTRecord(context.Background(), "example.com", "abc123xyz")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "no TXT records found")
	mockResolver.AssertExpectations(t)
}


func TestVerifyDNS_AllSuccess(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	domain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	// Setup mocks for all lookups
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{
		{Host: "mail.infinimail.local.", Pref: 10},
	}, nil)
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{"192.168.1.1"}, nil)
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.MXVerified)
	assert.True(t, result.AVerified)
	assert.True(t, result.TXTVerified)
	assert.True(t, result.AllVerified)
	assert.Empty(t, result.Errors)
	mockResolver.AssertExpectations(t)
}

func TestVerifyDNS_PartialFailure(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	domain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	// Setup mocks - MX succeeds, A fails, TXT succeeds
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{
		{Host: "mail.infinimail.local.", Pref: 10},
	}, nil)
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{"10.0.0.1"}, nil) // Wrong IP
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.MXVerified)
	assert.False(t, result.AVerified)
	assert.True(t, result.TXTVerified)
	assert.False(t, result.AllVerified)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "A record verification failed")
	mockResolver.AssertExpectations(t)
}

func TestVerifyDNS_AllFailure(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	domain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	// Setup mocks - all fail
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return(nil, errors.New("DNS timeout"))
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return(nil, errors.New("DNS timeout"))
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return(nil, errors.New("DNS timeout"))

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.MXVerified)
	assert.False(t, result.AVerified)
	assert.False(t, result.TXTVerified)
	assert.False(t, result.AllVerified)
	assert.Len(t, result.Errors, 3)
	mockResolver.AssertExpectations(t)
}

func TestVerifyDNS_NilDomain(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	result, err := service.VerifyDNS(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "domain cannot be nil")
}

func TestVerifyMXRecord_EmptyDomainName(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyMXRecord(context.Background(), "", "mail.infinimail.local")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "domain name cannot be empty")
}

func TestVerifyMXRecord_EmptyExpectedHost(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyMXRecord(context.Background(), "example.com", "")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "expected host cannot be empty")
}

func TestVerifyARecord_EmptyHostname(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyARecord(context.Background(), "", "192.168.1.1")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "hostname cannot be empty")
}

func TestVerifyARecord_EmptyExpectedIP(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyARecord(context.Background(), "mail.example.com", "")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "expected IP cannot be empty")
}

func TestVerifyTXTRecord_EmptyDomainName(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyTXTRecord(context.Background(), "", "abc123")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "domain name cannot be empty")
}

func TestVerifyTXTRecord_EmptyChallengeToken(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DefaultDNSVerifierConfig()
	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	verified, err := service.VerifyTXTRecord(context.Background(), "example.com", "")

	assert.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "challenge token cannot be empty")
}

func TestVerifyWithRetry_SuccessOnRetry(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    2,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	domain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	// MX: First call fails, second succeeds
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return(nil, errors.New("DNS timeout")).Once()
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return([]*net.MX{
		{Host: "mail.infinimail.local.", Pref: 10},
	}, nil).Once()

	// A: Succeeds on first try
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return([]string{"192.168.1.1"}, nil)

	// TXT: Succeeds on first try
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.MXVerified)
	assert.True(t, result.AVerified)
	assert.True(t, result.TXTVerified)
	assert.True(t, result.AllVerified)
	mockResolver.AssertExpectations(t)
}

func TestVerifyWithRetry_ContextCancellation(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.local",
		ServerIP:      "192.168.1.1",
		MaxRetries:    5,
		RetryDelay:    time.Second, // Long delay
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	domain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	// Setup mock to always fail
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return(nil, errors.New("DNS timeout"))
	mockResolver.On("LookupHost", mock.Anything, "mail.example.com").Return(nil, errors.New("DNS timeout"))
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.example.com").Return(nil, errors.New("DNS timeout"))

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := service.VerifyDNS(ctx, domain)

	// Should return with context error or partial results
	assert.NotNil(t, result)
	assert.False(t, result.AllVerified)
	_ = err // Error may or may not be set depending on timing
}

func TestDefaultDNSVerifierConfig(t *testing.T) {
	config := DefaultDNSVerifierConfig()

	assert.Equal(t, "mail.infinimail.local", config.SMTPHostname)
	assert.Equal(t, "127.0.0.1", config.ServerIP)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
	assert.Equal(t, 10*time.Second, config.LookupTimeout)
}


// Test helper functions for parent domain extraction
func TestGetParentDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain without mail prefix",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "domain with mail prefix",
			input:    "mail.example.com",
			expected: "example.com",
		},
		{
			name:     "domain with MAIL prefix (uppercase)",
			input:    "MAIL.example.com",
			expected: "example.com",
		},
		{
			name:     "domain with Mail prefix (mixed case)",
			input:    "Mail.example.com",
			expected: "example.com",
		},
		{
			name:     "subdomain without mail prefix",
			input:    "sub.example.com",
			expected: "sub.example.com",
		},
		{
			name:     "deep subdomain with mail prefix",
			input:    "mail.sub.example.com",
			expected: "sub.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParentDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMailHostname(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain without mail prefix",
			input:    "example.com",
			expected: "mail.example.com",
		},
		{
			name:     "domain already has mail prefix",
			input:    "mail.example.com",
			expected: "mail.example.com",
		},
		{
			name:     "domain with MAIL prefix (uppercase)",
			input:    "MAIL.example.com",
			expected: "MAIL.example.com",
		},
		{
			name:     "subdomain without mail prefix",
			input:    "sub.example.com",
			expected: "mail.sub.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMailHostname(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test VerifyDNS with domain that has mail. prefix (the bug scenario)
func TestVerifyDNS_DomainWithMailPrefix(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.webrana.id",
		ServerIP:      "103.127.136.43",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Domain stored with mail. prefix (the bug scenario)
	domain := &models.Domain{
		ID:           1,
		Name:         "mail.agungbesisentosa.com", // This is how it was stored
		DNSChallenge: "abc123xyz",
	}

	// The fix should:
	// - MX lookup on "agungbesisentosa.com" (parent domain)
	// - A record lookup on "mail.agungbesisentosa.com" (no double prefix)
	// - TXT lookup on "_infinimail.agungbesisentosa.com" (parent domain)
	mockResolver.On("LookupMX", mock.Anything, "agungbesisentosa.com").Return([]*net.MX{
		{Host: "mail.infinimail.webrana.id.", Pref: 10},
	}, nil)
	mockResolver.On("LookupHost", mock.Anything, "mail.agungbesisentosa.com").Return([]string{"103.127.136.43"}, nil)
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.agungbesisentosa.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.MXVerified, "MX should be verified on parent domain")
	assert.True(t, result.AVerified, "A record should be verified without double mail. prefix")
	assert.True(t, result.TXTVerified, "TXT should be verified on parent domain")
	assert.True(t, result.AllVerified)
	assert.Empty(t, result.Errors)
	mockResolver.AssertExpectations(t)
}

// Test VerifyDNS with normal domain (no mail. prefix)
func TestVerifyDNS_DomainWithoutMailPrefix(t *testing.T) {
	mockResolver := new(MockDNSResolver)
	mockRepo := new(MockDomainRepository)

	config := DNSVerifierConfig{
		SMTPHostname:  "mail.infinimail.webrana.id",
		ServerIP:      "103.127.136.43",
		MaxRetries:    0,
		RetryDelay:    time.Millisecond,
		LookupTimeout: time.Second,
	}

	service := NewDNSVerifierServiceWithResolver(mockRepo, config, mockResolver)

	// Domain stored without mail. prefix (normal case)
	domain := &models.Domain{
		ID:           1,
		Name:         "agungbesisentosa.com",
		DNSChallenge: "abc123xyz",
	}

	// Should work as before:
	// - MX lookup on "agungbesisentosa.com"
	// - A record lookup on "mail.agungbesisentosa.com"
	// - TXT lookup on "_infinimail.agungbesisentosa.com"
	mockResolver.On("LookupMX", mock.Anything, "agungbesisentosa.com").Return([]*net.MX{
		{Host: "mail.infinimail.webrana.id.", Pref: 10},
	}, nil)
	mockResolver.On("LookupHost", mock.Anything, "mail.agungbesisentosa.com").Return([]string{"103.127.136.43"}, nil)
	mockResolver.On("LookupTXT", mock.Anything, "_infinimail.agungbesisentosa.com").Return([]string{"infinimail-verify=abc123xyz"}, nil)

	result, err := service.VerifyDNS(context.Background(), domain)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.MXVerified)
	assert.True(t, result.AVerified)
	assert.True(t, result.TXTVerified)
	assert.True(t, result.AllVerified)
	assert.Empty(t, result.Errors)
	mockResolver.AssertExpectations(t)
}
