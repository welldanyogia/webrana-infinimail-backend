package services

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
)

// mockCertManagerForRenewal is a mock implementation for testing renewal service
type mockCertManagerForRenewal struct {
	mu                    sync.Mutex
	expiringCerts         []Certificate
	renewedDomainIDs      []uint
	getExpiringError      error
	renewError            error
	getExpiringCallCount  int
	renewCallCount        int
}

func (m *mockCertManagerForRenewal) GenerateCertificate(ctx context.Context, domain *models.Domain) (*Certificate, error) {
	return nil, nil
}

func (m *mockCertManagerForRenewal) GetCertificate(ctx context.Context, domainName string) (*Certificate, error) {
	return nil, nil
}

func (m *mockCertManagerForRenewal) GetCertificateByDomainID(ctx context.Context, domainID uint) (*Certificate, error) {
	return nil, nil
}

func (m *mockCertManagerForRenewal) RenewCertificate(ctx context.Context, domainID uint) (*Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renewCallCount++
	if m.renewError != nil {
		return nil, m.renewError
	}
	m.renewedDomainIDs = append(m.renewedDomainIDs, domainID)
	return &Certificate{DomainID: domainID}, nil
}

func (m *mockCertManagerForRenewal) GetExpiringCertificates(ctx context.Context, days int) ([]Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getExpiringCallCount++
	if m.getExpiringError != nil {
		return nil, m.getExpiringError
	}
	return m.expiringCerts, nil
}

func (m *mockCertManagerForRenewal) DeleteCertificate(ctx context.Context, domainID uint) error {
	return nil
}

func (m *mockCertManagerForRenewal) SetAutoRenew(ctx context.Context, domainID uint, autoRenew bool) error {
	return nil
}

func (m *mockCertManagerForRenewal) SetCertificateStore(certStore CertificateStore) {}

func (m *mockCertManagerForRenewal) getRenewedDomainIDs() []uint {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]uint, len(m.renewedDomainIDs))
	copy(result, m.renewedDomainIDs)
	return result
}

func (m *mockCertManagerForRenewal) getCallCounts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getExpiringCallCount, m.renewCallCount
}

func TestNewCertRenewalService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockManager := &mockCertManagerForRenewal{}

	t.Run("creates service with default config", func(t *testing.T) {
		service := NewCertRenewalService(mockManager, CertRenewalConfig{}, logger)
		if service == nil {
			t.Fatal("expected service to be created")
		}
		if service.config.CheckInterval != 24*time.Hour {
			t.Errorf("expected default check interval 24h, got %v", service.config.CheckInterval)
		}
		if service.config.RenewalDays != 30 {
			t.Errorf("expected default renewal days 30, got %d", service.config.RenewalDays)
		}
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		config := CertRenewalConfig{
			CheckInterval: 1 * time.Hour,
			RenewalDays:   14,
		}
		service := NewCertRenewalService(mockManager, config, logger)
		if service.config.CheckInterval != 1*time.Hour {
			t.Errorf("expected check interval 1h, got %v", service.config.CheckInterval)
		}
		if service.config.RenewalDays != 14 {
			t.Errorf("expected renewal days 14, got %d", service.config.RenewalDays)
		}
	})
}

func TestCertRenewalService_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockManager := &mockCertManagerForRenewal{}

	config := CertRenewalConfig{
		CheckInterval: 100 * time.Millisecond,
		RenewalDays:   30,
	}
	service := NewCertRenewalService(mockManager, config, logger)

	t.Run("starts and stops correctly", func(t *testing.T) {
		if service.IsRunning() {
			t.Error("service should not be running initially")
		}

		service.Start()
		if !service.IsRunning() {
			t.Error("service should be running after Start()")
		}

		// Wait a bit for the initial check
		time.Sleep(50 * time.Millisecond)

		service.Stop()
		if service.IsRunning() {
			t.Error("service should not be running after Stop()")
		}
	})

	t.Run("multiple starts are idempotent", func(t *testing.T) {
		service.Start()
		service.Start() // Should not panic or cause issues
		if !service.IsRunning() {
			t.Error("service should be running")
		}
		service.Stop()
	})

	t.Run("multiple stops are idempotent", func(t *testing.T) {
		service.Start()
		service.Stop()
		service.Stop() // Should not panic or cause issues
		if service.IsRunning() {
			t.Error("service should not be running")
		}
	})
}

