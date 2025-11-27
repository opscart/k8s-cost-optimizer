package analyzer

import (
	"math"
	"testing"
	"time"
)

func TestCalculatePercentiles(t *testing.T) {
	// Create test samples: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
	samples := make([]MetricSample, 10)
	for i := 0; i < 10; i++ {
		samples[i] = MetricSample{
			Timestamp: time.Now(),
			Value:     float64(i + 1),
		}
	}

	percentiles, err := CalculatePercentiles(samples)
	if err != nil {
		t.Fatalf("CalculatePercentiles failed: %v", err)
	}

	// Test results
	if percentiles.Average != 5.5 {
		t.Errorf("Expected average 5.5, got %.2f", percentiles.Average)
	}

	if percentiles.Min != 1.0 {
		t.Errorf("Expected min 1.0, got %.2f", percentiles.Min)
	}

	if percentiles.Peak != 10.0 {
		t.Errorf("Expected peak 10.0, got %.2f", percentiles.Peak)
	}

	// P50 should be around 5.5
	if math.Abs(percentiles.P50-5.5) > 0.5 {
		t.Errorf("Expected P50 ~5.5, got %.2f", percentiles.P50)
	}

	// P95 should be around 9.5
	if math.Abs(percentiles.P95-9.55) > 0.1 {
		t.Errorf("Expected P95 ~9.55, got %.2f", percentiles.P95)
	}
}

func TestAnalyzeUsagePattern(t *testing.T) {
	// Test steady pattern
	steadySamples := make([]MetricSample, 100)
	for i := 0; i < 100; i++ {
		steadySamples[i] = MetricSample{
			Timestamp: time.Now(),
			Value:     100.0 + float64(i%5), // Very little variation
		}
	}

	pattern := AnalyzeUsagePattern(steadySamples)
	if pattern.Type != "steady" {
		t.Errorf("Expected 'steady' pattern, got '%s'", pattern.Type)
	}

	// Test spiky pattern
	spikySamples := make([]MetricSample, 100)
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			spikySamples[i].Value = 500.0 // Spike
		} else {
			spikySamples[i].Value = 100.0 // Normal
		}
		spikySamples[i].Timestamp = time.Now()
	}

	spikyPattern := AnalyzeUsagePattern(spikySamples)
	if spikyPattern.Type == "steady" {
		t.Errorf("Expected 'spiky' or 'moderate' pattern, got '%s'", spikyPattern.Type)
	}
}
