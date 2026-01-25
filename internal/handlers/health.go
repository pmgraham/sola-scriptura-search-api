package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sola-scriptura-search-api/pkg/schema/db"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse is the response for basic health check
type HealthResponse struct {
	Status string `json:"status"`
}

// DatabaseHealthResponse is the response for database health check
type DatabaseHealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// Health handles GET /health
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status: "healthy",
	})
}

// PostgresHealth handles GET /health/postgres
func (h *HealthHandler) PostgresHealth(c echo.Context) error {
	if !db.PostgresEnabled() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not_configured",
			"error":  "PostgreSQL is not configured",
		})
	}

	pgDB := db.GetPostgres()
	if pgDB == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "error",
			"error":  "PostgreSQL connection not available",
		})
	}

	if err := pgDB.PingContext(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
	}

	return c.JSON(http.StatusOK, DatabaseHealthResponse{
		Status:   "connected",
		Database: "postgres",
	})
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/health", h.Health)
	g.GET("/health/postgres", h.PostgresHealth)
}
