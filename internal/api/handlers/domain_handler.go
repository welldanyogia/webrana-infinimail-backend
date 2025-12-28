package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// DomainHandler handles domain-related HTTP requests
type DomainHandler struct {
	repo repository.DomainRepository
}

// NewDomainHandler creates a new DomainHandler
func NewDomainHandler(repo repository.DomainRepository) *DomainHandler {
	return &DomainHandler{repo: repo}
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
func (h *DomainHandler) Create(c echo.Context) error {
	var req CreateDomainRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(c, "name is required")
	}

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
