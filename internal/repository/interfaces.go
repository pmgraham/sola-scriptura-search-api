package repository

import (
	"context"

	"github.com/sola-scriptura-search-api/internal/models"
)

// VectorSearchRepository defines operations for vector similarity search
type VectorSearchRepository interface {
	// SearchVersesByEmbedding performs vector similarity search on verses
	SearchVersesByEmbedding(ctx context.Context, embedding []float64, topK int) ([]models.ScoredVerse, error)
}

// TopicRepository defines operations for topical index data access
type TopicRepository interface {
	// SearchByWords searches topics by keyword matching
	SearchByWords(ctx context.Context, words []string, topK int) ([]models.TopicSearchResult, error)
}
