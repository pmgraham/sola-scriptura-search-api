// setup_vertex_index.go
//
// This script creates a Vertex AI Vector Search index and endpoint.
//
// Prerequisites:
// 1. Export embeddings to JSONL: go run scripts/export_embeddings.go
// 2. Upload to GCS: gsutil cp embeddings.jsonl gs://YOUR_BUCKET/embeddings/
// 3. Set environment variables (see below)
//
// Environment variables:
//   GCP_PROJECT_ID       - Your GCP project ID
//   VERTEX_LOCATION      - Region (default: us-central1)
//   GCS_BUCKET_URI       - Cloud Storage URI with embeddings (e.g., gs://bucket/embeddings)
//   INDEX_DISPLAY_NAME   - Display name for the index (default: sola-scriptura-verses)
//
// Usage:
//   go run scripts/setup_vertex_index.go
//
// After this script completes, note the Index ID and Endpoint ID and add them to your .env:
//   VERTEX_INDEX_ENDPOINT_ID=<endpoint_id>
//   VERTEX_DEPLOYED_INDEX_ID=<deployed_index_id>

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	embeddingDimensions = 3072 // Qwen3-Embedding-8B dimensions
)

func main() {
	createIndex := flag.Bool("create-index", false, "Create a new index")
	createEndpoint := flag.Bool("create-endpoint", false, "Create a new endpoint")
	deployIndex := flag.Bool("deploy", false, "Deploy index to endpoint")
	indexID := flag.String("index-id", "", "Index ID (for deploy)")
	endpointID := flag.String("endpoint-id", "", "Endpoint ID (for deploy)")
	flag.Parse()

	godotenv.Load()

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

	gcsBucketURI := os.Getenv("GCS_BUCKET_URI")
	displayName := os.Getenv("INDEX_DISPLAY_NAME")
	if displayName == "" {
		displayName = "sola-scriptura-verses"
	}

	ctx := context.Background()
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)

	if *createIndex {
		if gcsBucketURI == "" {
			log.Fatal("GCS_BUCKET_URI is required for index creation")
		}
		createNewIndex(ctx, endpoint, parent, displayName, gcsBucketURI)
	} else if *createEndpoint {
		createNewEndpoint(ctx, endpoint, parent, displayName)
	} else if *deployIndex {
		if *indexID == "" || *endpointID == "" {
			log.Fatal("--index-id and --endpoint-id are required for deployment")
		}
		deployIndexToEndpoint(ctx, endpoint, parent, *indexID, *endpointID, displayName)
	} else {
		fmt.Println("Vertex AI Vector Search Setup")
		fmt.Println("=============================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  1. Create index:    go run scripts/setup_vertex_index.go -create-index")
		fmt.Println("  2. Create endpoint: go run scripts/setup_vertex_index.go -create-endpoint")
		fmt.Println("  3. Deploy:          go run scripts/setup_vertex_index.go -deploy -index-id=XXX -endpoint-id=YYY")
		fmt.Println()
		fmt.Println("Current configuration:")
		fmt.Printf("  Project ID:     %s\n", projectID)
		fmt.Printf("  Location:       %s\n", location)
		fmt.Printf("  GCS Bucket URI: %s\n", gcsBucketURI)
		fmt.Printf("  Display Name:   %s\n", displayName)
		fmt.Printf("  Dimensions:     %d\n", embeddingDimensions)
	}
}

func createNewIndex(ctx context.Context, endpoint, parent, displayName, gcsBucketURI string) {
	log.Printf("Creating Vertex AI Vector Search index...")
	log.Printf("  Parent: %s", parent)
	log.Printf("  Display Name: %s", displayName)
	log.Printf("  GCS URI: %s", gcsBucketURI)
	log.Printf("  Dimensions: %d", embeddingDimensions)

	client, err := aiplatform.NewIndexClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		log.Fatalf("Failed to create index client: %v", err)
	}
	defer client.Close()

	// Build config for the index
	// The metadata has a nested "config" structure with algorithmConfig required
	treeAhConfig, _ := structpb.NewStruct(map[string]interface{}{
		"leafNodeEmbeddingCount":   1000,
		"leafNodesToSearchPercent": 5,
	})

	algorithmConfig, _ := structpb.NewStruct(map[string]interface{}{
		"treeAhConfig": treeAhConfig.AsMap(),
	})

	configStruct, _ := structpb.NewStruct(map[string]interface{}{
		"dimensions":                embeddingDimensions,
		"approximateNeighborsCount": 150,
		"distanceMeasureType":       "COSINE_DISTANCE",
		"algorithmConfig":           algorithmConfig.AsMap(),
	})

	indexConfig, _ := structpb.NewStruct(map[string]interface{}{
		"contentsDeltaUri": gcsBucketURI,
		"config":           configStruct.AsMap(),
	})

	req := &aiplatformpb.CreateIndexRequest{
		Parent: parent,
		Index: &aiplatformpb.Index{
			DisplayName: displayName,
			Description: "Verse embeddings for Sola Scriptura semantic search",
			Metadata:    structpb.NewStructValue(indexConfig),
			IndexUpdateMethod: aiplatformpb.Index_STREAM_UPDATE,
		},
	}

	// For batch index with initial data, add:
	// ContentsDeltaUri: gcsBucketURI,

	op, err := client.CreateIndex(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	log.Printf("Index creation started. Operation: %s", op.Name())
	log.Printf("This may take 30-60 minutes. You can check status in the GCP Console.")
	log.Println()
	log.Println("Waiting for index creation to complete...")

	index, err := op.Wait(ctx)
	if err != nil {
		log.Fatalf("Index creation failed: %v", err)
	}

	log.Printf("Index created successfully!")
	log.Printf("  Index Name: %s", index.Name)
	log.Printf("  Index ID: %s", extractID(index.Name))
	log.Println()
	log.Println("Next step: Create an endpoint:")
	log.Println("  go run scripts/setup_vertex_index.go -create-endpoint")
}

