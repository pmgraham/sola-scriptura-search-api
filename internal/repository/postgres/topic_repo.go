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
// Matches on topic and sub_topic columns for better relevance
func (r *TopicRepository) SearchByWords(ctx context.Context, words []string, topK int) ([]models.TopicSearchResult, error) {
	if len(words) == 0 {
		return []models.TopicSearchResult{}, nil
	}

	// Build scoring CASE for each word
	// Prioritize: exact topic match > topic prefix > sub_topic match > name contains
	scoreCases := ""
	for i := range words {
		if i > 0 {
			scoreCases += ",\n\t\t\t   "
		}
		paramNum := i + 1
		// Strip wildcards for scoring comparison (args have %word%)
		scoreCases += fmt.Sprintf(`CASE
			   WHEN LOWER(topic) = LOWER(TRIM('%%' FROM $%d)) THEN 1.0
			   WHEN LOWER(topic) LIKE LOWER(TRIM('%%' FROM $%d)) || '%%' THEN 0.95
			   WHEN LOWER(sub_topic) = LOWER(TRIM('%%' FROM $%d)) THEN 0.9
			   WHEN topic ILIKE $%d OR sub_topic ILIKE $%d THEN 0.85
			   WHEN name ILIKE $%d THEN 0.7
			   ELSE 0.0
		       END`, paramNum, paramNum, paramNum, paramNum, paramNum, paramNum)
	}

	// Use mv_topics_summary which has pre-computed verse_count
	// Match on topic, sub_topic, or name columns
	query := fmt.Sprintf(`
		SELECT topic_id::text, name, source, COALESCE(category, '') as category, verse_count,
		       GREATEST(%s) as score
		FROM api_views.mv_topics_summary
		WHERE `, scoreCases)

	args := make([]interface{}, 0, len(words)+1)
	for i, word := range words {
		if i > 0 {
			query += " OR "
		}
		query += fmt.Sprintf("(topic ILIKE $%d OR sub_topic ILIKE $%d OR name ILIKE $%d)", i+1, i+1, i+1)
		args = append(args, "%"+word+"%")
	}
	args = append(args, topK)

	query += fmt.Sprintf(`
		GROUP BY topic_id, name, source, category, topic, sub_topic, verse_count
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
			Category   string  `db:"category"`
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
				TopicID:  result.TopicID,
				Name:     result.Name,
				Source:   source,
				Category: result.Category,
			},
			Score:      result.Score,
			VerseCount: result.VerseCount,
			Category:   result.Category,
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

// GetTopicVerses returns verses mapped to a topic
func (r *TopicRepository) GetTopicVerses(ctx context.Context, topicID string, limit int) ([]models.Citation, error) {
	query := `
		SELECT v.osis_verse_id as verse_id, v.text, b.osis_id as book, v.chapter, v.verse
		FROM api.topic_verses tv
		JOIN api.verses v ON tv.verse_id = v.id
		JOIN api.books b ON v.book_id = b.id
		WHERE tv.topic_id = $1
		ORDER BY tv.importance_tier, b.book_order, v.chapter, v.verse
		LIMIT $2
	`

	var verses []models.Citation
	if err := r.db.SelectContext(ctx, &verses, query, topicID, limit); err != nil {
		return nil, fmt.Errorf("get topic verses: %w", err)
	}

	if verses == nil {
		verses = []models.Citation{}
	}
	return verses, nil
}
