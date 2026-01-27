package services

import (
	"context"
	"strings"

	"github.com/sola-scriptura-search-api/internal/models"
	"github.com/sola-scriptura-search-api/internal/repository"
	pkgservices "github.com/sola-scriptura-search-api/pkg/schema/services"
)

// VectorSearchService handles semantic search using PostgreSQL with pgvector
type VectorSearchService struct {
	vectorRepo    repository.VectorSearchRepository
	topicRepo     repository.TopicRepository
	embeddingsSvc *pkgservices.EmbeddingsService
}

// NewVectorSearchService creates a new vector search service
func NewVectorSearchService(
	vectorRepo repository.VectorSearchRepository,
	topicRepo repository.TopicRepository,
	embeddingsSvc *pkgservices.EmbeddingsService,
) *VectorSearchService {
	return &VectorSearchService{
		vectorRepo:    vectorRepo,
		topicRepo:     topicRepo,
		embeddingsSvc: embeddingsSvc,
	}
}

// SearchVerses embeds a query and performs vector search
func (s *VectorSearchService) SearchVerses(ctx context.Context, query string, topK int) ([]models.ScoredVerse, error) {
	embedding, err := s.embeddingsSvc.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return s.vectorRepo.SearchVersesByEmbedding(ctx, embedding, topK)
}

// SearchVersesCitations performs vector search and returns as citations
func (s *VectorSearchService) SearchVersesCitations(ctx context.Context, query string, topK int) ([]models.Citation, error) {
	scoredVerses, err := s.SearchVerses(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	citations := make([]models.Citation, len(scoredVerses))
	for i, v := range scoredVerses {
		score := v.Score
		citations[i] = models.Citation{
			VerseID:        v.VerseID,
			Text:           v.Text,
			Book:           v.Book,
			Chapter:        v.Chapter,
			Verse:          v.Verse,
			RelevanceScore: &score,
		}
	}
	return citations, nil
}

// SearchTopics searches topics by keywords
func (s *VectorSearchService) SearchTopics(ctx context.Context, query string, topK int) ([]models.ScoredTopic, error) {
	words := tokenizeWords(query)
	if len(words) == 0 {
		return []models.ScoredTopic{}, nil
	}

	results, err := s.topicRepo.SearchByWords(ctx, words, topK)
	if err != nil {
		return nil, err
	}

	topics := make([]models.ScoredTopic, len(results))
	for i, r := range results {
		topics[i] = models.ScoredTopic{
			TopicID:     r.Topic.TopicID,
			Name:        r.Topic.Name,
			Source:      r.Topic.Source,
			ChapterRefs: r.Topic.ChapterRefs,
			VerseCount:  r.VerseCount,
			Score:       r.Score,
		}
	}
	return topics, nil
}

// stopWords contains common words to exclude from search
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "with": true,
	"this": true, "are": true, "but": true, "not": true, "you": true,
	"all": true, "was": true, "his": true, "her": true, "from": true,
	"they": true, "have": true, "had": true, "been": true, "were": true,
	"will": true, "would": true, "could": true, "should": true, "shall": true,
	"unto": true, "them": true, "which": true, "there": true, "their": true,
	"when": true, "then": true, "than": true, "into": true, "upon": true,
}

// tokenizeWords splits query into searchable words
func tokenizeWords(query string) []string {
	words := strings.FieldsFunc(strings.ToLower(query), func(c rune) bool {
		return !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))
	})

	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if len(word) >= 2 && !stopWords[word] {
			filtered = append(filtered, word)
		}
	}
	return filtered
}
