package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	// Prometheus
	PrometheusURL string
	
	// Storage
	StorageEnabled bool
	DatabaseURL    string
	
	// Analysis
	MetricsDuration time.Duration
	SafetyBuffer    float64 // e.g., 1.5 = 50% buffer on P95
	
	// Output
	OutputFormat string // text, json, yaml
	Verbose      bool
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		PrometheusURL:   getEnv("PROMETHEUS_URL", "http://localhost:9090"),
		StorageEnabled:  getEnvBool("STORAGE_ENABLED", true),
		DatabaseURL:     getEnv("DATABASE_URL", "host=localhost port=5432 user=costuser password=devpassword dbname=costoptimizer sslmode=disable"),
		MetricsDuration: 7 * 24 * time.Hour, // 7 days
		SafetyBuffer:    1.5,                 // 50% buffer
		OutputFormat:    "text",
		Verbose:         false,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.StorageEnabled && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL must be set when storage is enabled")
	}
	if c.MetricsDuration < 1*time.Hour {
		return fmt.Errorf("metrics duration must be at least 1 hour")
	}
	if c.SafetyBuffer < 1.0 {
		return fmt.Errorf("safety buffer must be >= 1.0")
	}
	return nil
}
