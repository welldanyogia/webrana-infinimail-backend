package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// Health handles GET /health
func (h *HealthHandler) Health(c echo.Context) error {
	services := make(map[string]string)
	status := "healthy"

	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		services["database"] = "unhealthy"
		status = "unhealthy"
	} else if err := sqlDB.Ping(); err != nil {
		services["database"] = "unhealthy"
		status = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, HealthResponse{
		Status:   status,
		Services: services,
	})
}

// Ready handles GET /ready
func (h *HealthHandler) Ready(c echo.Context) error {
	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"reason": "database connection failed",
		})
	}

	if err := sqlDB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"reason": "database ping failed",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
	})
}
