package config

import (
	"os"
	"strconv"
	"sync"
)

// Config holds configuration for database and embedding operations
type Config struct {
	// PostgreSQL
	PostgresURI string

	// Embeddings
	EmbeddingProvider   string // "vertex" or "custom"
	EmbeddingServiceURL string // For custom provider
	EmbeddingDimensions int

	// Vertex AI (when EmbeddingProvider = "vertex")
	GCPProjectID string
	GCPLocation  string
	VertexModel  string
}

var (
	config *Config
	once   sync.Once
)

// GetConfig returns the singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		config = loadConfig()
	})
	return config
}

func loadConfig() *Config {
	return &Config{
		// PostgreSQL
		PostgresURI: getEnv("POSTGRES_URI", ""),

		// Embeddings
		EmbeddingProvider:   getEnv("EMBEDDING_PROVIDER", "vertex"),
		EmbeddingServiceURL: getEnv("EMBEDDING_SERVICE_URL", "http://localhost:8001"),
		EmbeddingDimensions: getEnvInt("EMBEDDING_DIMENSIONS", 3072),

		// Vertex AI
		GCPProjectID: getEnv("GCP_PROJECT_ID", ""),
		GCPLocation:  getEnv("GCP_LOCATION", "us-central1"),
		VertexModel:  getEnv("VERTEX_MODEL", "gemini-embedding-001"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}
