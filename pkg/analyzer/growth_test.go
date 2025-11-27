package analyzer

import (
	"math"
	"testing"
	"time"
)

func TestCalculateGrowthTrend(t *testing.T) {
	// Create growing workload: starts at 100, grows 10% per month
	// Simulate 7 days = 2016 samples at 5min resolution
	samples := make([]MetricSample, 2016)
	startTime := time.Now().Add(-7 * 24 * time.Hour)

	for i := 0; i < 2016; i++ {
		timestamp := startTime.Add(time.Duration(i) * 5 * time.Minute)

		// Linear growth: 100 + (time_in_hours * 0.0139)
		// 0.0139 per hour = ~10% per month on base of 100
		hours := float64(i) * 5.0 / 60.0
		value := 100.0 + (hours * 0.0139)

		samples[i] = MetricSample{
			Timestamp: timestamp,
			Value:     value,
		}
	}

	trend, err := CalculateGrowthTrend(samples)
	if err != nil {
		t.Fatalf("CalculateGrowthTrend failed: %v", err)
	}

	// Should detect ~10% monthly growth
	if math.Abs(trend.RatePerMonth-10.0) > 2.0 {
		t.Errorf("Expected ~10%% growth, got %.2f%%", trend.RatePerMonth)
	}

	if !trend.IsGrowing {
		t.Errorf("Expected IsGrowing=true, got false")
	}

	if trend.Predicted3Month <= 100.0 {
		t.Errorf("Expected 3-month prediction > 100, got %.2f", trend.Predicted3Month)
	}
}

func TestCalculateGrowthTrend_Steady(t *testing.T) {
	// Create steady workload (no growth)
	samples := make([]MetricSample, 2016)
	startTime := time.Now().Add(-7 * 24 * time.Hour)

	for i := 0; i < 2016; i++ {
		timestamp := startTime.Add(time.Duration(i) * 5 * time.Minute)
		value := 100.0 + float64(i%10) // Small random variation

		samples[i] = MetricSample{
			Timestamp: timestamp,
			Value:     value,
		}
	}

	trend, err := CalculateGrowthTrend(samples)
	if err != nil {
		t.Fatalf("CalculateGrowthTrend failed: %v", err)
	}

	// Should detect near-zero growth
	if math.Abs(trend.RatePerMonth) > 5.0 {
		t.Errorf("Expected ~0%% growth, got %.2f%%", trend.RatePerMonth)
	}

	if trend.IsGrowing {
		t.Errorf("Expected IsGrowing=false for steady workload")
	}
}

func TestDetectSeasonalPattern(t *testing.T) {
	// Create business hours pattern
	samples := make([]MetricSample, 2016)
	startTime := time.Now().Add(-7 * 24 * time.Hour)

	for i := 0; i < 2016; i++ {
		timestamp := startTime.Add(time.Duration(i) * 5 * time.Minute)
		hour := timestamp.Hour()

		// Higher usage during business hours (9am-5pm)
		var value float64
		if hour >= 9 && hour <= 17 {
			value = 200.0 // Business hours
		} else {
			value = 80.0 // Off hours
		}

		samples[i] = MetricSample{
			Timestamp: timestamp,
			Value:     value,
		}
	}

	pattern := DetectSeasonalPattern(samples)
	if pattern != "business-hours" {
		t.Errorf("Expected 'business-hours' pattern, got '%s'", pattern)
	}
}
