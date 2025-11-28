package models

import "time"

// RecommendationType represents the type of recommendation
type RecommendationType string

const (
	RecommendationRightSize RecommendationType = "RIGHT_SIZE"
	RecommendationScaleDown RecommendationType = "SCALE_DOWN"
	RecommendationNoAction  RecommendationType = "NO_ACTION"
)

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID          string
	Type        RecommendationType
	Workload    *Workload
	Environment string

	// Current state
	CurrentCPU    int64
	CurrentMemory int64

	// Recommended state
	RecommendedCPU    int64
	RecommendedMemory int64

	// Analysis
	Reason         string
	SavingsMonthly float64
	Impact         string // HIGH, MEDIUM, LOW
	Risk           RiskLevel

	// Generated command
	Command string

	// Metadata
	CreatedAt time.Time
	AppliedAt *time.Time
	AppliedBy string
}

// AuditEntry represents an action taken
type AuditEntry struct {
	ID               string
	RecommendationID string
	Action           string // APPLIED, ROLLED_BACK
	Status           string // SUCCESS, FAILED
	ErrorMessage     string
	ExecutedBy       string
	ExecutedAt       time.Time
}
