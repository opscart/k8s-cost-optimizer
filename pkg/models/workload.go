package models

import "time"

// Workload represents a Kubernetes workload
type Workload struct {
	Namespace   string
	Deployment  string
	Pod         string
	Container   string
	ClusterID   string
}

// Metrics represents usage metrics for a workload
type Metrics struct {
	// CPU in millicores
	P95CPU      int64
	P99CPU      int64
	MaxCPU      int64
	AvgCPU      int64
	
	// Memory in bytes
	P95Memory   int64
	P99Memory   int64
	MaxMemory   int64
	AvgMemory   int64
	
	// Current requests
	RequestedCPU    int64
	RequestedMemory int64
	
	// Metadata
	SampleCount     int
	CollectedAt     time.Time
	Duration        time.Duration
}

// Sample represents a single metric sample
type Sample struct {
	Timestamp time.Time
	Value     float64
}

// RiskLevel represents the risk of applying a recommendation
type RiskLevel string

const (
	RiskNone   RiskLevel = "NONE"
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)
