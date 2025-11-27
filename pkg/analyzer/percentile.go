package analyzer

import (
	"fmt"
	"math"
	"sort"
)

// CalculatePercentiles computes P50, P90, P95, P99, and peak from samples
func CalculatePercentiles(samples []MetricSample) (*Percentiles, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("no samples provided")
	}

	// Extract just the values for calculation
	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = sample.Value
	}

	// Sort values
	sort.Float64s(values)

	percentiles := &Percentiles{
		Average: calculateAverage(values),
		P50:     calculatePercentile(values, 50),
		P90:     calculatePercentile(values, 90),
		P95:     calculatePercentile(values, 95),
		P99:     calculatePercentile(values, 99),
		Peak:    values[len(values)-1], // Maximum value
		Min:     values[0],             // Minimum value
	}

	return percentiles, nil
}

// calculatePercentile computes the Nth percentile using linear interpolation
func calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	// Calculate the index for the percentile
	// Using the "nearest rank" method with linear interpolation
	n := float64(len(sortedValues))
	rank := (percentile / 100.0) * (n - 1)

	// Get lower and upper indices
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	// If indices are the same, return that value
	if lowerIndex == upperIndex {
		return sortedValues[lowerIndex]
	}

	// Linear interpolation between the two values
	lowerValue := sortedValues[lowerIndex]
	upperValue := sortedValues[upperIndex]
	fraction := rank - float64(lowerIndex)

	return lowerValue + (upperValue-lowerValue)*fraction
}

// calculateAverage computes the mean of values
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// CalculateCoefficientOfVariation measures the relative variability
// High CV (>0.5) = spiky workload
// Low CV (<0.2) = steady workload
func CalculateCoefficientOfVariation(samples []MetricSample) float64 {
	if len(samples) < 2 {
		return 0
	}

	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = sample.Value
	}

	mean := calculateAverage(values)
	if mean == 0 {
		return 0
	}

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	stdDev := math.Sqrt(variance)

	// Coefficient of variation = stdDev / mean
	return stdDev / mean
}

// AnalyzeUsagePattern determines if workload is steady, spiky, or periodic
func AnalyzeUsagePattern(samples []MetricSample) UsagePattern {
	if len(samples) < 10 {
		return UsagePattern{
			Type:       "unknown",
			Variation:  0,
			Confidence: 0,
		}
	}

	cv := CalculateCoefficientOfVariation(samples)

	// Classify based on coefficient of variation
	var patternType string
	var confidence float64

	if cv < 0.15 {
		patternType = "steady"
		confidence = 0.95
	} else if cv < 0.35 {
		patternType = "moderate"
		confidence = 0.85
	} else if cv < 0.70 {
		patternType = "spiky"
		confidence = 0.80
	} else {
		patternType = "highly-variable"
		confidence = 0.75
	}

	return UsagePattern{
		Type:       patternType,
		Variation:  cv,
		Confidence: confidence,
	}
}
