package models

import "time"

// SavingsTrend represents cost savings trend over time
type SavingsTrend struct {
	Namespace             string
	Days                  int
	DataPoints            []SavingsDataPoint
	TotalPotentialSavings float64
	TotalRealizedSavings  float64
	TotalRecommendations  int
	TotalApplied          int
	AdoptionRate          float64
}

// SavingsDataPoint represents a single day's data
type SavingsDataPoint struct {
	Date                time.Time
	RecommendationCount int
	PotentialSavings    float64
	AppliedCount        int
	RealizedSavings     float64
}

// DashboardStats represents aggregate statistics for dashboard
type DashboardStats struct {
	Namespace                   string
	PeriodDays                  int
	TotalRecommendations        int
	AppliedCount                int
	PotentialSavings            float64
	RealizedSavings             float64
	UniqueWorkloads             int
	AvgSavingsPerRecommendation float64
	AdoptionRate                float64
}

// PerformanceComparison compares current vs previous period
type PerformanceComparison struct {
	CurrentPeriodDays       int
	CurrentRecommendations  int
	CurrentSavings          float64
	PreviousRecommendations int
	PreviousSavings         float64
	RecommendationChange    float64
	SavingsChange           float64
}

// WorkloadTrend tracks a single workload over time
type WorkloadTrend struct {
	Namespace            string
	Deployment           string
	Recommendations      []*Recommendation
	FirstSeen            time.Time
	LastSeen             time.Time
	TotalRecommendations int
	AppliedCount         int
	AvgSavings           float64
	Status               string
}
