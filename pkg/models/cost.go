package models

import "time"

// CostInfo represents pricing information from cloud providers
type CostInfo struct {
	Provider         string    // azure, aws, gcp, default
	Region           string    // cloud region
	CPUCostPerCore   float64   // $/core/month
	MemoryCostPerGiB float64   // $/GiB/month
	Currency         string    // USD, EUR, etc.
	LastUpdated      time.Time
}

// WorkloadCost represents the cost analysis of a workload
type WorkloadCost struct {
	Workload               *Workload
	CurrentMonthlyCost     float64   // Current cost based on requests
	RecommendedMonthlyCost float64   // Recommended cost based on P95
	MonthlySavings         float64   // Potential savings
	Currency               string
	Provider               string
	CalculatedAt           time.Time
}

// CostSummary aggregates costs across multiple workloads
type CostSummary struct {
	TotalCurrentCost     float64
	TotalRecommendedCost float64
	TotalSavings         float64
	WorkloadCount        int
	Currency             string
	Provider             string
}
