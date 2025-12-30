package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/services"
)

// DomainHandler handles domain-related HTTP requests
type DomainHandler struct {
	repo           repository.DomainRepository
	domainManager  services.DomainManagerService
	dnsVerifier    services.DNSVerifierService
	dnsExporter    services.DNSExporter
	certManager    services.CertificateManagerService
}

// NewDomainHandler creates a new DomainHandler
func NewDomainHandler(repo repository.DomainRepository) *DomainHandler {
	return &DomainHandler{repo: repo}
}

// NewDomainHandlerWithServices creates a new DomainHandler with all services for SSL domain setup
func NewDomainHandlerWithServices(
	repo repository.DomainRepository,
	domainManager services.DomainManagerService,
	dnsVerifier services.DNSVerifierService,
	dnsExporter services.DNSExporter,
	certManager services.CertificateManagerService,
) *DomainHandler {
	return &DomainHandler{
		repo:          repo,
		domainManager: domainManager,
		dnsVerifier:   dnsVerifier,
		dnsExporter:   dnsExporter,
		certManager:   certManager,
	}
}

// CreateDomainRequest represents the request body for creating a domain
type CreateDomainRequest struct {
	Name     string `json:"name" validate:"required"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// UpdateDomainRequest represents the request body for updating a domain
type UpdateDomainRequest struct {
	Name     string `json:"name,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// Create handles POST /api/domains
// If domainManager is configured, creates domain with pending_dns status and DNS challenge token
// Otherwise, creates domain with active status (legacy behavior)
func (h *DomainHandler) Create(c echo.Context) error {
	var req CreateDomainRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	// If domain manager is configured, use the new SSL setup flow
	if h.domainManager != nil {
		domain, err := h.domainManager.CreateDomain(c.Request().Context(), req.Name)
		if err != nil {
			if errors.Is(err, repository.ErrDuplicateEntry) {
				return response.Conflict(c, "domain already exists")
			}
			return response.InternalError(c, "failed to create domain")
		}
		return response.Created(c, domain)
	}

	// Legacy behavior: create domain directly with active status
	domain := &models.Domain{
		Name:     req.Name,
		IsActive: true,
	}
	if req.IsActive != nil {
		domain.IsActive = *req.IsActive
	}

	if err := h.repo.Create(c.Request().Context(), domain); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			return response.Conflict(c, "domain already exists")
		}
		return response.InternalError(c, "failed to create domain")
	}

	return response.Created(c, domain)
}

// List handles GET /api/domains
func (h *DomainHandler) List(c echo.Context) error {
	activeOnly := c.QueryParam("active_only") == "true"

	domains, err := h.repo.List(c.Request().Context(), activeOnly)
	if err != nil {
		return response.InternalError(c, "failed to list domains")
	}

	return response.Success(c, domains)
}

// Get handles GET /api/domains/:id
func (h *DomainHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	return response.Success(c, domain)
}

// Update handles PUT /api/domains/:id
func (h *DomainHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	var req UpdateDomainRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	// Get existing domain
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Update fields
	if req.Name != "" {
		domain.Name = req.Name
	}
	if req.IsActive != nil {
		domain.IsActive = *req.IsActive
	}

	if err := h.repo.Update(c.Request().Context(), domain); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			return response.Conflict(c, "domain name already exists")
		}
		return response.InternalError(c, "failed to update domain")
	}

	return response.Success(c, domain)
}

// Delete handles DELETE /api/domains/:id
func (h *DomainHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	if err := h.repo.Delete(c.Request().Context(), uint(id)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to delete domain")
	}

	return response.NoContent(c)
}


// GetDNSGuide handles GET /api/domains/:id/dns-guide
// Returns DNS records to configure for domain setup
func (h *DomainHandler) GetDNSGuide(c echo.Context) error {
	if h.domainManager == nil {
		return response.InternalError(c, "domain manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	guide, err := h.domainManager.GetDNSGuide(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get DNS guide")
	}

	return response.Success(c, guide)
}

