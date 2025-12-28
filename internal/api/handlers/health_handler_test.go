package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupHealthTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	// GORM pings during initialization
	mock.ExpectPing()

	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return gormDB, mock, cleanup
}

func TestHealthHandler_Health_ReturnsOKWhenHealthy(t *testing.T) {
	gormDB, mock, cleanup := setupHealthTestDB(t)
	defer cleanup()

	// Expect ping to succeed during health check
	mock.ExpectPing()

	handler := NewHealthHandler(gormDB)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Health(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"healthy"`)
	assert.Contains(t, rec.Body.String(), `"database":"healthy"`)
}

func TestHealthHandler_Health_ReturnsServiceUnavailableWhenUnhealthy(t *testing.T) {
	gormDB, mock, cleanup := setupHealthTestDB(t)
	defer cleanup()

	// Expect ping to fail during health check
	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	handler := NewHealthHandler(gormDB)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Health(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"unhealthy"`)
	assert.Contains(t, rec.Body.String(), `"database":"unhealthy"`)
}

func TestHealthHandler_Ready_ReturnsOKWhenReady(t *testing.T) {
	gormDB, mock, cleanup := setupHealthTestDB(t)
	defer cleanup()

	// Expect ping to succeed during ready check
	mock.ExpectPing()

	handler := NewHealthHandler(gormDB)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Ready(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ready"`)
}

func TestHealthHandler_Ready_ReturnsServiceUnavailableWhenNotReady(t *testing.T) {
	gormDB, mock, cleanup := setupHealthTestDB(t)
	defer cleanup()

	// Expect ping to fail during ready check
	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	handler := NewHealthHandler(gormDB)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Ready(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"not ready"`)
	assert.Contains(t, rec.Body.String(), `"reason":"database ping failed"`)
}

func TestNewHealthHandler_CreatesHandler(t *testing.T) {
	gormDB, _, cleanup := setupHealthTestDB(t)
	defer cleanup()

	handler := NewHealthHandler(gormDB)

	assert.NotNil(t, handler)
	assert.Equal(t, gormDB, handler.db)
}

func TestHealthResponse_Structure(t *testing.T) {
	response := HealthResponse{
		Status: "healthy",
		Services: map[string]string{
			"database": "healthy",
		},
	}

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "healthy", response.Services["database"])
}
