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
			Category:    r.Category,
			ChapterRefs: r.Topic.ChapterRefs,
			VerseCount:  r.VerseCount,
			Score:       r.Score,
		}
	}
	return topics, nil
}

// preferredSources defines source priority for topic cards (higher index = lower priority)
var preferredSources = []string{
	"claude_4.5_opus",
	"torreys_topical_textbook",
	"naves_topical_bible",
}

// GetTopicCard returns a TopicCard for the best matching topic if score is high enough
// Prefers Claude-curated topics over other sources when available
func (s *VectorSearchService) GetTopicCard(ctx context.Context, topics []models.ScoredTopic, minScore float64, verseLimit int) (*models.TopicCard, error) {
	if len(topics) == 0 {
		return nil, nil
	}

	// Find the best topic: prefer Claude source, then by score
	var selectedTopic *models.ScoredTopic

	// First pass: look for preferred sources in order
	for _, preferredSource := range preferredSources {
		for i := range topics {
			if topics[i].Source == preferredSource && topics[i].Score >= minScore {
				selectedTopic = &topics[i]
				break
			}
		}
		if selectedTopic != nil {
			break
		}
	}

	// Fallback: use highest scoring topic if no preferred source found
	if selectedTopic == nil {
		if topics[0].Score >= minScore {
			selectedTopic = &topics[0]
		}
	}

	if selectedTopic == nil {
		return nil, nil
	}

	// Fetch verses for this topic
	verses, err := s.topicRepo.GetTopicVerses(ctx, selectedTopic.TopicID, verseLimit)
	if err != nil {
		return nil, err
	}

	return &models.TopicCard{
		TopicID:    selectedTopic.TopicID,
		Name:       selectedTopic.Name,
		Category:   selectedTopic.Category,
		Source:     selectedTopic.Source,
		VerseCount: selectedTopic.VerseCount,
		Score:      selectedTopic.Score,
		TopVerses:  verses,
	}, nil
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
