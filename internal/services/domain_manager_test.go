package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

// MockDomainRepository is a mock implementation of DomainRepository
type MockDomainRepository struct {
	mock.Mock
}

func (m *MockDomainRepository) Create(ctx context.Context, domain *models.Domain) error {
	args := m.Called(ctx, domain)
	return args.Error(0)
}

func (m *MockDomainRepository) GetByID(ctx context.Context, id uint) (*models.Domain, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Domain), args.Error(1)
}

func (m *MockDomainRepository) GetByName(ctx context.Context, name string) (*models.Domain, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Domain), args.Error(1)
}

func (m *MockDomainRepository) List(ctx context.Context, activeOnly bool) ([]models.Domain, error) {
	args := m.Called(ctx, activeOnly)
	return args.Get(0).([]models.Domain), args.Error(1)
}

func (m *MockDomainRepository) Update(ctx context.Context, domain *models.Domain) error {
	args := m.Called(ctx, domain)
	return args.Error(0)
}

func (m *MockDomainRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCreateDomain_Success(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{
		SMTPHostname: "mail.test.com",
		ServerIP:     "192.168.1.1",
	}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Name == "example.com" &&
			d.Status == models.StatusPendingDNS &&
			d.IsActive == false &&
			d.DNSChallenge != ""
	})).Return(nil)

	domain, err := service.CreateDomain(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.NotNil(t, domain)
	assert.Equal(t, "example.com", domain.Name)
	assert.Equal(t, models.StatusPendingDNS, domain.Status)
	assert.False(t, domain.IsActive)
	assert.NotEmpty(t, domain.DNSChallenge)
	assert.Len(t, domain.DNSChallenge, 32) // 16 bytes = 32 hex chars
	mockRepo.AssertExpectations(t)
}

func TestCreateDomain_EmptyName(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{
		SMTPHostname: "mail.test.com",
		ServerIP:     "192.168.1.1",
	}
	service := NewDomainManagerService(mockRepo, config)

	domain, err := service.CreateDomain(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, domain)
	assert.Contains(t, err.Error(), "domain name cannot be empty")
}

func TestCreateDomain_TrimsAndLowercases(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{
		SMTPHostname: "mail.test.com",
		ServerIP:     "192.168.1.1",
	}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Name == "example.com"
	})).Return(nil)

	domain, err := service.CreateDomain(context.Background(), "  EXAMPLE.COM  ")

	assert.NoError(t, err)
	assert.Equal(t, "example.com", domain.Name)
	mockRepo.AssertExpectations(t)
}

func TestCreateDomain_RepositoryError(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{
		SMTPHostname: "mail.test.com",
		ServerIP:     "192.168.1.1",
	}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error"))

	domain, err := service.CreateDomain(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Nil(t, domain)
	assert.Contains(t, err.Error(), "failed to create domain")
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus_Success(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:     1,
		Name:   "example.com",
		Status: models.StatusPendingDNS,
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Status == models.StatusDNSVerified && d.ErrorMessage == ""
	})).Return(nil)

	err := service.UpdateStatus(context.Background(), 1, models.StatusDNSVerified, "")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus_WithErrorMessage(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:     1,
		Name:   "example.com",
		Status: models.StatusPendingDNS,
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Status == models.StatusFailed && d.ErrorMessage == "DNS verification failed"
	})).Return(nil)

	err := service.UpdateStatus(context.Background(), 1, models.StatusFailed, "DNS verification failed")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus_InvalidStatus(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	err := service.UpdateStatus(context.Background(), 1, "invalid_status", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid domain status")
}

func TestUpdateStatus_DomainNotFound(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, repository.ErrNotFound)

	err := service.UpdateStatus(context.Background(), 1, models.StatusDNSVerified, "")

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetDNSGuide_Success(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{
		SMTPHostname: "mail.infinimail.webrana.id",
		ServerIP:     "103.123.45.67",
	}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:           1,
		Name:         "example.com",
		DNSChallenge: "abc123xyz",
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)

	guide, err := service.GetDNSGuide(context.Background(), 1)

	assert.NoError(t, err)
	assert.NotNil(t, guide)

	// Verify MX Record
	assert.Equal(t, "MX", guide.MXRecord.Type)
	assert.Equal(t, "example.com", guide.MXRecord.Name)
	assert.Equal(t, "mail.infinimail.webrana.id", guide.MXRecord.Value)
	assert.Equal(t, 10, guide.MXRecord.Priority)
	assert.Equal(t, 3600, guide.MXRecord.TTL)

	// Verify A Record
	assert.Equal(t, "A", guide.ARecord.Type)
	assert.Equal(t, "mail.example.com", guide.ARecord.Name)
	assert.Equal(t, "103.123.45.67", guide.ARecord.Value)
	assert.Equal(t, 3600, guide.ARecord.TTL)

	// Verify TXT Record
	assert.Equal(t, "TXT", guide.TXTRecord.Type)
	assert.Equal(t, "_infinimail.example.com", guide.TXTRecord.Name)
	assert.Equal(t, "infinimail-verify=abc123xyz", guide.TXTRecord.Value)
	assert.Equal(t, 3600, guide.TXTRecord.TTL)

	// Verify config values
	assert.Equal(t, "mail.infinimail.webrana.id", guide.SMTPHost)
	assert.Equal(t, "103.123.45.67", guide.ServerIP)

	mockRepo.AssertExpectations(t)
}

func TestGetDNSGuide_DomainNotFound(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, repository.ErrNotFound)

	guide, err := service.GetDNSGuide(context.Background(), 1)

	assert.Error(t, err)
	assert.Nil(t, guide)
	mockRepo.AssertExpectations(t)
}

func TestActivateDomain_Success(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:       1,
		Name:     "example.com",
		Status:   models.StatusCertificateIssued,
		IsActive: false,
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *models.Domain) bool {
		return d.Status == models.StatusActive && d.IsActive == true
	})).Return(nil)

	err := service.ActivateDomain(context.Background(), 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestActivateDomain_WrongStatus(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:     1,
		Name:   "example.com",
		Status: models.StatusPendingDNS,
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)

	err := service.ActivateDomain(context.Background(), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be in certificate_issued status")
	mockRepo.AssertExpectations(t)
}

func TestActivateDomain_DomainNotFound(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, repository.ErrNotFound)

	err := service.ActivateDomain(context.Background(), 1)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetDomain_Success(t *testing.T) {
	mockRepo := new(MockDomainRepository)
	config := DomainManagerConfig{}
	service := NewDomainManagerService(mockRepo, config)

	existingDomain := &models.Domain{
		ID:   1,
		Name: "example.com",
	}

	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(existingDomain, nil)

	domain, err := service.GetDomain(context.Background(), 1)

	assert.NoError(t, err)
	assert.NotNil(t, domain)
	assert.Equal(t, uint(1), domain.ID)
	mockRepo.AssertExpectations(t)
}

func TestGenerateChallengeToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := generateChallengeToken()
		assert.NoError(t, err)
		assert.Len(t, token, 32)
		assert.False(t, tokens[token], "Token should be unique")
		tokens[token] = true
	}
}
