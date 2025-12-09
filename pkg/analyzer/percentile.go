package analyzer

import (
	"fmt"
	"math"
	"sort"
	"time"
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

// SplitSamplesByWeekday separates samples into weekday and weekend buckets
func SplitSamplesByWeekday(samples []MetricSample) (weekday []MetricSample, weekend []MetricSample) {
	for _, sample := range samples {
		// time.Weekday: Sunday = 0, Monday = 1, ..., Saturday = 6
		day := sample.Timestamp.Weekday()
		if day == time.Saturday || day == time.Sunday {
			weekend = append(weekend, sample)
		} else {
			weekday = append(weekday, sample)
		}
	}
	return weekday, weekend
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

// CalculatePercentilesFromValues calculates percentiles from a slice of float64 values
func CalculatePercentilesFromValues(values []float64) Percentiles {
	if len(values) == 0 {
		return Percentiles{}
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate statistics
	sum := 0.0
	min := sorted[0]
	max := sorted[len(sorted)-1]

	for _, v := range sorted {
		sum += v
	}
	avg := sum / float64(len(sorted))

	return Percentiles{
		Average: avg,
		P50:     getPercentile(sorted, 0.50),
		P90:     getPercentile(sorted, 0.90),
		P95:     getPercentile(sorted, 0.95),
		P99:     getPercentile(sorted, 0.99),
		Peak:    max,
		Min:     min,
	}
}

// getPercentile gets the value at a given percentile (0.0 to 1.0)
func getPercentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := int(float64(len(sorted)-1) * percentile)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
