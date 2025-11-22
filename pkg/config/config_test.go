package config

import (
	"os"
	"testing"
	"time"
)

func TestNewConfigDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("METRICS_LOOKBACK_DAYS")
	os.Unsetenv("SAFETY_BUFFER")
	os.Unsetenv("PROMETHEUS_URL")
	
	cfg := NewConfig()
	
	// Verify defaults
	if cfg.MetricsLookbackDays != 7 {
		t.Errorf("Expected default lookback 7 days, got %d", cfg.MetricsLookbackDays)
	}
	
	if cfg.MetricsDuration != 7*24*time.Hour {
		t.Errorf("Expected duration 168h, got %v", cfg.MetricsDuration)
	}
	
	if cfg.SafetyBuffer != 1.5 {
		t.Errorf("Expected safety buffer 1.5, got %.1f", cfg.SafetyBuffer)
	}
	
	if cfg.PrometheusURL != "http://localhost:9090" {
		t.Errorf("Expected default Prometheus URL, got %s", cfg.PrometheusURL)
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("METRICS_LOOKBACK_DAYS", "15")
	os.Setenv("SAFETY_BUFFER", "2.0")
	os.Setenv("PROMETHEUS_URL", "http://prometheus:9090")
	defer os.Unsetenv("METRICS_LOOKBACK_DAYS")
	defer os.Unsetenv("SAFETY_BUFFER")
	defer os.Unsetenv("PROMETHEUS_URL")
	
	cfg := NewConfig()
	
	if cfg.MetricsLookbackDays != 15 {
		t.Errorf("Expected lookback 15 days from env, got %d", cfg.MetricsLookbackDays)
	}
	
	if cfg.SafetyBuffer != 2.0 {
		t.Errorf("Expected safety buffer 2.0 from env, got %.1f", cfg.SafetyBuffer)
	}
	
	if cfg.PrometheusURL != "http://prometheus:9090" {
		t.Errorf("Expected custom Prometheus URL, got %s", cfg.PrometheusURL)
	}
	
	expectedDuration := 15 * 24 * time.Hour
	if cfg.MetricsDuration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, cfg.MetricsDuration)
	}
}

func TestDevPreset(t *testing.T) {
	cfg := NewConfig()
	cfg.UseDevPreset()
	
	if cfg.MetricsLookbackDays != 3 {
		t.Errorf("Dev preset should be 3 days, got %d", cfg.MetricsLookbackDays)
	}
	
	if cfg.MetricsDuration != 3*24*time.Hour {
		t.Errorf("Dev preset duration should be 72h, got %v", cfg.MetricsDuration)
	}
	
	if cfg.SafetyBuffer != 1.5 {
		t.Errorf("Dev preset buffer should be 1.5, got %.1f", cfg.SafetyBuffer)
	}
}

func TestProductionPreset(t *testing.T) {
	cfg := NewConfig()
	cfg.UseProductionPreset()
	
	if cfg.MetricsLookbackDays != 14 {
		t.Errorf("Production preset should be 14 days, got %d", cfg.MetricsLookbackDays)
	}
	
	if cfg.MetricsDuration != 14*24*time.Hour {
		t.Errorf("Production preset duration should be 336h, got %v", cfg.MetricsDuration)
	}
	
	if cfg.SafetyBuffer != 2.0 {
		t.Errorf("Production preset buffer should be 2.0, got %.1f", cfg.SafetyBuffer)
	}
}

func TestCriticalPreset(t *testing.T) {
	cfg := NewConfig()
	cfg.UseCriticalPreset()
	
	if cfg.MetricsLookbackDays != 30 {
		t.Errorf("Critical preset should be 30 days, got %d", cfg.MetricsLookbackDays)
	}
	
	if cfg.MetricsDuration != 30*24*time.Hour {
		t.Errorf("Critical preset duration should be 720h, got %v", cfg.MetricsDuration)
	}
	
	if cfg.SafetyBuffer != 2.5 {
		t.Errorf("Critical preset buffer should be 2.5, got %.1f", cfg.SafetyBuffer)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func(*Config)
		expectError   bool
		errorContains string
	}{
		{
			name: "valid default config",
			setupConfig: func(c *Config) {
				// Use defaults
			},
			expectError: false,
		},
		{
			name: "lookback too low",
			setupConfig: func(c *Config) {
				c.MetricsLookbackDays = 0
			},
			expectError:   true,
			errorContains: "at least 1 day",
		},
		{
			name: "lookback too high",
			setupConfig: func(c *Config) {
				c.MetricsLookbackDays = 100
			},
			expectError:   true,
			errorContains: "cannot exceed 90 days",
		},
		{
			name: "safety buffer too low",
			setupConfig: func(c *Config) {
				c.SafetyBuffer = 0.5
			},
			expectError:   true,
			errorContains: "must be >= 1.0",
		},
		{
			name: "valid edge case - 1 day",
			setupConfig: func(c *Config) {
				c.MetricsLookbackDays = 1
			},
			expectError: false,
		},
		{
			name: "valid edge case - 90 days",
			setupConfig: func(c *Config) {
				c.MetricsLookbackDays = 90
			},
			expectError: false,
		},
		{
			name: "valid edge case - buffer 1.0",
			setupConfig: func(c *Config) {
				c.SafetyBuffer = 1.0
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			tt.setupConfig(cfg)
			
			err := cfg.Validate()
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", 
						tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestInvalidEnvValues(t *testing.T) {
	// Test invalid integer
	os.Setenv("METRICS_LOOKBACK_DAYS", "invalid")
	defer os.Unsetenv("METRICS_LOOKBACK_DAYS")
	
	cfg := NewConfig()
	
	// Should fall back to default
	if cfg.MetricsLookbackDays != 7 {
		t.Errorf("Expected fallback to default 7, got %d", cfg.MetricsLookbackDays)
	}
}

func TestStorageConfiguration(t *testing.T) {
	os.Setenv("STORAGE_ENABLED", "true")
	os.Setenv("DATABASE_URL", "postgres://test")
	defer os.Unsetenv("STORAGE_ENABLED")
	defer os.Unsetenv("DATABASE_URL")
	
	cfg := NewConfig()
	
	if !cfg.StorageEnabled {
		t.Error("Expected storage to be enabled")
	}
	
	if cfg.DatabaseURL != "postgres://test" {
		t.Errorf("Expected custom database URL, got %s", cfg.DatabaseURL)
	}
}

func TestStorageValidation(t *testing.T) {
	cfg := NewConfig()
	cfg.StorageEnabled = true
	cfg.DatabaseURL = ""
	
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error when storage enabled but no database URL")
	}
	
	if !contains(err.Error(), "DATABASE_URL") {
		t.Errorf("Expected error about DATABASE_URL, got: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || len(s) > len(substr) && 
		   (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		   containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
