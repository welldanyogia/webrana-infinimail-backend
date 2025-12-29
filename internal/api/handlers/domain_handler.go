package handlers

import (
	"errors"
	"strconv"

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

	// Check domain is in pending_dns status
	if domain.Status != models.StatusPendingDNS && domain.Status != models.StatusFailed {
		return response.BadRequest(c, "domain must be in pending_dns or failed status to verify DNS")
	}

	// Verify DNS records
	result, err := h.dnsVerifier.VerifyDNS(c.Request().Context(), domain)
	if err != nil {
		return response.InternalError(c, "DNS verification failed")
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

// Retry handles POST /api/domains/:id/retry
// Allows retry from failed step
func (h *DomainHandler) Retry(c echo.Context) error {
	if h.domainManager == nil {
		return response.InternalError(c, "domain manager not configured")
	}

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

	// Determine which step to retry based on current status
	var newStatus models.DomainStatus
	var message string

	switch domain.Status {
	case models.StatusFailed:
		// Determine the step that failed based on error message or previous state
		// For simplicity, reset to pending_dns to allow full retry
		newStatus = models.StatusPendingDNS
		message = "Domain reset to pending_dns status for retry"
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