func TestCertRenewalService_RenewsCertificates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("renews expiring certificates with auto-renew enabled", func(t *testing.T) {
		mockManager := &mockCertManagerForRenewal{
			expiringCerts: []Certificate{
				{DomainID: 1, DomainName: "example1.com", AutoRenew: true, ExpiresAt: time.Now().Add(15 * 24 * time.Hour)},
				{DomainID: 2, DomainName: "example2.com", AutoRenew: true, ExpiresAt: time.Now().Add(20 * 24 * time.Hour)},
			},
		}

		config := CertRenewalConfig{
			CheckInterval: 10 * time.Hour, // Long interval to prevent multiple checks
			RenewalDays:   30,
		}
		service := NewCertRenewalService(mockManager, config, logger)
		service.Start()

		// Wait for initial check to complete
		time.Sleep(50 * time.Millisecond)
		service.Stop()

		renewedIDs := mockManager.getRenewedDomainIDs()
		if len(renewedIDs) != 2 {
			t.Errorf("expected 2 certificates to be renewed, got %d", len(renewedIDs))
		}
	})

	t.Run("skips certificates with auto-renew disabled", func(t *testing.T) {
		mockManager := &mockCertManagerForRenewal{
			expiringCerts: []Certificate{
				{DomainID: 1, DomainName: "example1.com", AutoRenew: true, ExpiresAt: time.Now().Add(15 * 24 * time.Hour)},
				{DomainID: 2, DomainName: "example2.com", AutoRenew: false, ExpiresAt: time.Now().Add(20 * 24 * time.Hour)},
			},
		}

		config := CertRenewalConfig{
			CheckInterval: 10 * time.Hour, // Long interval to prevent multiple checks
			RenewalDays:   30,
		}
		service := NewCertRenewalService(mockManager, config, logger)
		service.Start()

		// Wait for initial check to complete
		time.Sleep(50 * time.Millisecond)
		service.Stop()

		renewedIDs := mockManager.getRenewedDomainIDs()
		if len(renewedIDs) != 1 {
			t.Errorf("expected 1 certificate to be renewed, got %d", len(renewedIDs))
		}
		if len(renewedIDs) > 0 && renewedIDs[0] != 1 {
			t.Errorf("expected domain ID 1 to be renewed, got %d", renewedIDs[0])
		}
	})

	t.Run("continues on renewal error", func(t *testing.T) {
		mockManager := &mockCertManagerForRenewal{
			expiringCerts: []Certificate{
				{DomainID: 1, DomainName: "example1.com", AutoRenew: true},
				{DomainID: 2, DomainName: "example2.com", AutoRenew: true},
			},
			renewError: context.DeadlineExceeded,
		}

		config := CertRenewalConfig{
			CheckInterval: 10 * time.Hour, // Long interval to prevent multiple checks
			RenewalDays:   30,
		}
		service := NewCertRenewalService(mockManager, config, logger)
		service.Start()

		// Wait for initial check to complete
		time.Sleep(50 * time.Millisecond)
		service.Stop()

		// Should have attempted to renew both certificates despite errors
		_, renewCount := mockManager.getCallCounts()
		if renewCount != 2 {
			t.Errorf("expected 2 renewal attempts, got %d", renewCount)
		}
	})
}

func TestCertRenewalService_ForceCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("force check triggers immediate check", func(t *testing.T) {
		mockManager := &mockCertManagerForRenewal{
			expiringCerts: []Certificate{
				{DomainID: 1, DomainName: "example.com", AutoRenew: true},
			},
		}

		config := CertRenewalConfig{
			CheckInterval: 1 * time.Hour, // Long interval
			RenewalDays:   30,
		}
		service := NewCertRenewalService(mockManager, config, logger)
		service.Start()

		// Wait for initial check
		time.Sleep(50 * time.Millisecond)

		// Get initial call count
		initialCount, _ := mockManager.getCallCounts()

		// Force check
		service.ForceCheck()
		time.Sleep(50 * time.Millisecond)

		// Should have additional call
		finalCount, _ := mockManager.getCallCounts()
		if finalCount <= initialCount {
			t.Errorf("expected additional check after ForceCheck, initial: %d, final: %d", initialCount, finalCount)
		}

		service.Stop()
	})

	t.Run("force check does nothing when service not running", func(t *testing.T) {
		mockManager := &mockCertManagerForRenewal{}

		config := CertRenewalConfig{
			CheckInterval: 1 * time.Hour,
			RenewalDays:   30,
		}
		service := NewCertRenewalService(mockManager, config, logger)

		// Don't start the service
		service.ForceCheck() // Should not panic

		count, _ := mockManager.getCallCounts()
		if count != 0 {
			t.Errorf("expected no checks when service not running, got %d", count)
		}
	})
}
