package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/gorm"
)

// DomainRepository defines the interface for domain data access
type DomainRepository interface {
	Create(ctx context.Context, domain *models.Domain) error
	GetByID(ctx context.Context, id uint) (*models.Domain, error)
	GetByName(ctx context.Context, name string) (*models.Domain, error)
	List(ctx context.Context, activeOnly bool) ([]models.Domain, error)
	Update(ctx context.Context, domain *models.Domain) error
	Delete(ctx context.Context, id uint) error
}

// domainRepository implements DomainRepository using GORM
type domainRepository struct {
	db *gorm.DB
}

// NewDomainRepository creates a new DomainRepository instance
func NewDomainRepository(db *gorm.DB) DomainRepository {
	return &domainRepository{db: db}
}

// Create creates a new domain
func (r *domainRepository) Create(ctx context.Context, domain *models.Domain) error {
	result := r.db.WithContext(ctx).Create(domain)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return fmt.Errorf("domain with name '%s' already exists: %w", domain.Name, ErrDuplicateEntry)
		}
		return fmt.Errorf("failed to create domain: %w", result.Error)
	}
	return nil
}

// GetByID retrieves a domain by its ID
func (r *domainRepository) GetByID(ctx context.Context, id uint) (*models.Domain, error) {
	var domain models.Domain
	result := r.db.WithContext(ctx).First(&domain, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get domain by ID: %w", result.Error)
	}
	return &domain, nil
}

// GetByName retrieves a domain by its name
func (r *domainRepository) GetByName(ctx context.Context, name string) (*models.Domain, error) {
	var domain models.Domain
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&domain)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get domain by name: %w", result.Error)
	}
	return &domain, nil
}

// List retrieves all domains, optionally filtering by active status
func (r *domainRepository) List(ctx context.Context, activeOnly bool) ([]models.Domain, error) {
	var domains []models.Domain
	query := r.db.WithContext(ctx)
	
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	
	result := query.Order("name ASC").Find(&domains)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list domains: %w", result.Error)
	}
	return domains, nil
}

// Update updates an existing domain
func (r *domainRepository) Update(ctx context.Context, domain *models.Domain) error {
	result := r.db.WithContext(ctx).Save(domain)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return fmt.Errorf("domain with name '%s' already exists: %w", domain.Name, ErrDuplicateEntry)
		}
		return fmt.Errorf("failed to update domain: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a domain by its ID (cascade deletes mailboxes, messages, attachments)
func (r *domainRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Domain{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete domain: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
