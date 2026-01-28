package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	pkgservices "github.com/sola-scriptura-search-api/pkg/schema/services"
	"google.golang.org/api/option"
)

// EnrichmentResult matches the structure from main.go
type EnrichmentResult struct {
	Verse struct {
		VerseID string `json:"osis_verse_id"`
		Text    string `json:"text"`
	} `json:"verse"`
	TheoAnnotations  []string `json:"theological_annotations"`
	SyntheticQueries []string `json:"synthetic_queries"`
	AugmentedText    string   `json:"augmented_text"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	godotenv.Load()
	ctx := context.Background()

	// Load enrichment results
	data, err := os.ReadFile("enrichment_results.json")
	if err != nil {
		return fmt.Errorf("read enrichment results: %w", err)
	}

	var results []EnrichmentResult
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("parse enrichment results: %w", err)
	}
	log.Printf("Loaded %d enrichment results\n", len(results))

	// Get embeddings service (uses existing config)
	embeddingSvc := pkgservices.GetEmbeddingsService()
	if err := pkgservices.GetInitError(); err != nil {
		return fmt.Errorf("init embeddings service: %w", err)
	}

	// Config for Vertex AI Index
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	if projectID == "" {
		projectID = "sola-scriptura-project-dev"
	}
	if location == "" {
		location = "us-central1"
	}

	indexID := os.Getenv("VERTEX_INDEX_ID")
	if indexID == "" {
		indexID = "4664508756049002496"
	}

	// Create index client for upserting
	indexEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	indexClient, err := aiplatform.NewIndexClient(ctx, option.WithEndpoint(indexEndpoint))
	if err != nil {
		return fmt.Errorf("create index client: %w", err)
	}
	defer indexClient.Close()

	indexName := fmt.Sprintf("projects/%s/locations/%s/indexes/%s", projectID, location, indexID)

	// Process each result - generate embeddings
	var datapoints []*aiplatformpb.IndexDatapoint
	for i, result := range results {
		log.Printf("[%d/%d] Embedding %s...\n", i+1, len(results), result.Verse.VerseID)

		// Generate embedding for augmented text using existing service
		embedding, err := embeddingSvc.EmbedVerse(ctx, result.AugmentedText)
		if err != nil {
			log.Printf("  Warning: failed to embed %s: %v\n", result.Verse.VerseID, err)
			continue
		}

		// Convert []float64 to []float32 for Vertex AI
		embedding32 := make([]float32, len(embedding))
		for j, v := range embedding {
			embedding32[j] = float32(v)
		}

		// Create datapoint
		datapoint := &aiplatformpb.IndexDatapoint{
			DatapointId:   result.Verse.VerseID,
			FeatureVector: embedding32,
		}
		datapoints = append(datapoints, datapoint)

		log.Printf("  Embedded: %d dimensions\n", len(embedding))
	}

	// Upsert all datapoints
	log.Printf("Upserting %d datapoints to index...\n", len(datapoints))

	// Batch upsert (max 100 at a time)
	batchSize := 100
	for i := 0; i < len(datapoints); i += batchSize {
		end := i + batchSize
		if end > len(datapoints) {
			end = len(datapoints)
		}
		batch := datapoints[i:end]

		req := &aiplatformpb.UpsertDatapointsRequest{
			Index:      indexName,
			Datapoints: batch,
		}

		_, err := indexClient.UpsertDatapoints(ctx, req)
		if err != nil {
			return fmt.Errorf("upsert batch %d-%d: %w", i, end, err)
		}
		log.Printf("  Upserted batch %d-%d\n", i+1, end)
	}

	log.Println("Done! Enriched embeddings uploaded to Vertex AI.")
	return nil
}
