// upsert_embeddings.go
//
// This script streams verse embeddings from PostgreSQL to Vertex AI Vector Search
// using the UpsertDatapoints API for streaming updates.
//
// Prerequisites:
// 1. Create and deploy the index using setup_vertex_index.go
// 2. Set environment variables (see below)
//
// Environment variables:
//   POSTGRES_URI              - PostgreSQL connection string
//   GCP_PROJECT_ID            - Your GCP project ID
//   VERTEX_LOCATION           - Region (default: us-central1)
//   VERTEX_INDEX_ID           - The index ID to update
//
// Usage:
//   go run scripts/upsert_embeddings.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
)

const (
	batchSize = 100 // Number of datapoints per upsert request
)

func main() {
	godotenv.Load()

	postgresURI := os.Getenv("POSTGRES_URI")
	if postgresURI == "" {
		log.Fatal("POSTGRES_URI environment variable is required")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("VERTEX_PROJECT_ID")
	}
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID or VERTEX_PROJECT_ID environment variable is required")
	}

	location := os.Getenv("VERTEX_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	indexID := os.Getenv("VERTEX_INDEX_ID")
	if indexID == "" {
		log.Fatal("VERTEX_INDEX_ID environment variable is required")
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	db, err := sqlx.ConnectContext(ctx, "postgres", postgresURI)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create Vertex AI Index client
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	client, err := aiplatform.NewIndexClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		log.Fatalf("Failed to create index client: %v", err)
	}
	defer client.Close()

	indexName := fmt.Sprintf("projects/%s/locations/%s/indexes/%s", projectID, location, indexID)

	log.Printf("Upserting embeddings to index: %s", indexName)

	// Query all verses with embeddings
	rows, err := db.QueryxContext(ctx, `
		SELECT
			verse_id,
			book,
			embedding::text as embedding_text
		FROM api_views.mv_verses_search
		WHERE embedding IS NOT NULL
		ORDER BY book_order, chapter, verse
	`)
	if err != nil {
		log.Fatalf("Failed to query verses: %v", err)
	}
	defer rows.Close()

	var batch []*aiplatformpb.IndexDatapoint
	totalCount := 0
	batchCount := 0

	for rows.Next() {
		var verseID, book, embeddingText string
		if err := rows.Scan(&verseID, &book, &embeddingText); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		// Parse embedding
		embedding, err := parseEmbedding(embeddingText)
		if err != nil {
			log.Printf("Warning: failed to parse embedding for %s: %v", verseID, err)
			continue
		}

		// Create datapoint with book as a restricts filter
		dp := &aiplatformpb.IndexDatapoint{
			DatapointId:   verseID,
			FeatureVector: embedding,
			Restricts: []*aiplatformpb.IndexDatapoint_Restriction{
				{
					Namespace:  "book",
					AllowList:  []string{book},
				},
			},
		}

		batch = append(batch, dp)
		totalCount++

		// Upsert when batch is full
		if len(batch) >= batchSize {
			if err := upsertBatch(ctx, client, indexName, batch); err != nil {
				log.Fatalf("Failed to upsert batch: %v", err)
			}
			batchCount++
			log.Printf("Upserted batch %d (%d total datapoints)", batchCount, totalCount)
			batch = batch[:0] // Reset batch
		}
	}

	// Upsert remaining datapoints
	if len(batch) > 0 {
		if err := upsertBatch(ctx, client, indexName, batch); err != nil {
			log.Fatalf("Failed to upsert final batch: %v", err)
		}
		batchCount++
		log.Printf("Upserted final batch %d (%d total datapoints)", batchCount, totalCount)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	log.Printf("Successfully upserted %d embeddings to Vertex AI Vector Search", totalCount)
}

func upsertBatch(ctx context.Context, client *aiplatform.IndexClient, indexName string, datapoints []*aiplatformpb.IndexDatapoint) error {
	req := &aiplatformpb.UpsertDatapointsRequest{
		Index:      indexName,
		Datapoints: datapoints,
	}

	_, err := client.UpsertDatapoints(ctx, req)
	return err
}

// parseEmbedding parses a pgvector text representation like "[0.1,0.2,0.3]"
func parseEmbedding(text string) ([]float32, error) {
	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")

	if text == "" {
		return nil, fmt.Errorf("empty embedding")
	}

	parts := strings.Split(text, ",")
	result := make([]float32, len(parts))

	for i, p := range parts {
		var val float32
		_, err := fmt.Sscanf(strings.TrimSpace(p), "%f", &val)
		if err != nil {
			return nil, fmt.Errorf("parse float at position %d: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}
