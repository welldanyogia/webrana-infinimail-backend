package handlers

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// MailboxHandler handles mailbox-related HTTP requests
type MailboxHandler struct {
	mailboxRepo repository.MailboxRepository
	domainRepo  repository.DomainRepository
}

// NewMailboxHandler creates a new MailboxHandler
func NewMailboxHandler(mailboxRepo repository.MailboxRepository, domainRepo repository.DomainRepository) *MailboxHandler {
	return &MailboxHandler{
		mailboxRepo: mailboxRepo,
		domainRepo:  domainRepo,
	}
}

// CreateMailboxRequest represents the request body for creating a mailbox
type CreateMailboxRequest struct {
	LocalPart string `json:"local_part" validate:"required"`
	DomainID  uint   `json:"domain_id" validate:"required"`
}

// CreateRandomMailboxRequest represents the request body for creating a random mailbox
type CreateRandomMailboxRequest struct {
	DomainID uint `json:"domain_id" validate:"required"`
}

// Create handles POST /api/mailboxes
func (h *MailboxHandler) Create(c echo.Context) error {
	var req CreateMailboxRequest
	if err := c.Bind(&req); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if req.LocalPart == "" {
		return api.BadRequest(c, "local_part is required")
	}
	if req.DomainID == 0 {
		return api.BadRequest(c, "domain_id is required")
	}

	// Get domain to verify it exists and is active
	domain, err := h.domainRepo.GetByID(c.Request().Context(), req.DomainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "domain not found")
		}
		return api.InternalError(c, "failed to get domain")
	}

	if !domain.IsActive {
		return api.BadRequest(c, "domain is not active")
	}

	mailbox := &models.Mailbox{
		LocalPart:   req.LocalPart,
		DomainID:    req.DomainID,
		FullAddress: req.LocalPart + "@" + domain.Name,
	}

	if err := h.mailboxRepo.Create(c.Request().Context(), mailbox); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			return api.Conflict(c, "mailbox already exists")
		}
		return api.InternalError(c, "failed to create mailbox")
	}

	return api.Created(c, mailbox)
}

// CreateRandom handles POST /api/mailboxes/random
func (h *MailboxHandler) CreateRandom(c echo.Context) error {
	var req CreateRandomMailboxRequest
	if err := c.Bind(&req); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if req.DomainID == 0 {
		return api.BadRequest(c, "domain_id is required")
	}

	// Get domain to verify it exists and is active
	domain, err := h.domainRepo.GetByID(c.Request().Context(), req.DomainID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "domain not found")
		}
		return api.InternalError(c, "failed to get domain")
	}

	if !domain.IsActive {
		return api.BadRequest(c, "domain is not active")
	}

	// Generate random 8-character alphanumeric local part
	localPart := generateRandomString(8)

	mailbox := &models.Mailbox{
		LocalPart:   localPart,
		DomainID:    req.DomainID,
		FullAddress: localPart + "@" + domain.Name,
	}

	if err := h.mailboxRepo.Create(c.Request().Context(), mailbox); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			// Extremely rare collision, try again
			localPart = generateRandomString(8)
			mailbox.LocalPart = localPart
			mailbox.FullAddress = localPart + "@" + domain.Name
			if err := h.mailboxRepo.Create(c.Request().Context(), mailbox); err != nil {
				return api.InternalError(c, "failed to create mailbox")
			}
		} else {
			return api.InternalError(c, "failed to create mailbox")
		}
	}

	return api.Created(c, mailbox)
}

// List handles GET /api/mailboxes
func (h *MailboxHandler) List(c echo.Context) error {
	domainIDStr := c.QueryParam("domain_id")
	if domainIDStr == "" {
		return api.BadRequest(c, "domain_id is required")
	}

	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid domain_id")
	}

	limit := 20
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	mailboxes, total, err := h.mailboxRepo.ListByDomain(c.Request().Context(), uint(domainID), limit, offset)
	if err != nil {
		return api.InternalError(c, "failed to list mailboxes")
	}

	return api.Paginated(c, mailboxes, total, limit, offset)
}

// Get handles GET /api/mailboxes/:id
func (h *MailboxHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid mailbox ID")
	}

	mailbox, err := h.mailboxRepo.GetByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "mailbox not found")
		}
		return api.InternalError(c, "failed to get mailbox")
	}

	// Update last accessed timestamp
	_ = h.mailboxRepo.UpdateLastAccessed(c.Request().Context(), uint(id))

	return api.Success(c, mailbox)
}

// Delete handles DELETE /api/mailboxes/:id
func (h *MailboxHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return api.BadRequest(c, "invalid mailbox ID")
	}

	if err := h.mailboxRepo.Delete(c.Request().Context(), uint(id)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.NotFound(c, "mailbox not found")
		}
		return api.InternalError(c, "failed to delete mailbox")
	}

	return api.NoContent(c)
}

// generateRandomString generates a random alphanumeric string of given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}
