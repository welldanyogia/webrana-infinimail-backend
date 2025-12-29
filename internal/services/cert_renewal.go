package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// CertRenewalConfig holds configuration for the certificate renewal service
type CertRenewalConfig struct {
	// CheckInterval is how often to check for expiring certificates
	CheckInterval time.Duration
	// RenewalDays is the number of days before expiry to trigger renewal
	RenewalDays int
}

// CertRenewalService handles automatic certificate renewal
type CertRenewalService struct {
	certManager CertificateManagerService
	config      CertRenewalConfig
	logger      *slog.Logger
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewCertRenewalService creates a new certificate renewal service
func NewCertRenewalService(
	certManager CertificateManagerService,
	config CertRenewalConfig,
	logger *slog.Logger,
) *CertRenewalService {
	// Set defaults
	if config.CheckInterval <= 0 {
		config.CheckInterval = 24 * time.Hour
	}
	if config.RenewalDays <= 0 {
		config.RenewalDays = 30
	}

	return &CertRenewalService{
		certManager: certManager,
		config:      config,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the certificate renewal background job
func (s *CertRenewalService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.renewalLoop()

	s.logger.Info("certificate renewal service started",
		slog.Duration("check_interval", s.config.CheckInterval),
		slog.Int("renewal_days", s.config.RenewalDays))
}

// Stop gracefully stops the certificate renewal background job
func (s *CertRenewalService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("certificate renewal service stopped")
}

// IsRunning returns whether the renewal service is currently running
func (s *CertRenewalService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// renewalLoop is the main loop that periodically checks and renews certificates
func (s *CertRenewalService) renewalLoop() {
	defer s.wg.Done()

	// Run immediately on start
	s.checkAndRenewCertificates()

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRenewCertificates()
		}
	}
}

// checkAndRenewCertificates checks for expiring certificates and renews them
func (s *CertRenewalService) checkAndRenewCertificates() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.logger.Debug("checking for expiring certificates",
		slog.Int("renewal_days", s.config.RenewalDays))

	// Get certificates expiring within the configured days
	expiringCerts, err := s.certManager.GetExpiringCertificates(ctx, s.config.RenewalDays)
	if err != nil {
		s.logger.Error("failed to get expiring certificates",
			slog.Any("error", err))
		return
	}

	if len(expiringCerts) == 0 {
		s.logger.Debug("no certificates need renewal")
		return
	}

	s.logger.Info("found certificates needing renewal",
		slog.Int("count", len(expiringCerts)))

	// Renew each expiring certificate
	for _, cert := range expiringCerts {
		// Skip if auto-renew is disabled
		if !cert.AutoRenew {
			s.logger.Info("skipping certificate with auto-renew disabled",
				slog.String("domain", cert.DomainName),
				slog.Uint64("domain_id", uint64(cert.DomainID)))
			continue
		}

		s.renewCertificate(ctx, cert)
	}
}

// renewCertificate attempts to renew a single certificate
func (s *CertRenewalService) renewCertificate(ctx context.Context, cert Certificate) {
	s.logger.Info("attempting to renew certificate",
		slog.String("domain", cert.DomainName),
		slog.Uint64("domain_id", uint64(cert.DomainID)),
		slog.Time("expires_at", cert.ExpiresAt))

	_, err := s.certManager.RenewCertificate(ctx, cert.DomainID)
	if err != nil {
		// Log error and continue - don't stop the renewal loop
		s.logger.Error("failed to renew certificate",
			slog.String("domain", cert.DomainName),
			slog.Uint64("domain_id", uint64(cert.DomainID)),
			slog.Any("error", err))
		return
	}

	s.logger.Info("certificate renewed successfully",
		slog.String("domain", cert.DomainName),
		slog.Uint64("domain_id", uint64(cert.DomainID)))
}

// ForceCheck triggers an immediate check for expiring certificates
// This is useful for testing or manual intervention
func (s *CertRenewalService) ForceCheck() {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if !running {
		s.logger.Warn("force check called but service is not running")
		return
	}

	s.logger.Info("force check triggered")
	go s.checkAndRenewCertificates()
}
