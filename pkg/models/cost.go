package models

// CostInfo represents pricing information
type CostInfo struct {
	CPUCostPerCoreHour    float64
	MemoryCostPerGBHour   float64
	StorageCostPerGBMonth float64
	Region                string
	NodeType              string
}

// WorkloadCost represents the cost of a workload
type WorkloadCost struct {
	Workload         *Workload
	MonthlyCPUCost   float64
	MonthlyMemoryCost float64
	TotalMonthlyCost float64
}
