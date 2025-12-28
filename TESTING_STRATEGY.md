# Comprehensive Testing Strategy - Webrana Infinimail Backend

## 📋 Ringkasan Strategi

Strategi testing ini mencakup **4 layer testing** yang saling melengkapi:

```
┌─────────────────────────────────────────────────────────────┐
│                    E2E Tests (End-to-End)                   │
│         Full system integration dengan real services        │
├─────────────────────────────────────────────────────────────┤
│                   Integration Tests                         │
│           Database, SMTP, WebSocket, Storage                │
├─────────────────────────────────────────────────────────────┤
│                     Unit Tests                              │
│        Repository, Handler, SMTP Parser, Models             │
├─────────────────────────────────────────────────────────────┤
│                   Mocks & Fixtures                          │
│          Test doubles, data builders, helpers               │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 Testing Layers

### Layer 1: Unit Tests

Unit tests fokus pada logic individual tanpa dependencies external.

**Target Coverage:** 80%+

| Component | File Location | Test Focus |
|-----------|---------------|------------|
| Repository | `internal/repository/*_test.go` | CRUD operations dengan mock DB |
| Handlers | `internal/api/handlers/*_test.go` | HTTP request/response handling |
| SMTP Parser | `internal/smtp/parser_test.go` | Email parsing logic |
| Models | `internal/models/*_test.go` | Model validation |
| Errors | `internal/repository/errors_test.go` | Error detection functions |

### Layer 2: Integration Tests

Integration tests memverifikasi interaksi antar komponen.

**Target Coverage:** 70%+

| Integration | Focus |
|-------------|-------|
| Database | Repository + Real PostgreSQL (Docker) |
| SMTP | SMTP Backend + Email Processing |
| WebSocket | Real-time notification delivery |
| Storage | File storage operations |

### Layer 3: E2E Tests

End-to-end tests memverifikasi full user workflows.

**Scenarios:**
- Email receiving flow (SMTP → Database → WebSocket notification)
- Domain management via REST API
- Mailbox CRUD operations
- Attachment handling

### Layer 4: Performance & Load Tests

- Concurrent email receiving
- API throughput under load
- Database connection pool behavior

---

## 📁 Struktur File Testing

```
webrana-infinimail-backend/
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       ├── domain_handler.go
│   │       ├── domain_handler_test.go          # ← Unit test
│   │       ├── mailbox_handler.go
│   │       ├── mailbox_handler_test.go         # ← Unit test
│   │       ├── message_handler.go
│   │       ├── message_handler_test.go         # ← Unit test
│   │       └── attachment_handler.go
│   │       └── attachment_handler_test.go      # ← Unit test
│   ├── repository/
│   │   ├── domain_repository.go
│   │   ├── domain_repository_test.go           # ← Unit test
│   │   ├── mailbox_repository.go
│   │   ├── mailbox_repository_test.go          # ← Unit test
│   │   ├── message_repository.go
│   │   ├── message_repository_test.go          # ← Unit test
│   │   └── attachment_repository.go
│   │   └── attachment_repository_test.go       # ← Unit test
│   ├── smtp/
│   │   ├── parser.go
│   │   ├── parser_test.go                      # ← Unit test
│   │   ├── session.go
│   │   ├── session_test.go                     # ← Unit test
│   │   └── backend.go
│   │   └── backend_test.go                     # ← Unit test
│   └── models/
│       ├── domain_test.go                      # ← Unit test
│       ├── mailbox_test.go                     # ← Unit test
│       └── message_test.go                     # ← Unit test
├── tests/
│   ├── mocks/
│   │   ├── repository_mocks.go                 # Mock interfaces
│   │   ├── storage_mocks.go
│   │   └── websocket_mocks.go
│   ├── fixtures/
│   │   ├── emails/                             # Sample email files
│   │   └── testdata.go                         # Test data builders
│   ├── integration/
│   │   ├── database_test.go
│   │   ├── smtp_integration_test.go
│   │   └── api_integration_test.go
│   └── e2e/
│       └── email_flow_test.go
├── Makefile                                     # Test commands
└── docker-compose.test.yml                      # Test dependencies
```

---

## 🔧 Tools & Dependencies

Tambahkan dependencies testing ke `go.mod`:

```bash
# Install testing tools
go get github.com/stretchr/testify
go get github.com/golang/mock/mockgen
go get github.com/DATA-DOG/go-sqlmock
go get github.com/testcontainers/testcontainers-go
```

**Required Tools:**
| Tool | Purpose |
|------|---------|
| `testify` | Assertions & test suites |
| `mockgen` | Generate mock interfaces |
| `go-sqlmock` | Mock database for unit tests |
| `testcontainers-go` | Docker containers for integration tests |

---

## 📝 Contoh Implementasi

### 1. Mock Repository Interface

**File:** `tests/mocks/repository_mocks.go`

```go
package mocks

import (
    "context"
    "github.com/stretchr/testify/mock"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/models"
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
```

### 2. Handler Unit Test

**File:** `internal/api/handlers/domain_handler_test.go`

```go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/models"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
    "github.com/welldanyogia/webrana-infinimail-backend/tests/mocks"
)

func TestDomainHandler_Create_Success(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    reqBody := `{"name": "example.com", "is_active": true}`
    req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewBufferString(reqBody))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Expectations
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Domain")).
        Return(nil).
        Run(func(args mock.Arguments) {
            domain := args.Get(1).(*models.Domain)
            domain.ID = 1 // Simulate DB setting ID
        })

    // Execute
    err := handler.Create(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_Create_DuplicateError(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    reqBody := `{"name": "example.com"}`
    req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewBufferString(reqBody))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Expectations - return duplicate error
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Domain")).
        Return(repository.ErrDuplicateEntry)

    // Execute
    err := handler.Create(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusConflict, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_Create_EmptyName(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    reqBody := `{"name": ""}`
    req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewBufferString(reqBody))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Execute
    err := handler.Create(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDomainHandler_Get_Success(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    expectedDomain := &models.Domain{
        ID:       1,
        Name:     "example.com",
        IsActive: true,
    }

    req := httptest.NewRequest(http.MethodGet, "/api/domains/1", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("1")

    // Expectations
    mockRepo.On("GetByID", mock.Anything, uint(1)).Return(expectedDomain, nil)

    // Execute
    err := handler.Get(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_Get_NotFound(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    req := httptest.NewRequest(http.MethodGet, "/api/domains/999", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("999")

    // Expectations
    mockRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrNotFound)

    // Execute
    err := handler.Get(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusNotFound, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_List_Success(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    domains := []models.Domain{
        {ID: 1, Name: "example.com", IsActive: true},
        {ID: 2, Name: "test.com", IsActive: true},
    }

    req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Expectations
    mockRepo.On("List", mock.Anything, false).Return(domains, nil)

    // Execute
    err := handler.List(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_List_ActiveOnly(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    domains := []models.Domain{
        {ID: 1, Name: "example.com", IsActive: true},
    }

    req := httptest.NewRequest(http.MethodGet, "/api/domains?active_only=true", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Expectations
    mockRepo.On("List", mock.Anything, true).Return(domains, nil)

    // Execute
    err := handler.List(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
    mockRepo.AssertExpectations(t)
}

func TestDomainHandler_Delete_Success(t *testing.T) {
    // Setup
    e := echo.New()
    mockRepo := new(mocks.MockDomainRepository)
    handler := NewDomainHandler(mockRepo)

    req := httptest.NewRequest(http.MethodDelete, "/api/domains/1", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("1")

    // Expectations
    mockRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

    // Execute
    err := handler.Delete(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusNoContent, rec.Code)
    mockRepo.AssertExpectations(t)
}
```

### 3. Repository Unit Test dengan SQLMock

**File:** `internal/repository/domain_repository_test.go`

```go
package repository

import (
    "context"
    "regexp"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/models"
)

type DomainRepositoryTestSuite struct {
    suite.Suite
    db   *gorm.DB
    mock sqlmock.Sqlmock
    repo DomainRepository
}

func (s *DomainRepositoryTestSuite) SetupTest() {
    sqlDB, mock, err := sqlmock.New()
    s.Require().NoError(err)
    s.mock = mock

    dialector := postgres.New(postgres.Config{
        Conn:       sqlDB,
        DriverName: "postgres",
    })
    s.db, err = gorm.Open(dialector, &gorm.Config{})
    s.Require().NoError(err)

    s.repo = NewDomainRepository(s.db)
}

func (s *DomainRepositoryTestSuite) TearDownTest() {
    db, _ := s.db.DB()
    db.Close()
}

func TestDomainRepositorySuite(t *testing.T) {
    suite.Run(t, new(DomainRepositoryTestSuite))
}

func (s *DomainRepositoryTestSuite) TestCreate_Success() {
    domain := &models.Domain{
        Name:     "example.com",
        IsActive: true,
    }

    s.mock.ExpectBegin()
    s.mock.ExpectQuery(regexp.QuoteMeta(
        `INSERT INTO "domains"`)).
        WithArgs(domain.Name, domain.IsActive, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
    s.mock.ExpectCommit()

    err := s.repo.Create(context.Background(), domain)

    assert.NoError(s.T(), err)
    assert.Equal(s.T(), uint(1), domain.ID)
    assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *DomainRepositoryTestSuite) TestGetByID_Success() {
    rows := sqlmock.NewRows([]string{"id", "name", "is_active", "created_at", "updated_at"}).
        AddRow(1, "example.com", true, time.Now(), time.Now())

    s.mock.ExpectQuery(regexp.QuoteMeta(
        `SELECT * FROM "domains" WHERE "domains"."id" = $1`)).
        WithArgs(1, 1). // GORM adds LIMIT 1
        WillReturnRows(rows)

    domain, err := s.repo.GetByID(context.Background(), 1)

    assert.NoError(s.T(), err)
    assert.NotNil(s.T(), domain)
    assert.Equal(s.T(), "example.com", domain.Name)
    assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *DomainRepositoryTestSuite) TestGetByID_NotFound() {
    s.mock.ExpectQuery(regexp.QuoteMeta(
        `SELECT * FROM "domains" WHERE "domains"."id" = $1`)).
        WithArgs(999, 1).
        WillReturnRows(sqlmock.NewRows([]string{}))

    domain, err := s.repo.GetByID(context.Background(), 999)

    assert.Error(s.T(), err)
    assert.Nil(s.T(), domain)
    assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *DomainRepositoryTestSuite) TestList_AllDomains() {
    rows := sqlmock.NewRows([]string{"id", "name", "is_active", "created_at", "updated_at"}).
        AddRow(1, "active.com", true, time.Now(), time.Now()).
        AddRow(2, "inactive.com", false, time.Now(), time.Now())

    s.mock.ExpectQuery(regexp.QuoteMeta(
        `SELECT * FROM "domains" ORDER BY name ASC`)).
        WillReturnRows(rows)

    domains, err := s.repo.List(context.Background(), false)

    assert.NoError(s.T(), err)
    assert.Len(s.T(), domains, 2)
}

func (s *DomainRepositoryTestSuite) TestList_ActiveOnly() {
    rows := sqlmock.NewRows([]string{"id", "name", "is_active", "created_at", "updated_at"}).
        AddRow(1, "active.com", true, time.Now(), time.Now())

    s.mock.ExpectQuery(regexp.QuoteMeta(
        `SELECT * FROM "domains" WHERE is_active = $1 ORDER BY name ASC`)).
        WithArgs(true).
        WillReturnRows(rows)

    domains, err := s.repo.List(context.Background(), true)

    assert.NoError(s.T(), err)
    assert.Len(s.T(), domains, 1)
    assert.True(s.T(), domains[0].IsActive)
}

func (s *DomainRepositoryTestSuite) TestDelete_Success() {
    s.mock.ExpectBegin()
    s.mock.ExpectExec(regexp.QuoteMeta(
        `DELETE FROM "domains" WHERE "domains"."id" = $1`)).
        WithArgs(1).
        WillReturnResult(sqlmock.NewResult(0, 1))
    s.mock.ExpectCommit()

    err := s.repo.Delete(context.Background(), 1)

    assert.NoError(s.T(), err)
}

func (s *DomainRepositoryTestSuite) TestDelete_NotFound() {
    s.mock.ExpectBegin()
    s.mock.ExpectExec(regexp.QuoteMeta(
        `DELETE FROM "domains" WHERE "domains"."id" = $1`)).
        WithArgs(999).
        WillReturnResult(sqlmock.NewResult(0, 0))
    s.mock.ExpectCommit()

    err := s.repo.Delete(context.Background(), 999)

    assert.ErrorIs(s.T(), err, ErrNotFound)
}
```

### 4. Integration Test dengan Testcontainers

**File:** `tests/integration/database_test.go`

```go
//go:build integration

package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/models"
    "github.com/welldanyogia/webrana-infinimail-backend/internal/repository"
)

type DatabaseIntegrationSuite struct {
    suite.Suite
    container testcontainers.Container
    db        *gorm.DB
}

func (s *DatabaseIntegrationSuite) SetupSuite() {
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "postgres:15-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "testdb",
            "POSTGRES_USER":     "testuser",
            "POSTGRES_PASSWORD": "testpass",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections").
            WithOccurrence(2).
            WithStartupTimeout(60 * time.Second),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    s.Require().NoError(err)
    s.container = container

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")

    dsn := "host=" + host + " user=testuser password=testpass dbname=testdb port=" + port.Port() + " sslmode=disable"
    s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    s.Require().NoError(err)

    // Auto migrate
    s.db.AutoMigrate(&models.Domain{}, &models.Mailbox{}, &models.Message{}, &models.Attachment{})
}

func (s *DatabaseIntegrationSuite) TearDownSuite() {
    if s.container != nil {
        s.container.Terminate(context.Background())
    }
}

func (s *DatabaseIntegrationSuite) SetupTest() {
    // Clean tables before each test
    s.db.Exec("TRUNCATE TABLE attachments, messages, mailboxes, domains CASCADE")
}

func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    suite.Run(t, new(DatabaseIntegrationSuite))
}

func (s *DatabaseIntegrationSuite) TestDomainRepository_CRUD() {
    repo := repository.NewDomainRepository(s.db)
    ctx := context.Background()

    // Create
    domain := &models.Domain{Name: "test.com", IsActive: true}
    err := repo.Create(ctx, domain)
    s.NoError(err)
    s.NotZero(domain.ID)

    // Read
    found, err := repo.GetByID(ctx, domain.ID)
    s.NoError(err)
    s.Equal("test.com", found.Name)

    // Update
    found.Name = "updated.com"
    err = repo.Update(ctx, found)
    s.NoError(err)

    updated, _ := repo.GetByID(ctx, domain.ID)
    s.Equal("updated.com", updated.Name)

    // Delete
    err = repo.Delete(ctx, domain.ID)
    s.NoError(err)

    _, err = repo.GetByID(ctx, domain.ID)
    s.ErrorIs(err, repository.ErrNotFound)
}

func (s *DatabaseIntegrationSuite) TestMailboxRepository_WithDomain() {
    domainRepo := repository.NewDomainRepository(s.db)
    mailboxRepo := repository.NewMailboxRepository(s.db)
    ctx := context.Background()

    // Create domain first
    domain := &models.Domain{Name: "mail.com", IsActive: true}
    domainRepo.Create(ctx, domain)

    // Create mailbox
    mailbox := &models.Mailbox{
        LocalPart: "user",
        DomainID:  domain.ID,
    }
    err := mailboxRepo.Create(ctx, mailbox)
    s.NoError(err)

    // Find by email
    found, err := mailboxRepo.GetByEmail(ctx, "user@mail.com")
    s.NoError(err)
    s.Equal("user", found.LocalPart)
}

func (s *DatabaseIntegrationSuite) TestCascadeDelete() {
    domainRepo := repository.NewDomainRepository(s.db)
    mailboxRepo := repository.NewMailboxRepository(s.db)
    ctx := context.Background()

    // Create domain with mailbox
    domain := &models.Domain{Name: "cascade.com", IsActive: true}
    domainRepo.Create(ctx, domain)

    mailbox := &models.Mailbox{LocalPart: "test", DomainID: domain.ID}
    mailboxRepo.Create(ctx, mailbox)

    // Delete domain should cascade delete mailbox
    err := domainRepo.Delete(ctx, domain.ID)
    s.NoError(err)

    _, err = mailboxRepo.GetByID(ctx, mailbox.ID)
    s.ErrorIs(err, repository.ErrNotFound)
}
```

### 5. SMTP Parser Unit Test

**File:** `internal/smtp/parser_test.go`

```go
package smtp

import (
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestParseEmail_SimpleText(t *testing.T) {
    rawEmail := `From: sender@example.com
To: receiver@test.com
Subject: Test Email
Content-Type: text/plain; charset=utf-8

Hello, this is a test email.`

    parsed, err := ParseEmail(strings.NewReader(rawEmail))

    assert.NoError(t, err)
    assert.Equal(t, "sender@example.com", parsed.From)
    assert.Contains(t, parsed.To, "receiver@test.com")
    assert.Equal(t, "Test Email", parsed.Subject)
    assert.Contains(t, parsed.TextBody, "Hello, this is a test email")
}

func TestParseEmail_WithAttachment(t *testing.T) {
    rawEmail := `From: sender@example.com
To: receiver@test.com
Subject: Email with attachment
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="----=_Part_0"

------=_Part_0
Content-Type: text/plain; charset=utf-8

Email body text.

------=_Part_0
Content-Type: application/pdf; name="document.pdf"
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQKJeLjz9MKMSAwIG9iago8PAovVHlwZSAvQ2F0YWxvZwo+PgplbmRvYmoKCg==

------=_Part_0--`

    parsed, err := ParseEmail(strings.NewReader(rawEmail))

    assert.NoError(t, err)
    assert.Len(t, parsed.Attachments, 1)
    assert.Equal(t, "document.pdf", parsed.Attachments[0].Filename)
    assert.Equal(t, "application/pdf", parsed.Attachments[0].ContentType)
}

func TestParseEmail_HTMLContent(t *testing.T) {
    rawEmail := `From: sender@example.com
To: receiver@test.com
Subject: HTML Email
Content-Type: text/html; charset=utf-8

<html><body><h1>Hello World</h1></body></html>`

    parsed, err := ParseEmail(strings.NewReader(rawEmail))

    assert.NoError(t, err)
    assert.Contains(t, parsed.HTMLBody, "<h1>Hello World</h1>")
}

func TestParseEmailAddress(t *testing.T) {
    tests := []struct {
        input    string
        expected struct {
            localPart string
            domain    string
        }
    }{
        {"user@example.com", struct{ localPart, domain string }{"user", "example.com"}},
        {"john.doe@sub.domain.com", struct{ localPart, domain string }{"john.doe", "sub.domain.com"}},
        {"test+filter@mail.com", struct{ localPart, domain string }{"test+filter", "mail.com"}},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            local, domain, err := ParseEmailAddress(tt.input)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected.localPart, local)
            assert.Equal(t, tt.expected.domain, domain)
        })
    }
}

func TestParseEmailAddress_Invalid(t *testing.T) {
    invalidEmails := []string{
        "invalid",
        "no-at-sign",
        "@nodomain.com",
        "nolocal@",
        "",
    }

    for _, email := range invalidEmails {
        t.Run(email, func(t *testing.T) {
            _, _, err := ParseEmailAddress(email)
            assert.Error(t, err)
        })
    }
}
```

### 6. E2E Test - Full Email Flow

**File:** `tests/e2e/email_flow_test.go`

```go
//go:build e2e

package e2e

import (
    "context"
    "net"
    "net/smtp"
    "testing"
    "time"

    "github.com/stretchr/testify/suite"
    "github.com/gorilla/websocket"
)

type EmailFlowE2ESuite struct {
    suite.Suite
    smtpAddr string
    apiAddr  string
    wsAddr   string
}

func (s *EmailFlowE2ESuite) SetupSuite() {
    // These should be configured via environment variables
    s.smtpAddr = "localhost:2525"
    s.apiAddr = "http://localhost:8080"
    s.wsAddr = "ws://localhost:8080/ws"
}

func TestEmailFlowE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    suite.Run(t, new(EmailFlowE2ESuite))
}

func (s *EmailFlowE2ESuite) TestCompleteEmailReceivingFlow() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 1. Connect to WebSocket for real-time notifications
    wsConn, _, err := websocket.DefaultDialer.Dial(s.wsAddr+"/mailbox/test@example.com", nil)
    s.Require().NoError(err)
    defer wsConn.Close()

    notificationReceived := make(chan bool, 1)
    go func() {
        _, _, err := wsConn.ReadMessage()
        if err == nil {
            notificationReceived <- true
        }
    }()

    // 2. Send email via SMTP
    msg := []byte("To: test@example.com\r\n" +
        "Subject: E2E Test Email\r\n" +
        "\r\n" +
        "This is an E2E test email.\r\n")

    err = smtp.SendMail(s.smtpAddr, nil, "sender@external.com", []string{"test@example.com"}, msg)
    s.Require().NoError(err)

    // 3. Wait for WebSocket notification
    select {
    case <-notificationReceived:
        // Success - notification received
    case <-ctx.Done():
        s.Fail("Timeout waiting for WebSocket notification")
    }

    // 4. Verify email was stored via API
    // (Add HTTP client calls to verify message exists)
}

func (s *EmailFlowE2ESuite) TestSMTPServerAcceptsConnections() {
    conn, err := net.DialTimeout("tcp", s.smtpAddr, 5*time.Second)
    s.Require().NoError(err)
    defer conn.Close()

    // Read SMTP banner
    buf := make([]byte, 256)
    n, err := conn.Read(buf)
    s.Require().NoError(err)
    s.Contains(string(buf[:n]), "220")
}
```

---

## 🚀 Makefile untuk Testing

**File:** `Makefile`

```makefile
.PHONY: test test-unit test-integration test-e2e test-coverage test-verbose lint

# Default test command
test:
	go test ./... -v

# Unit tests only (fast)
test-unit:
	go test ./internal/... -v -short

# Integration tests (requires Docker)
test-integration:
	go test ./tests/integration/... -v -tags=integration

# E2E tests (requires running server)
test-e2e:
	go test ./tests/e2e/... -v -tags=e2e

# Test with coverage report
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with race detector
test-race:
	go test ./... -race

# Run specific test
test-run:
	go test ./... -v -run $(TEST)

# Generate mocks
generate-mocks:
	mockgen -source=internal/repository/domain_repository.go -destination=tests/mocks/domain_repository_mock.go -package=mocks
	mockgen -source=internal/repository/mailbox_repository.go -destination=tests/mocks/mailbox_repository_mock.go -package=mocks
	mockgen -source=internal/repository/message_repository.go -destination=tests/mocks/message_repository_mock.go -package=mocks
	mockgen -source=internal/repository/attachment_repository.go -destination=tests/mocks/attachment_repository_mock.go -package=mocks

# Lint
lint:
	golangci-lint run

# Benchmark tests
bench:
	go test ./... -bench=. -benchmem

# Clean test artifacts
clean-test:
	rm -f coverage.out coverage.html
	go clean -testcache
```

---

## 📊 CI/CD Integration

**File:** `.github/workflows/test.yml`

```yaml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run unit tests
        run: go test ./internal/... -v -short -coverprofile=coverage.out
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_DB: testdb
          POSTGRES_USER: testuser
          POSTGRES_PASSWORD: testpass
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run integration tests
        env:
          DATABASE_URL: postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable
        run: go test ./tests/integration/... -v -tags=integration

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

---

## 📈 Testing Metrics & Goals

| Metric | Target | Current |
|--------|--------|---------|
| Unit Test Coverage | 80% | - |
| Integration Test Coverage | 70% | - |
| Test Execution Time (Unit) | < 30s | - |
| Test Execution Time (Integration) | < 5min | - |
| Flaky Test Rate | < 1% | - |

---

## 🎯 Prioritas Implementasi

### Phase 1: Foundation (Week 1)
- [ ] Setup testing dependencies
- [ ] Create mock interfaces
- [ ] Implement handler unit tests
- [ ] Implement repository unit tests

### Phase 2: Integration (Week 2)
- [ ] Setup Docker test environment
- [ ] Database integration tests
- [ ] SMTP integration tests
- [ ] API integration tests

### Phase 3: E2E & Polish (Week 3)
- [ ] E2E test suite
- [ ] CI/CD pipeline
- [ ] Coverage reporting
- [ ] Performance benchmarks

---

## 💡 Best Practices

1. **Test Naming**: Use `Test<Function>_<Scenario>` pattern
2. **Table-Driven Tests**: For multiple input/output scenarios
3. **Test Isolation**: Each test should be independent
4. **Clean Data**: Use `SetupTest`/`TearDownTest` for data cleanup
5. **Parallel Tests**: Use `t.Parallel()` where safe
6. **Error Messages**: Provide clear assertion messages
7. **Golden Files**: For complex output validation
8. **Test Tags**: Use build tags to separate test types
