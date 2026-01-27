package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/sola-scriptura-search-api/internal/config"
	"github.com/sola-scriptura-search-api/internal/handlers"
	"github.com/sola-scriptura-search-api/internal/middleware"
	"github.com/sola-scriptura-search-api/internal/repository"
	"github.com/sola-scriptura-search-api/internal/repository/postgres"
	"github.com/sola-scriptura-search-api/internal/repository/vertex"
	"github.com/sola-scriptura-search-api/internal/services"
	"github.com/sola-scriptura-search-api/pkg/schema/db"
	pkgservices "github.com/sola-scriptura-search-api/pkg/schema/services"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Get configuration
	cfg := config.GetConfig()

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(middleware.CORSMiddleware())

	// Initialize PostgreSQL
	ctx := context.Background()
	if err := db.InitPostgres(ctx); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	log.Println("Database initialization complete")

	// Create repositories
	pgDB := db.GetPostgres()
	topicRepo := postgres.NewTopicRepository(pgDB)

	// Create vector search repository based on configuration
	var vectorRepo repository.VectorSearchRepository
	var vertexRepo *vertex.VectorSearchRepository // For cleanup

	switch cfg.VectorBackend {
	case "vertex":
		log.Println("Using Vertex AI Vector Search backend")
		vertexCfg := vertex.Config{
			ProjectID:            cfg.VertexProjectID,
			Location:             cfg.VertexLocation,
			IndexEndpointID:      cfg.VertexIndexEndpointID,
			DeployedIndexID:      cfg.VertexDeployedIndexID,
			PublicEndpointDomain: cfg.VertexPublicEndpointDomain,
		}
		var err error
		vertexRepo, err = vertex.NewVectorSearchRepository(ctx, vertexCfg, pgDB)
		if err != nil {
			log.Fatalf("Failed to create Vertex AI vector repository: %v", err)
		}
		vectorRepo = vertexRepo
	default:
		log.Println("Using pgvector backend (unindexed)")
		vectorRepo = postgres.NewVectorSearchRepository(pgDB)
	}

	// Create services
	embeddingsSvc := pkgservices.GetEmbeddingsService()
	if err := pkgservices.GetInitError(); err != nil {
		log.Fatalf("Failed to initialize embeddings service: %v", err)
	}

	vectorSearchSvc := services.NewVectorSearchService(vectorRepo, topicRepo, embeddingsSvc)

	// Create API group with prefix
	api := e.Group(cfg.APIPrefix)

	// Register handlers
	healthHandler := handlers.NewHealthHandler()
	healthHandler.RegisterRoutes(api)

	searchHandler := handlers.NewSearchHandler(vectorSearchSvc)
	searchHandler.RegisterRoutes(api)

	// Root health check
	e.GET("/", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"name":    cfg.APITitle,
			"version": cfg.APIVersion,
			"status":  "running",
		})
	})

	// Start server
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Starting %s v%s on %s", cfg.APITitle, cfg.APIVersion, addr)
		if err := e.Start(addr); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	if err := db.ClosePostgres(); err != nil {
		log.Printf("Error closing PostgreSQL: %v", err)
	}

	// Close Vertex AI client if used
	if vertexRepo != nil {
		if err := vertexRepo.Close(); err != nil {
			log.Printf("Error closing Vertex AI client: %v", err)
		}
	}

	log.Println("Server stopped")
}
