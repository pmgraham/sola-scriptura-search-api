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
		APITitle:   getEnv("API_TITLE", "Sola Scriptura Search API"),
		APIVersion: getEnv("API_VERSION", "1.0.0"),
		APIPrefix:  getEnv("API_PREFIX", "/api/v1"),
		Port:       getEnv("PORT", "8081"),
		CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),
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
