package vertex

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/jmoiron/sqlx"
	"github.com/sola-scriptura-search-api/internal/models"
	"github.com/sola-scriptura-search-api/internal/repository"
	"google.golang.org/api/option"
)

// Ensure VectorSearchRepository implements repository.VectorSearchRepository
var _ repository.VectorSearchRepository = (*VectorSearchRepository)(nil)

// Config holds Vertex AI Vector Search configuration
type Config struct {
	ProjectID            string // GCP project ID
	Location             string // e.g., "us-central1"
	IndexEndpointID      string // Deployed index endpoint ID
	DeployedIndexID      string // The deployed index ID within the endpoint
	PublicEndpointDomain string // Public endpoint domain for queries (e.g., "123.us-central1-456.vdb.vertexai.goog")
}

// VectorSearchRepository implements repository.VectorSearchRepository using Vertex AI Vector Search
type VectorSearchRepository struct {
	config      Config
	matchClient *aiplatform.MatchClient
	db          *sqlx.DB // Used to look up verse text after getting IDs from Vertex AI
}

// NewVectorSearchRepository creates a new Vertex AI vector search repository
func NewVectorSearchRepository(ctx context.Context, config Config, db *sqlx.DB) (*VectorSearchRepository, error) {
	// For public endpoints, use the public domain; otherwise use regional endpoint
	var endpoint string
	if config.PublicEndpointDomain != "" {
		endpoint = fmt.Sprintf("%s:443", config.PublicEndpointDomain)
	} else {
		endpoint = fmt.Sprintf("%s-aiplatform.googleapis.com:443", config.Location)
	}

	matchClient, err := aiplatform.NewMatchClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("create match client: %w", err)
	}

	return &VectorSearchRepository{
		config:      config,
		matchClient: matchClient,
		db:          db,
	}, nil
}

// Close closes the Vertex AI client
func (r *VectorSearchRepository) Close() error {
	if r.matchClient != nil {
		return r.matchClient.Close()
	}
	return nil
}

// SearchVersesByEmbedding performs vector similarity search using Vertex AI Vector Search
func (r *VectorSearchRepository) SearchVersesByEmbedding(ctx context.Context, embedding []float64, topK int) ([]models.ScoredVerse, error) {
	// Build the index endpoint resource name
	indexEndpoint := fmt.Sprintf(
		"projects/%s/locations/%s/indexEndpoints/%s",
		r.config.ProjectID,
		r.config.Location,
		r.config.IndexEndpointID,
	)

	// Convert embedding to float32
	featureVector := make([]float32, len(embedding))
	for i, v := range embedding {
		featureVector[i] = float32(v)
	}

	// Build the FindNeighbors request
	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   indexEndpoint,
		DeployedIndexId: r.config.DeployedIndexID,
		Queries: []*aiplatformpb.FindNeighborsRequest_Query{
			{
				Datapoint: &aiplatformpb.IndexDatapoint{
					FeatureVector: featureVector,
				},
				NeighborCount: int32(topK),
			},
		},
	}

	// Execute the search
	resp, err := r.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("find neighbors: %w", err)
	}

	// Extract verse IDs and scores from the response
	if len(resp.NearestNeighbors) == 0 || len(resp.NearestNeighbors[0].Neighbors) == 0 {
		return []models.ScoredVerse{}, nil
	}

	neighbors := resp.NearestNeighbors[0].Neighbors

	// Collect verse IDs for batch lookup
	verseIDs := make([]string, len(neighbors))
	scoreMap := make(map[string]float64, len(neighbors))

	for i, neighbor := range neighbors {
		verseID := neighbor.Datapoint.DatapointId
		verseIDs[i] = verseID
		// Vertex AI returns distance, convert to similarity score
		// For cosine distance: similarity = 1 - distance
		scoreMap[verseID] = float64(1 - neighbor.Distance)
	}

	// Look up verse details from PostgreSQL
	results, err := r.lookupVerses(ctx, verseIDs, scoreMap)
	if err != nil {
		return nil, fmt.Errorf("lookup verses: %w", err)
	}

	return results, nil
}

// lookupVerses retrieves verse details from PostgreSQL given a list of verse IDs
func (r *VectorSearchRepository) lookupVerses(ctx context.Context, verseIDs []string, scoreMap map[string]float64) ([]models.ScoredVerse, error) {
	if len(verseIDs) == 0 {
		return []models.ScoredVerse{}, nil
	}

	// Use the materialized view for verse lookup
	query, args, err := sqlx.In(`
		SELECT verse_id, book, chapter, verse, text
		FROM api_views.mv_verses_search
		WHERE verse_id IN (?)
	`, verseIDs)
	if err != nil {
		return nil, fmt.Errorf("build IN query: %w", err)
	}

	// Rebind for PostgreSQL
	query = r.db.Rebind(query)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query verses: %w", err)
	}
	defer rows.Close()

	// Create a map for ordering results by score
	verseMap := make(map[string]models.ScoredVerse)
	for rows.Next() {
		var v models.ScoredVerse
		if err := rows.Scan(&v.VerseID, &v.Book, &v.Chapter, &v.Verse, &v.Text); err != nil {
			return nil, fmt.Errorf("scan verse: %w", err)
		}
		v.Score = scoreMap[v.VerseID]
		verseMap[v.VerseID] = v
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verses: %w", err)
	}

	// Preserve the order from Vertex AI (sorted by relevance)
	results := make([]models.ScoredVerse, 0, len(verseIDs))
	for _, id := range verseIDs {
		if v, ok := verseMap[id]; ok {
			results = append(results, v)
		}
	}

	return results, nil
}
