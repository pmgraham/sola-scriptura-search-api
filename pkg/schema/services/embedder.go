package services

import "context"

// TaskType represents the type of embedding task for Vertex AI
type TaskType string

const (
	TaskTypeQuery    TaskType = "RETRIEVAL_QUERY"
	TaskTypeDocument TaskType = "RETRIEVAL_DOCUMENT"
)

// Embedder defines the interface for text embedding operations
type Embedder interface {
	// Embed generates an embedding for a single text with the given task type
	Embed(ctx context.Context, text string, taskType TaskType) ([]float64, error)

	// EmbedBatch generates embeddings for multiple texts with the given task type
	EmbedBatch(ctx context.Context, texts []string, taskType TaskType) ([][]float64, error)
}
