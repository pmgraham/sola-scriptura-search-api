package models

// Citation represents a cited verse with relevance score
type Citation struct {
	VerseID        string   `json:"verse_id" db:"verse_id"`
	Text           string   `json:"text" db:"text"`
	Book           string   `json:"book" db:"book"`
	Chapter        int      `json:"chapter" db:"chapter"`
	Verse          int      `json:"verse" db:"verse"`
	RelevanceScore *float64 `json:"relevance_score,omitempty" db:"relevance_score"`
}

// ScoredVerse represents a verse with similarity score
type ScoredVerse struct {
	VerseID string  `json:"verse_id"`
	Book    string  `json:"book"`
	Chapter int     `json:"chapter"`
	Verse   int     `json:"verse"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
}

// ScoredTopic represents a topic with relevance score
type ScoredTopic struct {
	TopicID      string   `json:"topic_id"`
	Name         string   `json:"name"`
	Source       string   `json:"source"`
	Category     string   `json:"category,omitempty"`
	ChapterRefs  []string `json:"chapter_refs,omitempty"`
	VerseCount   int      `json:"verse_count"`
	Score        float64  `json:"score"`
	MatchedWords []string `json:"matched_words,omitempty"`
}

// Topic represents a topical index entry
type Topic struct {
	TopicID     string   `json:"topic_id"`
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Category    string   `json:"category,omitempty"`
	ChapterRefs []string `json:"chapter_refs,omitempty"`
}

// TopicSearchResult wraps a topic with search score
type TopicSearchResult struct {
	Topic      Topic   `json:"topic"`
	Score      float64 `json:"score"`
	VerseCount int     `json:"verse_count"`
	Category   string  `json:"category,omitempty"`
}

// SemanticSearchRequest is the request for semantic search
type SemanticSearchRequest struct {
	Query string `json:"query" validate:"required"`
	Limit int    `json:"limit" validate:"min=1,max=50"`
}

// SemanticSearchResponse is the response for semantic search
type SemanticSearchResponse struct {
	Query   string     `json:"query"`
	Results []Citation `json:"results"`
}

// HybridSearchRequest is the request for hybrid search
type HybridSearchRequest struct {
	Query      string `json:"query" validate:"required"`
	VerseLimit int    `json:"verse_limit" validate:"min=1,max=50"`
	TopicLimit int    `json:"topic_limit" validate:"min=1,max=50"`
}

// ResourceMatches contains results from curated sources
type ResourceMatches struct {
	Topics []ScoredTopic `json:"topics,omitempty"`
}

// SemanticMatches contains results from embedding-based search
type SemanticMatches struct {
	Verses []Citation `json:"verses"`
}

// TopicCard represents a featured topic with its key verses
type TopicCard struct {
	TopicID    string     `json:"topic_id"`
	Name       string     `json:"name"`
	Category   string     `json:"category,omitempty"`
	Source     string     `json:"source,omitempty"`
	VerseCount int        `json:"verse_count"`
	Score      float64    `json:"score"`
	TopVerses  []Citation `json:"top_verses"`
}

// HybridSearchResponse is the response for hybrid search
type HybridSearchResponse struct {
	Query           string          `json:"query"`
	TopicCard       *TopicCard      `json:"topic_card,omitempty"`
	ResourceMatches ResourceMatches `json:"resource_matches"`
	SemanticMatches SemanticMatches `json:"semantic_matches"`
}
