package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
	"github.com/sola-scriptura-search-api/internal/models"
	"github.com/sola-scriptura-search-api/internal/repository"
)

// VectorSearchRepository implements repository.VectorSearchRepository for PostgreSQL with pgvector
type VectorSearchRepository struct {
	db *sqlx.DB
}

// NewVectorSearchRepository creates a new PostgreSQL vector search repository
func NewVectorSearchRepository(db *sqlx.DB) repository.VectorSearchRepository {
	return &VectorSearchRepository{db: db}
}

// SearchVersesByEmbedding performs vector similarity search on verses using pgvector
func (r *VectorSearchRepository) SearchVersesByEmbedding(ctx context.Context, embedding []float64, topK int) ([]models.ScoredVerse, error) {
	vec := pgvector.NewVector(float32Slice(embedding))

	rows, err := r.db.QueryxContext(ctx, `
		SELECT v.osis_verse_id as verse_id, b.osis_id as book, v.chapter, v.verse, v.text,
		       1 - (v.embedding <=> $1::vector) as score
		FROM verses v
		JOIN books b ON v.book_id = b.id
		WHERE v.embedding IS NOT NULL
		ORDER BY v.embedding <=> $1::vector
		LIMIT $2
	`, vec, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search verses: %w", err)
	}
	defer rows.Close()

	var results []models.ScoredVerse
	for rows.Next() {
		var v models.ScoredVerse
		if err := rows.Scan(&v.VerseID, &v.Book, &v.Chapter, &v.Verse, &v.Text, &v.Score); err != nil {
			return nil, fmt.Errorf("scan verse result: %w", err)
		}
		results = append(results, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verse results: %w", err)
	}

	if results == nil {
		results = []models.ScoredVerse{}
	}
	return results, nil
}

// float32Slice converts []float64 to []float32 for pgvector
func float32Slice(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}