func createNewEndpoint(ctx context.Context, endpoint, parent, displayName string) {
	log.Printf("Creating Vertex AI Vector Search endpoint...")
	log.Printf("  Parent: %s", parent)

	client, err := aiplatform.NewIndexEndpointClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		log.Fatalf("Failed to create endpoint client: %v", err)
	}
	defer client.Close()

	req := &aiplatformpb.CreateIndexEndpointRequest{
		Parent: parent,
		IndexEndpoint: &aiplatformpb.IndexEndpoint{
			DisplayName:           displayName + "-endpoint",
			Description:           "Public endpoint for Sola Scriptura verse search",
			PublicEndpointEnabled: true,
		},
	}

	op, err := client.CreateIndexEndpoint(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create endpoint: %v", err)
	}

	log.Printf("Endpoint creation started. Operation: %s", op.Name())
	log.Println("Waiting for endpoint creation...")

	indexEndpoint, err := op.Wait(ctx)
	if err != nil {
		log.Fatalf("Endpoint creation failed: %v", err)
	}

	log.Printf("Endpoint created successfully!")
	log.Printf("  Endpoint Name: %s", indexEndpoint.Name)
	log.Printf("  Endpoint ID: %s", extractID(indexEndpoint.Name))
	log.Printf("  Public Domain: %s", indexEndpoint.PublicEndpointDomainName)
	log.Println()
	log.Println("Next step: Deploy the index to the endpoint:")
	log.Printf("  go run scripts/setup_vertex_index.go -deploy -index-id=<INDEX_ID> -endpoint-id=%s", extractID(indexEndpoint.Name))
}

func deployIndexToEndpoint(ctx context.Context, endpoint, parent, indexID, endpointID, displayName string) {
	log.Printf("Deploying index to endpoint...")
	log.Printf("  Index ID: %s", indexID)
	log.Printf("  Endpoint ID: %s", endpointID)

	client, err := aiplatform.NewIndexEndpointClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		log.Fatalf("Failed to create endpoint client: %v", err)
	}
	defer client.Close()

	indexEndpointName := fmt.Sprintf("%s/indexEndpoints/%s", parent, endpointID)
	indexName := fmt.Sprintf("%s/indexes/%s", parent, indexID)

	// Generate a unique deployed index ID (must start with letter, only letters/numbers/underscores)
	sanitizedName := strings.ReplaceAll(displayName, "-", "_")
	deployedIndexID := fmt.Sprintf("deployed_%s_%d", sanitizedName, time.Now().Unix())

	req := &aiplatformpb.DeployIndexRequest{
		IndexEndpoint: indexEndpointName,
		DeployedIndex: &aiplatformpb.DeployedIndex{
			Id:    deployedIndexID,
			Index: indexName,
			// Use automatic resources for simplicity
			AutomaticResources: &aiplatformpb.AutomaticResources{
				MinReplicaCount: 1,
				MaxReplicaCount: 2,
			},
		},
	}

	op, err := client.DeployIndex(ctx, req)
	if err != nil {
		log.Fatalf("Failed to deploy index: %v", err)
	}

	log.Printf("Deployment started. Operation: %s", op.Name())
	log.Println("This may take 20-30 minutes. Waiting...")

	resp, err := op.Wait(ctx)
	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	log.Printf("Index deployed successfully!")
	log.Println()
	log.Println("Add these to your .env file:")
	log.Printf("  VERTEX_INDEX_ENDPOINT_ID=%s", endpointID)
	log.Printf("  VERTEX_DEPLOYED_INDEX_ID=%s", deployedIndexID)
	log.Println()
	log.Printf("Deployed index: %+v", resp.DeployedIndex)
}

func extractID(resourceName string) string {
	// Resource names are like: projects/X/locations/Y/indexes/Z
	// Extract the last component
	for i := len(resourceName) - 1; i >= 0; i-- {
		if resourceName[i] == '/' {
			return resourceName[i+1:]
		}
	}
	return resourceName
}
