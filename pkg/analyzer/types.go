package analyzer

import "time"

// HistoricalMetrics contains 30-day historical data for a pod
type HistoricalMetrics struct {
	PodName       string
	Namespace     string
	ContainerName string
	StartTime     time.Time
	EndTime       time.Time

	// CPU metrics (in millicores)
	CPUSamples []MetricSample

	// Memory metrics (in bytes)
	MemorySamples []MetricSample

	// Metadata
	SampleCount int
	Resolution  time.Duration // e.g., 5 minutes between samples
}

// MetricSample represents a single metric data point
type MetricSample struct {
	Timestamp time.Time
	Value     float64
}

// HistoricalAnalysis is the output after analyzing historical data
type HistoricalAnalysis struct {
	Pod       string
	Namespace string
	Container string

	// CPU Analysis
	CPUPercentiles Percentiles
	CPUPattern     UsagePattern
	CPUGrowth      GrowthTrend

	// Memory Analysis
	MemoryPercentiles Percentiles
	MemoryPattern     UsagePattern
	MemoryGrowth      GrowthTrend

	// Current vs Historical
	CurrentCPURequest        float64
	RecommendedCPURequest    float64
	CurrentMemoryRequest     int64
	RecommendedMemoryRequest int64

	// Reasoning
	Reasoning string
	Risk      string // "LOW", "MEDIUM", "HIGH"
}

// Percentiles contains statistical percentiles
type Percentiles struct {
	Average float64
	P50     float64
	P90     float64
	P95     float64
	P99     float64
	Peak    float64
	Min     float64
}

// UsagePattern describes usage behavior
type UsagePattern struct {
	Type       string  // "steady", "spiky", "periodic", "growing"
	Variation  float64 // Coefficient of variation (0-1)
	Confidence float64 // How confident we are (0-1)
}

// GrowthTrend describes growth over time
type GrowthTrend struct {
	RatePerMonth    float64 // % growth per month
	Confidence      float64
	Predicted3Month float64
	Predicted6Month float64
	IsGrowing       bool
}
