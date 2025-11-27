package analyzer

import (
	"testing"
	"time"
)

func TestParsePrometheusResult(t *testing.T) {
	// Test parsing logic
	// We'll add mock Prometheus responses here
}

func TestGetHistoricalMetrics(t *testing.T) {
	// Test with mock Prometheus client
	// This will be expanded with actual test fixtures
}

// Helper to create mock metric samples for testing
func createMockSamples(count int, baseValue float64) []MetricSample {
	samples := make([]MetricSample, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		samples[i] = MetricSample{
			Timestamp: now.Add(-time.Duration(count-i) * 5 * time.Minute),
			Value:     baseValue + float64(i%10), // Slight variation
		}
	}

	return samples
}
