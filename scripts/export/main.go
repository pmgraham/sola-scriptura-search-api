// export_embeddings.go
//
// This script exports verse embeddings from PostgreSQL to a JSONL file
// formatted for Vertex AI Vector Search.
//
// Usage:
//   go run scripts/export_embeddings.go -output embeddings.jsonl
//
// The output format is one JSON object per line:
//   {"id": "John.3.16", "embedding": [0.1, 0.2, ...], "restricts": [{"namespace": "book", "allow": ["John"]}]}
//
// After running this script:
// 1. Upload the file to Cloud Storage:
//    gsutil cp embeddings.jsonl gs://YOUR_BUCKET/embeddings/
//
// 2. Create the Vertex AI index using the setup script or console

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DataPoint represents a single embedding for Vertex AI Vector Search
type DataPoint struct {
	ID        string     `json:"id"`
	Embedding []float32  `json:"embedding"`
	Restricts []Restrict `json:"restricts,omitempty"`
}

// Restrict defines a token-based filter
type Restrict struct {
	Namespace string   `json:"namespace"`
	Allow     []string `json:"allow"`
}

func main() {
	outputFile := flag.String("output", "embeddings.jsonl", "Output JSONL file path")
	flag.Parse()

	// Load environment variables
	godotenv.Load()

	postgresURI := os.Getenv("POSTGRES_URI")
	if postgresURI == "" {
		log.Fatal("POSTGRES_URI environment variable is required")
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	db, err := sqlx.ConnectContext(ctx, "postgres", postgresURI)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Open output file
	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	log.Printf("Exporting embeddings to %s...\n", *outputFile)

	// Get list of books to process in batches (avoids temp file limit)
	var books []string
	if err := db.SelectContext(ctx, &books, `
		SELECT book FROM api_views.mv_verses_search
		WHERE embedding IS NOT NULL
		GROUP BY book, book_order
		ORDER BY book_order
	`); err != nil {
		log.Fatalf("Failed to get books: %v", err)
	}
	log.Printf("Processing %d books...\n", len(books))

	encoder := json.NewEncoder(f)
	count := 0

	// Process one book at a time to avoid temp file limits
	for _, book := range books {
		rows, err := db.QueryxContext(ctx, `
			SELECT
				verse_id,
				book,
				embedding::text as embedding_text
			FROM api_views.mv_verses_search
			WHERE embedding IS NOT NULL AND book = $1
			ORDER BY chapter, verse
		`, book)
		if err != nil {
			log.Fatalf("Failed to query verses for book %s: %v", book, err)
		}

		bookCount := 0
		for rows.Next() {
			var verseID, bookName, embeddingText string
			if err := rows.Scan(&verseID, &bookName, &embeddingText); err != nil {
				rows.Close()
				log.Fatalf("Failed to scan row: %v", err)
			}

			// Parse the embedding from pgvector text format: "[0.1,0.2,...]"
			embedding, err := parseEmbedding(embeddingText)
			if err != nil {
				log.Printf("Warning: failed to parse embedding for %s: %v", verseID, err)
				continue
			}

			// Create the data point with book as a filter
			dp := DataPoint{
				ID:        verseID,
				Embedding: embedding,
				Restricts: []Restrict{
					{
						Namespace: "book",
						Allow:     []string{bookName},
					},
				},
			}

			if err := encoder.Encode(dp); err != nil {
				rows.Close()
				log.Fatalf("Failed to encode data point: %v", err)
			}

			count++
			bookCount++
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			log.Fatalf("Error iterating rows for book %s: %v", book, err)
		}
		rows.Close()

		log.Printf("  %s: %d verses", book, bookCount)
	}

	log.Printf("Successfully exported %d embeddings to %s\n", count, *outputFile)
	log.Println("\nNext steps:")
	log.Println("1. Upload to Cloud Storage:")
	log.Printf("   gsutil cp %s gs://YOUR_BUCKET/embeddings/\n", *outputFile)
	log.Println("\n2. Create Vertex AI index (see scripts/setup_vertex_index.go)")
}

// parseEmbedding parses a pgvector text representation like "[0.1,0.2,0.3]"
func parseEmbedding(text string) ([]float32, error) {
	// Remove brackets
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
