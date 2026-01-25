package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sola-scriptura-search-api/internal/config"
)

// CORSMiddleware returns a configured CORS middleware
func CORSMiddleware() echo.MiddlewareFunc {
	cfg := config.GetConfig()

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	})
}
