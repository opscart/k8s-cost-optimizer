package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	// Prometheus
	PrometheusURL string
	
	// Storage
	StorageEnabled bool
	DatabaseURL    string
	
	// Metrics Analysis
	MetricsLookbackDays int           // 3, 7, 14, 30
	MetricsDuration     time.Duration // Computed from LookbackDays
	SafetyBuffer        float64       // e.g., 1.5 = 50% buffer on P95
	
	// Output
	OutputFormat string // text, json, yaml
	Verbose      bool
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	lookbackDays := getEnvInt("METRICS_LOOKBACK_DAYS", 7) // Default: 7 days
	
	return &Config{
		PrometheusURL:       getEnv("PROMETHEUS_URL", "http://localhost:9090"),
		StorageEnabled:      getEnvBool("STORAGE_ENABLED", true),
		DatabaseURL:         getEnv("DATABASE_URL", "host=localhost port=5432 user=costuser password=devpassword dbname=costoptimizer sslmode=disable"),
		MetricsLookbackDays: lookbackDays,
		MetricsDuration:     time.Duration(lookbackDays) * 24 * time.Hour,
		SafetyBuffer:        getEnvFloat("SAFETY_BUFFER", 1.5), // 50% buffer
		OutputFormat:        "text",
		Verbose:             false,
	}
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.StorageEnabled && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL must be set when storage is enabled")
	}
	if c.MetricsLookbackDays < 1 {
		return fmt.Errorf("metrics lookback must be at least 1 day")
	}
	if c.MetricsLookbackDays > 90 {
		return fmt.Errorf("metrics lookback cannot exceed 90 days (Prometheus retention)")
	}
	if c.SafetyBuffer < 1.0 {
		return fmt.Errorf("safety buffer must be >= 1.0")
	}
	return nil
}

// Presets for common scenarios
func (c *Config) UseDevPreset() {
	c.MetricsLookbackDays = 3
	c.MetricsDuration = 3 * 24 * time.Hour
	c.SafetyBuffer = 1.5
}

func (c *Config) UseProductionPreset() {
	c.MetricsLookbackDays = 14
	c.MetricsDuration = 14 * 24 * time.Hour
	c.SafetyBuffer = 2.0 // More conservative
}

func (c *Config) UseCriticalPreset() {
	c.MetricsLookbackDays = 30
	c.MetricsDuration = 30 * 24 * time.Hour
	c.SafetyBuffer = 2.5 // Very conservative
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
