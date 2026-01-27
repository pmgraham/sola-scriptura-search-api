package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

// SearchByWords searches topics by keyword matching
func (r *TopicRepository) SearchByWords(ctx context.Context, words []string, topK int) ([]models.TopicSearchResult, error) {
	if len(words) == 0 {
		return []models.TopicSearchResult{}, nil
	}

	query := `
		WITH matched_topics AS (
			SELECT t.topic_id, t.name, t.source, t.chapter_refs,
			       MAX(CASE
			           WHEN LOWER(t.name) = LOWER($1) THEN 1.0
			           WHEN LOWER(t.name) LIKE LOWER($1) || '%' THEN 0.9
			           ELSE 0.7
			       END) as score
			FROM topics t
			WHERE `

	args := make([]interface{}, 0, len(words)+1)
	for i, word := range words {
		if i > 0 {
			query += " OR "
		}
		query += fmt.Sprintf("t.name ILIKE $%d", i+1)
		args = append(args, "%"+word+"%")
	}
	args = append(args, topK)

	query += fmt.Sprintf(`
			GROUP BY t.topic_id, t.name, t.source, t.chapter_refs
		)
		SELECT mt.topic_id, mt.name, mt.source, mt.chapter_refs,
		       COUNT(tv.verse_id) as verse_count, mt.score
		FROM matched_topics mt
		LEFT JOIN topic_verses tv ON mt.topic_id = tv.topic_id
		GROUP BY mt.topic_id, mt.name, mt.source, mt.chapter_refs, mt.score
		HAVING COUNT(tv.verse_id) > 0
		ORDER BY mt.score DESC, COUNT(tv.verse_id) DESC
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
			TopicID     string         `db:"topic_id"`
			Name        string         `db:"name"`
			Source      string         `db:"source"`
			ChapterRefs pq.StringArray `db:"chapter_refs"`
			VerseCount  int            `db:"verse_count"`
			Score       float64        `db:"score"`
		}
		if err := rows.StructScan(&result); err != nil {
			return nil, fmt.Errorf("scan topic result: %w", err)
		}
		results = append(results, models.TopicSearchResult{
			Topic: models.Topic{
				TopicID:     result.TopicID,
				Name:        result.Name,
				Source:      result.Source,
				ChapterRefs: result.ChapterRefs,
			},
			Score:      result.Score,
			VerseCount: result.VerseCount,
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
