package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sola-scriptura-search-api/internal/models"
	"github.com/sola-scriptura-search-api/internal/repository"
)

// TopicRepository implements repository.TopicRepository for PostgreSQL
type TopicRepository struct {
	db *sqlx.DB
}

// NewTopicRepository creates a new PostgreSQL topic repository
func NewTopicRepository(db *sqlx.DB) repository.TopicRepository {
	return &TopicRepository{db: db}
}

// SearchByWords searches topics by keyword matching using mv_topics_summary
func (r *TopicRepository) SearchByWords(ctx context.Context, words []string, topK int) ([]models.TopicSearchResult, error) {
	if len(words) == 0 {
		return []models.TopicSearchResult{}, nil
	}

	// Use mv_topics_summary which has pre-computed verse_count
	query := `
		SELECT topic_id::text, name, source, verse_count,
		       MAX(CASE
		           WHEN LOWER(name) = LOWER($1) THEN 1.0
		           WHEN LOWER(name) LIKE LOWER($1) || '%' THEN 0.9
		           ELSE 0.7
		       END) as score
		FROM api_views.mv_topics_summary
		WHERE `

	args := make([]interface{}, 0, len(words)+1)
	for i, word := range words {
		if i > 0 {
			query += " OR "
		}
		query += fmt.Sprintf("name ILIKE $%d", i+1)
		args = append(args, "%"+word+"%")
	}
	args = append(args, topK)

	query += fmt.Sprintf(`
		GROUP BY topic_id, name, source, verse_count
		HAVING verse_count > 0
		ORDER BY score DESC, verse_count DESC
		LIMIT $%d
	`, len(words)+1)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search topics by words: %w", err)
	}
	defer rows.Close()

	var results []models.TopicSearchResult
	for rows.Next() {
		var result struct {
			TopicID    string  `db:"topic_id"`
			Name       string  `db:"name"`
			Source     *string `db:"source"`
			VerseCount int     `db:"verse_count"`
			Score      float64 `db:"score"`
		}
		if err := rows.StructScan(&result); err != nil {
			return nil, fmt.Errorf("scan topic result: %w", err)
		}
		source := ""
		if result.Source != nil {
			source = *result.Source
		}
		results = append(results, models.TopicSearchResult{
			Topic: models.Topic{
				TopicID: result.TopicID,
				Name:    result.Name,
				Source:  source,
			},
			Score: result.Score,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic results: %w", err)
	}

	if results == nil {
		results = []models.TopicSearchResult{}
	}
	return results, nil
}
