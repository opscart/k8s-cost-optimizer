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

// TestSplitSamplesByWeekday tests weekday/weekend split logic
func TestSplitSamplesByWeekday(t *testing.T) {
	// Create test samples spanning different days
	samples := []MetricSample{
		{Timestamp: time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC), Value: 100},  // Monday
		{Timestamp: time.Date(2025, 12, 2, 12, 0, 0, 0, time.UTC), Value: 110},  // Tuesday
		{Timestamp: time.Date(2025, 12, 3, 12, 0, 0, 0, time.UTC), Value: 120},  // Wednesday
		{Timestamp: time.Date(2025, 12, 4, 12, 0, 0, 0, time.UTC), Value: 130},  // Thursday
		{Timestamp: time.Date(2025, 12, 5, 12, 0, 0, 0, time.UTC), Value: 140},  // Friday
		{Timestamp: time.Date(2025, 12, 6, 12, 0, 0, 0, time.UTC), Value: 50},   // Saturday
		{Timestamp: time.Date(2025, 12, 7, 12, 0, 0, 0, time.UTC), Value: 60},   // Sunday
	}

	weekday, weekend := SplitSamplesByWeekday(samples)

	// Verify counts
	if len(weekday) != 5 {
		t.Errorf("Expected 5 weekday samples, got %d", len(weekday))
	}
	if len(weekend) != 2 {
		t.Errorf("Expected 2 weekend samples, got %d", len(weekend))
	}

	// Verify weekday values (Monday-Friday)
	expectedWeekday := []float64{100, 110, 120, 130, 140}
	for i, sample := range weekday {
		if sample.Value != expectedWeekday[i] {
			t.Errorf("Weekday sample %d: expected %.0f, got %.0f", i, expectedWeekday[i], sample.Value)
		}
	}

	// Verify weekend values (Saturday-Sunday)
	expectedWeekend := []float64{50, 60}
	for i, sample := range weekend {
		if sample.Value != expectedWeekend[i] {
			t.Errorf("Weekend sample %d: expected %.0f, got %.0f", i, expectedWeekend[i], sample.Value)
		}
	}
}

// TestSplitSamplesByWeekday_EmptyInput tests edge case
func TestSplitSamplesByWeekday_EmptyInput(t *testing.T) {
	samples := []MetricSample{}
	weekday, weekend := SplitSamplesByWeekday(samples)

	if len(weekday) != 0 {
		t.Errorf("Expected 0 weekday samples for empty input, got %d", len(weekday))
	}
	if len(weekend) != 0 {
		t.Errorf("Expected 0 weekend samples for empty input, got %d", len(weekend))
	}
}

// TestSplitSamplesByWeekday_AllWeekday tests all weekday samples
func TestSplitSamplesByWeekday_AllWeekday(t *testing.T) {
	samples := []MetricSample{
		{Timestamp: time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC), Value: 100},  // Monday
		{Timestamp: time.Date(2025, 12, 2, 12, 0, 0, 0, time.UTC), Value: 110},  // Tuesday
		{Timestamp: time.Date(2025, 12, 3, 12, 0, 0, 0, time.UTC), Value: 120},  // Wednesday
	}

	weekday, weekend := SplitSamplesByWeekday(samples)

	if len(weekday) != 3 {
		t.Errorf("Expected 3 weekday samples, got %d", len(weekday))
	}
	if len(weekend) != 0 {
		t.Errorf("Expected 0 weekend samples, got %d", len(weekend))
	}
}

// TestSplitSamplesByWeekday_AllWeekend tests all weekend samples
func TestSplitSamplesByWeekday_AllWeekend(t *testing.T) {
	samples := []MetricSample{
		{Timestamp: time.Date(2025, 12, 6, 12, 0, 0, 0, time.UTC), Value: 50},   // Saturday
		{Timestamp: time.Date(2025, 12, 7, 12, 0, 0, 0, time.UTC), Value: 60},   // Sunday
	}

	weekday, weekend := SplitSamplesByWeekday(samples)

	if len(weekday) != 0 {
		t.Errorf("Expected 0 weekday samples, got %d", len(weekday))
	}
	if len(weekend) != 2 {
		t.Errorf("Expected 2 weekend samples, got %d", len(weekend))
	}
}
