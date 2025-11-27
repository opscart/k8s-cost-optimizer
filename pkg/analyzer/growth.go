package analyzer

import (
	"fmt"
	"math"
)

// CalculateGrowthTrend analyzes usage trends over time using linear regression
func CalculateGrowthTrend(samples []MetricSample) (*GrowthTrend, error) {
	if len(samples) < 100 { // Need at least ~8 hours of data (100 samples at 5min resolution)
		return &GrowthTrend{
			RatePerMonth:    0,
			Confidence:      0,
			Predicted3Month: 0,
			Predicted6Month: 0,
			IsGrowing:       false,
		}, fmt.Errorf("insufficient data for trend analysis (need 100+ samples, got %d)", len(samples))
	}

	// Convert timestamps to hours since start for regression
	startTime := samples[0].Timestamp
	x := make([]float64, len(samples)) // Time in hours
	y := make([]float64, len(samples)) // Usage values

	for i, sample := range samples {
		x[i] = sample.Timestamp.Sub(startTime).Hours()
		y[i] = sample.Value
	}

	// Linear regression: y = mx + b
	slope, intercept, r2 := linearRegression(x, y)

	// Calculate average value
	currentAvg := calculateAverage(y)

	// Convert slope to percentage growth per month
	// slope is in units per hour, convert to percentage per month
	hoursPerMonth := 24.0 * 30.0
	absoluteGrowthPerMonth := slope * hoursPerMonth

	var ratePerMonth float64
	if currentAvg > 0 {
		ratePerMonth = (absoluteGrowthPerMonth / currentAvg) * 100.0
	}

	// Predict future values
	currentHours := x[len(x)-1]
	hours3Month := currentHours + (24 * 90)  // +90 days
	hours6Month := currentHours + (24 * 180) // +180 days

	predicted3Month := slope*hours3Month + intercept
	predicted6Month := slope*hours6Month + intercept

	// Ensure predictions don't go negative
	if predicted3Month < 0 {
		predicted3Month = currentAvg
	}
	if predicted6Month < 0 {
		predicted6Month = currentAvg
	}

	// Determine if growing (threshold: >3% per month)
	isGrowing := ratePerMonth > 3.0

	// Confidence based on R² (how well line fits data)
	confidence := r2

	return &GrowthTrend{
		RatePerMonth:    ratePerMonth,
		Confidence:      confidence,
		Predicted3Month: predicted3Month,
		Predicted6Month: predicted6Month,
		IsGrowing:       isGrowing,
	}, nil
}

// linearRegression performs simple linear regression
// Returns: slope, intercept, R² (coefficient of determination)
func linearRegression(x, y []float64) (slope, intercept, r2 float64) {
	n := float64(len(x))

	if n == 0 {
		return 0, 0, 0
	}

	// Calculate means
	meanX := calculateAverage(x)
	meanY := calculateAverage(y)

	// Calculate slope (m) and intercept (b)
	numerator := 0.0
	denominator := 0.0

	for i := 0; i < len(x); i++ {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denominator += (x[i] - meanX) * (x[i] - meanX)
	}

	if denominator == 0 {
		return 0, meanY, 0
	}

	slope = numerator / denominator
	intercept = meanY - slope*meanX

	// Calculate R² (coefficient of determination)
	ssTotal := 0.0 // Total sum of squares
	ssRes := 0.0   // Residual sum of squares

	for i := 0; i < len(x); i++ {
		predicted := slope*x[i] + intercept
		ssRes += (y[i] - predicted) * (y[i] - predicted)
		ssTotal += (y[i] - meanY) * (y[i] - meanY)
	}

	if ssTotal == 0 {
		r2 = 0
	} else {
		r2 = 1.0 - (ssRes / ssTotal)
	}

	// Clamp R² between 0 and 1
	if r2 < 0 {
		r2 = 0
	} else if r2 > 1 {
		r2 = 1
	}

	return slope, intercept, r2
}

// DetectSeasonalPattern checks for time-based patterns (weekday vs weekend, business hours)
func DetectSeasonalPattern(samples []MetricSample) string {
	if len(samples) < 336 { // Less than 1 day of data at 5min resolution
		return "insufficient-data"
	}

	// Group by hour of day
	hourlyAvg := make(map[int][]float64)

	for _, sample := range samples {
		hour := sample.Timestamp.Hour()
		hourlyAvg[hour] = append(hourlyAvg[hour], sample.Value)
	}

	// Calculate average for each hour
	hourlyMeans := make([]float64, 24)
	for hour := 0; hour < 24; hour++ {
		if values, exists := hourlyAvg[hour]; exists && len(values) > 0 {
			hourlyMeans[hour] = calculateAverage(values)
		}
	}

	// Detect business hours pattern (9am-5pm higher than nights)
	businessHoursAvg := (hourlyMeans[9] + hourlyMeans[12] + hourlyMeans[15]) / 3.0
	nightAvg := (hourlyMeans[0] + hourlyMeans[3] + hourlyMeans[23]) / 3.0

	if businessHoursAvg > nightAvg*1.5 {
		return "business-hours"
	}

	// Check if relatively flat
	cv := calculateCoefficientOfVariation(hourlyMeans)
	if cv < 0.15 {
		return "steady"
	}

	return "variable"
}

// Helper function to calculate CV from values
func calculateCoefficientOfVariation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := calculateAverage(values)
	if mean == 0 {
		return 0
	}

	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	stdDev := math.Sqrt(variance)

	return stdDev / mean
}
