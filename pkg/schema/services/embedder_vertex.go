package services

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/sola-scriptura-search-api/pkg/schema/config"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	vertexBatchLimit = 250
)

// VertexEmbedder implements Embedder using Google Cloud Vertex AI
type VertexEmbedder struct {
	cfg      *config.Config
	client   *aiplatform.PredictionClient
	endpoint string
}

// NewVertexEmbedder creates a new Vertex AI embedder
func NewVertexEmbedder(ctx context.Context, cfg *config.Config) (*VertexEmbedder, error) {
	if cfg.GCPProjectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID is required for Vertex AI embeddings")
	}

	clientEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", cfg.GCPLocation)
	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(clientEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		cfg.GCPProjectID, cfg.GCPLocation, cfg.VertexModel)

	return &VertexEmbedder{
		cfg:      cfg,
		client:   client,
		endpoint: endpoint,
	}, nil
}

// Close closes the Vertex AI client
func (e *VertexEmbedder) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// Embed generates an embedding for a single text
func (e *VertexEmbedder) Embed(ctx context.Context, text string, taskType TaskType) ([]float64, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text}, taskType)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *VertexEmbedder) EmbedBatch(ctx context.Context, texts []string, taskType TaskType) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	if len(texts) > vertexBatchLimit {
		var allEmbeddings [][]float64
		for i := 0; i < len(texts); i += vertexBatchLimit {
			end := i + vertexBatchLimit
			if end > len(texts) {
				end = len(texts)
			}
			batch, err := e.embedBatchInternal(ctx, texts[i:end], taskType)
			if err != nil {
				return nil, err
			}
			allEmbeddings = append(allEmbeddings, batch...)
		}
		return allEmbeddings, nil
	}

	return e.embedBatchInternal(ctx, texts, taskType)
}

func (e *VertexEmbedder) embedBatchInternal(ctx context.Context, texts []string, taskType TaskType) ([][]float64, error) {
	instances := make([]*structpb.Value, len(texts))
	for i, text := range texts {
		instance, err := structpb.NewStruct(map[string]interface{}{
			"content":   text,
			"task_type": string(taskType),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create instance: %w", err)
		}
		instances[i] = structpb.NewStructValue(instance)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:  e.endpoint,
		Instances: instances,
	}

	resp, err := e.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vertex AI prediction failed: %w", err)
	}

	embeddings := make([][]float64, len(resp.Predictions))
	for i, prediction := range resp.Predictions {
		predStruct := prediction.GetStructValue()
		if predStruct == nil {
			return nil, fmt.Errorf("unexpected prediction format at index %d", i)
		}

		embeddingsField := predStruct.Fields["embeddings"]
		if embeddingsField == nil {
			return nil, fmt.Errorf("no embeddings field in prediction at index %d", i)
		}

		embStruct := embeddingsField.GetStructValue()
		if embStruct == nil {
			return nil, fmt.Errorf("unexpected embeddings format at index %d", i)
		}

		valuesField := embStruct.Fields["values"]
		if valuesField == nil {
			return nil, fmt.Errorf("no values field in embeddings at index %d", i)
		}

		valuesList := valuesField.GetListValue()
		if valuesList == nil {
			return nil, fmt.Errorf("unexpected values format at index %d", i)
		}

		embedding := make([]float64, len(valuesList.Values))
		for j, v := range valuesList.Values {
			embedding[j] = v.GetNumberValue()
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}
