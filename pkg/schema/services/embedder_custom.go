package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sola-scriptura-search-api/pkg/schema/config"
)

// CustomEmbedder implements Embedder using a custom HTTP embedding service
type CustomEmbedder struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewCustomEmbedder creates a new custom HTTP embedder
func NewCustomEmbedder(cfg *config.Config) *CustomEmbedder {
	return &CustomEmbedder{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

var taskTypeToInstruction = map[TaskType]string{
	TaskTypeQuery:    "Represent the question for retrieving relevant Bible verses: ",
	TaskTypeDocument: "Represent the Bible verse for retrieval: ",
}

type customEmbeddingRequest struct {
	Text        string `json:"text"`
	Instruction string `json:"instruction"`
}

type customEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

type customBatchEmbeddingRequest struct {
	Texts       []string `json:"texts"`
	Instruction string   `json:"instruction"`
}

type customBatchEmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// Embed generates an embedding for a single text
func (e *CustomEmbedder) Embed(ctx context.Context, text string, taskType TaskType) ([]float64, error) {
	instruction := taskTypeToInstruction[taskType]
	if instruction == "" {
		instruction = taskTypeToInstruction[TaskTypeDocument]
	}

	url := e.cfg.EmbeddingServiceURL + "/embed"

	reqBody := customEmbeddingRequest{
		Text:        text,
		Instruction: instruction,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service error: %s", string(body))
	}

	var embResp customEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *CustomEmbedder) EmbedBatch(ctx context.Context, texts []string, taskType TaskType) ([][]float64, error) {
	instruction := taskTypeToInstruction[taskType]
	if instruction == "" {
		instruction = taskTypeToInstruction[TaskTypeDocument]
	}

	url := e.cfg.EmbeddingServiceURL + "/embed/batch"

	reqBody := customBatchEmbeddingRequest{
		Texts:       texts,
		Instruction: instruction,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service error: %s", string(body))
	}

	var batchResp customBatchEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return batchResp.Embeddings, nil
}
