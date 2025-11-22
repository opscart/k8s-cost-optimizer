package pricing

import (
	"context"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// DefaultProvider provides fallback pricing for on-prem or unknown clouds
type DefaultProvider struct {
	cpuCost    float64
	memoryCost float64
}

func NewDefaultProvider(cpuCost, memoryCost float64) *DefaultProvider {
	if cpuCost == 0 {
		cpuCost = 23.0 // Conservative default
	}
	if memoryCost == 0 {
		memoryCost = 3.0
	}
	return &DefaultProvider{
		cpuCost:    cpuCost,
		memoryCost: memoryCost,
	}
}

func (d *DefaultProvider) Name() string {
	return "default"
}

func (d *DefaultProvider) GetCostInfo(ctx context.Context, region, nodeType string) (*models.CostInfo, error) {
	return &models.CostInfo{
		Provider:         "default",
		Region:           "unknown",
		CPUCostPerCore:   d.cpuCost,
		MemoryCostPerGiB: d.memoryCost,
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}, nil
}

func (d *DefaultProvider) CalculateWorkloadCost(ctx context.Context, workload *models.Workload, metrics *models.Metrics) (*models.WorkloadCost, error) {
	costInfo, err := d.GetCostInfo(ctx, "", "")
	if err != nil {
		return nil, err
	}

	cpuCores := float64(metrics.RequestedCPU) / 1000.0
	memoryGiB := float64(metrics.RequestedMemory) / (1024.0 * 1024.0 * 1024.0)
	currentCost := (cpuCores * costInfo.CPUCostPerCore) + (memoryGiB * costInfo.MemoryCostPerGiB)

	recommendedCPU := float64(metrics.P95CPU) * 1.5 / 1000.0
	recommendedMemory := float64(metrics.P95Memory) * 1.5 / (1024.0 * 1024.0 * 1024.0)
	recommendedCost := (recommendedCPU * costInfo.CPUCostPerCore) + (recommendedMemory * costInfo.MemoryCostPerGiB)

	return &models.WorkloadCost{
		Workload:              workload,
		CurrentMonthlyCost:    currentCost,
		RecommendedMonthlyCost: recommendedCost,
		MonthlySavings:        currentCost - recommendedCost,
		Currency:              costInfo.Currency,
		Provider:              costInfo.Provider,
		CalculatedAt:          time.Now(),
	}, nil
}
