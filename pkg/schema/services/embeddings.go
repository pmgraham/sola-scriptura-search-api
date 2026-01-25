package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/sola-scriptura-search-api/pkg/schema/config"
)

// EmbeddingsService handles text embedding operations using a pluggable backend
type EmbeddingsService struct {
	embedder Embedder
}

var (
	embeddingsService *EmbeddingsService
	embeddingsOnce    sync.Once
	initErr           error
)

// GetEmbeddingsService returns the singleton embeddings service
func GetEmbeddingsService() *EmbeddingsService {
	embeddingsOnce.Do(func() {
		cfg := config.GetConfig()
		ctx := context.Background()

		var embedder Embedder
		switch cfg.EmbeddingProvider {
		case "vertex":
			var err error
			embedder, err = NewVertexEmbedder(ctx, cfg)
			if err != nil {
				initErr = fmt.Errorf("failed to create Vertex AI embedder: %w", err)
				return
			}
		default:
			embedder = NewCustomEmbedder(cfg)
		}

		embeddingsService = &EmbeddingsService{
			embedder: embedder,
		}
	})
	return embeddingsService
}

// GetInitError returns any error that occurred during initialization
func GetInitError() error {
	return initErr
}

// EmbedQuery embeds a query for retrieval
func (s *EmbeddingsService) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	return s.embedder.Embed(ctx, query, TaskTypeQuery)
}

// EmbedVerse embeds a verse as a document for retrieval
func (s *EmbeddingsService) EmbedVerse(ctx context.Context, text string) ([]float64, error) {
	return s.embedder.Embed(ctx, text, TaskTypeDocument)
}
