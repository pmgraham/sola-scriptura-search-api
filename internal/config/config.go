package config

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Config holds all application configuration
type Config struct {
	// API Settings
	APITitle   string
	APIVersion string
	APIPrefix  string
	Port       string

	// CORS
	CORSOrigins []string

	// Vector Search Backend: "pgvector" or "vertex"
	VectorBackend string

	// Vertex AI Vector Search settings (used when VectorBackend = "vertex")
	VertexProjectID            string
	VertexLocation             string
	VertexIndexEndpointID      string
	VertexDeployedIndexID      string
	VertexPublicEndpointDomain string
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
		APITitle:    getEnv("API_TITLE", "Sola Scriptura Search API"),
		APIVersion:  getEnv("API_VERSION", "1.0.0"),
		APIPrefix:   getEnv("API_PREFIX", "/api/v1"),
		Port:        getEnv("PORT", "8081"),
		CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),

		// Vector search backend configuration
		VectorBackend: getEnv("VECTOR_BACKEND", "pgvector"), // "pgvector" or "vertex"

		// Vertex AI settings
		VertexProjectID:            getEnv("VERTEX_PROJECT_ID", ""),
		VertexLocation:             getEnv("VERTEX_LOCATION", "us-central1"),
		VertexIndexEndpointID:      getEnv("VERTEX_INDEX_ENDPOINT_ID", ""),
		VertexDeployedIndexID:      getEnv("VERTEX_DEPLOYED_INDEX_ID", ""),
		VertexPublicEndpointDomain: getEnv("VERTEX_PUBLIC_ENDPOINT_DOMAIN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseCORSOrigins(value string) []string {
	var origins []string
	if err := json.Unmarshal([]byte(value), &origins); err == nil {
		return origins
	}
	parts := strings.Split(value, ",")
	origins = make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