// GetDNSExport handles GET /api/domains/:id/dns-export?format={bind|cloudflare|route53|csv}
// Returns exported DNS records in the requested format
func (h *DomainHandler) GetDNSExport(c echo.Context) error {
	if h.domainManager == nil || h.dnsExporter == nil {
		return response.InternalError(c, "domain manager or DNS exporter not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get format from query parameter
	formatStr := c.QueryParam("format")
	if formatStr == "" {
		formatStr = "bind" // Default to BIND format
	}

	format := services.DNSExportFormat(formatStr)
	if !format.IsValid() {
		return response.BadRequest(c, "invalid export format, must be one of: bind, cloudflare, route53, csv")
	}

	// Get domain to get the name
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Get DNS guide
	guide, err := h.domainManager.GetDNSGuide(c.Request().Context(), uint(id))
	if err != nil {
		return response.InternalError(c, "failed to get DNS guide")
	}

	// Export in requested format
	result, err := h.dnsExporter.Export(guide, domain.Name, format)
	if err != nil {
		return response.InternalError(c, "failed to export DNS records")
	}

	return response.Success(c, result)
}

// VerifyDNS handles POST /api/domains/:id/verify-dns
// Verifies DNS records and updates domain status on success
func (h *DomainHandler) VerifyDNS(c echo.Context) error {
	if h.domainManager == nil || h.dnsVerifier == nil {
		return response.InternalError(c, "domain manager or DNS verifier not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get domain
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Check domain is in pending_dns status or is a legacy domain (active without proper status)
	// Legacy domains may have empty status or be active without going through the new flow
	isLegacyDomain := domain.Status == "" || (domain.IsActive && domain.Status != models.StatusActive)
	if !isLegacyDomain && domain.Status != models.StatusPendingDNS && domain.Status != models.StatusFailed {
		return response.BadRequest(c, "domain must be in pending_dns or failed status to verify DNS")
	}

	// For legacy domains without DNS challenge, generate one and update the domain
	if domain.DNSChallenge == "" {
		if err := h.domainManager.GenerateChallengeForLegacyDomain(c.Request().Context(), uint(id)); err != nil {
			return response.InternalError(c, "failed to generate DNS challenge for legacy domain")
		}
		// Reload domain with new challenge token
		domain, err = h.repo.GetByID(c.Request().Context(), uint(id))
		if err != nil {
			return response.InternalError(c, "failed to reload domain")
		}
		// Return early with message to configure DNS first
		return response.SuccessWithMessage(c, map[string]interface{}{
			"domain":  domain,
			"message": "DNS challenge token generated. Please configure the TXT record and verify again.",
		}, "DNS challenge token generated for legacy domain")
	}

	// Verify DNS records
	result, err := h.dnsVerifier.VerifyDNS(c.Request().Context(), domain)
	if err != nil {
		return response.InternalError(c, "DNS verification failed: "+err.Error())
	}

	// Update domain status if all verified
	if result.AllVerified {
		if err := h.domainManager.UpdateStatus(c.Request().Context(), uint(id), models.StatusDNSVerified, ""); err != nil {
			return response.InternalError(c, "failed to update domain status")
		}
	}

	return response.Success(c, result)
}

// GenerateCertificate handles POST /api/domains/:id/generate-cert
// Triggers ACME certificate request and updates domain status
// IMPORTANT: User must add _acme-challenge TXT record before calling this endpoint
func (h *DomainHandler) GenerateCertificate(c echo.Context) error {
	if h.domainManager == nil || h.certManager == nil {
		return response.InternalError(c, "domain manager or certificate manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get domain
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Check domain is in dns_verified status
	if domain.Status != models.StatusDNSVerified {
		return response.BadRequest(c, "domain must be in dns_verified status to generate certificate")
	}

	// Generate certificate
	cert, err := h.certManager.GenerateCertificate(c.Request().Context(), domain)
	if err != nil {
		// Check if error contains ACME challenge info
		errMsg := err.Error()
		if strings.Contains(errMsg, "_acme-challenge") {
			// Return more helpful error message
			return response.BadRequestWithData(c, "ACME DNS challenge failed. Please add the required TXT record and try again.", map[string]interface{}{
				"error":       errMsg,
				"hint":        "Add a TXT record with name '_acme-challenge." + domain.Name + "' before generating certificate",
				"retry_after": "Wait 1-5 minutes for DNS propagation after adding the TXT record",
			})
		}
		// The certificate manager already updates the domain status to failed
		return response.InternalError(c, "certificate generation failed: "+err.Error())
	}

	// Activate domain after certificate is issued
	if err := h.domainManager.ActivateDomain(c.Request().Context(), uint(id)); err != nil {
		return response.InternalError(c, "failed to activate domain")
	}

	// Get updated domain
	domain, _ = h.repo.GetByID(c.Request().Context(), uint(id))

	return response.Success(c, map[string]interface{}{
		"domain":      domain,
		"certificate": cert,
	})
}

// DomainStatusResponse represents the domain status response
type DomainStatusResponse struct {
	ID           uint                 `json:"id"`
	Name         string               `json:"name"`
	Status       models.DomainStatus  `json:"status"`
	IsActive     bool                 `json:"is_active"`
	ErrorMessage string               `json:"error_message,omitempty"`
	Progress     DomainSetupProgress  `json:"progress"`
	NextStep     string               `json:"next_step,omitempty"`
}

// DomainSetupProgress represents the progress of domain setup
type DomainSetupProgress struct {
	CurrentStep  int    `json:"current_step"`
	TotalSteps   int    `json:"total_steps"`
	StepName     string `json:"step_name"`
	IsComplete   bool   `json:"is_complete"`
}

// GetStatus handles GET /api/domains/:id/status
// Returns current domain status with progress info
func (h *DomainHandler) GetStatus(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Calculate progress based on status
	progress := getProgressForStatus(domain.Status)
	nextStep := getNextStepForStatus(domain.Status)

	statusResponse := DomainStatusResponse{
		ID:           domain.ID,
		Name:         domain.Name,
		Status:       domain.Status,
		IsActive:     domain.IsActive,
		ErrorMessage: domain.ErrorMessage,
		Progress:     progress,
		NextStep:     nextStep,
	}

	return response.Success(c, statusResponse)
}

// RetryRequest represents the request body for retry endpoint
type RetryRequest struct {
	// ResetTo allows specifying which status to reset to
	// Options: "pending_dns", "dns_verified"
	ResetTo string `json:"reset_to,omitempty"`
}

// Retry handles POST /api/domains/:id/retry
// Allows retry from failed step
// Body: { "reset_to": "dns_verified" } to retry certificate generation
func (h *DomainHandler) Retry(c echo.Context) error {
	if h.domainManager == nil {
		return response.InternalError(c, "domain manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Parse optional request body
	var req RetryRequest
	_ = c.Bind(&req) // Ignore error, body is optional

	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Determine which step to retry based on current status
	var newStatus models.DomainStatus
	var message string

	switch domain.Status {
	case models.StatusFailed:
		// Check if user wants to reset to a specific status
		if req.ResetTo == "dns_verified" {
			// Reset to dns_verified to retry certificate generation
			newStatus = models.StatusDNSVerified
			message = "Domain reset to dns_verified status for certificate retry"
		} else if req.ResetTo == "pending_dns" {
			// Reset to pending_dns to redo DNS verification
			newStatus = models.StatusPendingDNS
			message = "Domain reset to pending_dns status for retry"
		} else {
			// Default: check error message to determine best reset point
			// If error is related to certificate/ACME, reset to dns_verified
			if strings.Contains(strings.ToLower(domain.ErrorMessage), "acme") ||
				strings.Contains(strings.ToLower(domain.ErrorMessage), "certificate") ||
				strings.Contains(strings.ToLower(domain.ErrorMessage), "dns txt") ||
				strings.Contains(strings.ToLower(domain.ErrorMessage), "_acme-challenge") {
				newStatus = models.StatusDNSVerified
				message = "Domain reset to dns_verified status for certificate retry"
			} else {
				newStatus = models.StatusPendingDNS
				message = "Domain reset to pending_dns status for retry"
			}
		}
	case models.StatusPendingDNS:
		// Already in pending_dns, just clear error
		newStatus = models.StatusPendingDNS
		message = "Ready to verify DNS"
	case models.StatusDNSVerified:
		// Ready for certificate generation
		message = "Ready to generate certificate"
		return response.SuccessWithMessage(c, domain, message)
	case models.StatusPendingCertificate:
		// Certificate generation in progress, reset to dns_verified to retry
		newStatus = models.StatusDNSVerified
		message = "Domain reset to dns_verified status for certificate retry"
	case models.StatusCertificateIssued:
		// Ready for activation
		message = "Ready to activate domain"
		return response.SuccessWithMessage(c, domain, message)
	case models.StatusActive:
		message = "Domain is already active"
		return response.SuccessWithMessage(c, domain, message)
	default:
		return response.BadRequest(c, "unknown domain status")
	}

	// Update status
	if err := h.domainManager.UpdateStatus(c.Request().Context(), uint(id), newStatus, ""); err != nil {
		return response.InternalError(c, "failed to update domain status")
	}

	// Get updated domain
	domain, _ = h.repo.GetByID(c.Request().Context(), uint(id))

	return response.SuccessWithMessage(c, domain, message)
}

// getProgressForStatus returns the progress info for a given domain status
func getProgressForStatus(status models.DomainStatus) DomainSetupProgress {
	totalSteps := 4 // DNS Setup, DNS Verify, Certificate, Activate

	switch status {
	case models.StatusPendingDNS:
		return DomainSetupProgress{
			CurrentStep: 1,
			TotalSteps:  totalSteps,
			StepName:    "Configure DNS Records",
			IsComplete:  false,
		}
	case models.StatusDNSVerified:
		return DomainSetupProgress{
			CurrentStep: 2,
			TotalSteps:  totalSteps,
			StepName:    "DNS Verified",
			IsComplete:  false,
		}
	case models.StatusPendingCertificate:
		return DomainSetupProgress{
			CurrentStep: 3,
			TotalSteps:  totalSteps,
			StepName:    "Generating Certificate",
			IsComplete:  false,
		}
	case models.StatusCertificateIssued:
		return DomainSetupProgress{
			CurrentStep: 3,
			TotalSteps:  totalSteps,
			StepName:    "Certificate Issued",
			IsComplete:  false,
		}
	case models.StatusActive:
		return DomainSetupProgress{
			CurrentStep: 4,
			TotalSteps:  totalSteps,
			StepName:    "Ready to Receive Email",
			IsComplete:  true,
		}
	case models.StatusFailed:
		return DomainSetupProgress{
			CurrentStep: 0,
			TotalSteps:  totalSteps,
			StepName:    "Setup Failed",
			IsComplete:  false,
		}
	default:
		return DomainSetupProgress{
			CurrentStep: 0,
			TotalSteps:  totalSteps,
			StepName:    "Unknown",
			IsComplete:  false,
		}
	}
}

// getNextStepForStatus returns the next action for a given domain status
func getNextStepForStatus(status models.DomainStatus) string {
	switch status {
	case models.StatusPendingDNS:
		return "Configure DNS records and verify"
	case models.StatusDNSVerified:
		return "Generate SSL certificate"
	case models.StatusPendingCertificate:
		return "Waiting for certificate generation"
	case models.StatusCertificateIssued:
		return "Activate domain"
	case models.StatusActive:
		return "Domain is ready to receive email"
	case models.StatusFailed:
		return "Retry the failed step"
	default:
		return ""
	}
}


// ============================================================================
// Manual DNS Verification Handlers
// ============================================================================

// RequestACMEChallenge handles POST /api/domains/:id/request-acme-challenge
// Requests ACME challenge from Let's Encrypt and returns TXT record info for user
func (h *DomainHandler) RequestACMEChallenge(c echo.Context) error {
	if h.certManager == nil {
		return response.InternalError(c, "certificate manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get domain to validate status
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Validate domain is in dns_verified status
	if domain.Status != models.StatusDNSVerified {
		return response.BadRequestWithData(c, "domain must be in dns_verified status to request ACME challenge", map[string]interface{}{
			"current_status":   domain.Status,
			"required_status":  models.StatusDNSVerified,
			"suggested_action": "Please verify DNS records first using POST /api/domains/:id/verify-dns",
		})
	}

	// Request ACME challenge
	challengeInfo, err := h.certManager.RequestACMEChallenge(c.Request().Context(), uint(id))
	if err != nil {
		return response.InternalError(c, "failed to request ACME challenge: "+err.Error())
	}

	return response.Success(c, challengeInfo)
}


// VerifyACMEDNS handles POST /api/domains/:id/verify-acme-dns
// Verifies that the ACME challenge TXT record is correctly configured
// Does NOT submit to Let's Encrypt - just local DNS check
func (h *DomainHandler) VerifyACMEDNS(c echo.Context) error {
	if h.certManager == nil {
		return response.InternalError(c, "certificate manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get domain to validate it has an active challenge
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Validate domain has an active challenge
	if domain.ACMEChallengeToken == "" || domain.ACMEChallengeValue == "" {
		return response.BadRequestWithData(c, "no active ACME challenge found", map[string]interface{}{
			"current_status":   domain.Status,
			"suggested_action": "Please request an ACME challenge first using POST /api/domains/:id/request-acme-challenge",
		})
	}

	// Verify ACME DNS
	result, err := h.certManager.VerifyACMEDNS(c.Request().Context(), uint(id))
	if err != nil {
		return response.InternalError(c, "failed to verify ACME DNS: "+err.Error())
	}

	return response.Success(c, result)
}


// SubmitACMEChallenge handles POST /api/domains/:id/submit-acme-challenge
// Submits the ACME challenge to Let's Encrypt for validation and generates certificate
// Should only be called after VerifyACMEDNS returns success
func (h *DomainHandler) SubmitACMEChallenge(c echo.Context) error {
	if h.certManager == nil {
		return response.InternalError(c, "certificate manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Get domain to validate status
	domain, err := h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Validate domain is in acme_challenge_ready status
	if domain.Status != models.StatusACMEChallengeReady {
		return response.BadRequestWithData(c, "domain must be in acme_challenge_ready status to submit ACME challenge", map[string]interface{}{
			"current_status":   domain.Status,
			"required_status":  models.StatusACMEChallengeReady,
			"suggested_action": "Please verify DNS first using POST /api/domains/:id/verify-acme-dns",
		})
	}

	// Submit ACME challenge
	err = h.certManager.SubmitACMEChallenge(c.Request().Context(), uint(id))
	if err != nil {
		// Get updated domain to include error message
		domain, _ = h.repo.GetByID(c.Request().Context(), uint(id))
		return response.BadRequestWithData(c, "ACME challenge submission failed", map[string]interface{}{
			"error":            err.Error(),
			"domain_status":    domain.Status,
			"suggested_action": "Please check the error message and try again. You may need to verify DNS again or request a new challenge.",
		})
	}

	// Get updated domain with certificate info
	domain, _ = h.repo.GetByID(c.Request().Context(), uint(id))

	// Activate domain after certificate is issued
	if h.domainManager != nil && domain.Status == models.StatusCertificateIssued {
		if err := h.domainManager.ActivateDomain(c.Request().Context(), uint(id)); err != nil {
			return response.InternalError(c, "certificate issued but failed to activate domain: "+err.Error())
		}
		// Get updated domain
		domain, _ = h.repo.GetByID(c.Request().Context(), uint(id))
	}

	return response.SuccessWithMessage(c, domain, "Certificate generated and domain activated successfully!")
}


// GetACMEStatus handles GET /api/domains/:id/acme-status
// Returns the current ACME challenge status with challenge info
func (h *DomainHandler) GetACMEStatus(c echo.Context) error {
	if h.certManager == nil {
		return response.InternalError(c, "certificate manager not configured")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "invalid domain ID")
	}

	// Check domain exists
	_, err = h.repo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "domain not found")
		}
		return response.InternalError(c, "failed to get domain")
	}

	// Get ACME status
	status, err := h.certManager.GetACMEStatus(c.Request().Context(), uint(id))
	if err != nil {
		return response.InternalError(c, "failed to get ACME status: "+err.Error())
	}

	return response.Success(c, status)
}
