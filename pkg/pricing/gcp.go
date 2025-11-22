package pricing

import (
	"context"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// GCPProvider implements GCP GKE pricing
type GCPProvider struct {
	region string
	cache  *PriceCache
}

func NewGCPProvider(region string) *GCPProvider {
	return &GCPProvider{
		region: region,
		cache:  NewPriceCache(24 * time.Hour),
	}
}

func (g *GCPProvider) Name() string {
	return "gcp"
}

func (g *GCPProvider) GetCostInfo(ctx context.Context, region, nodeType string) (*models.CostInfo, error) {
	// GCP pricing (e2-medium average)
	// TODO: Integrate with GCP Pricing API
	return &models.CostInfo{
		Provider:         "gcp",
		Region:           region,
		CPUCostPerCore:   31.0,  // $/core/month
		MemoryCostPerGiB: 4.2,   // $/GiB/month
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}, nil
}

func (g *GCPProvider) CalculateWorkloadCost(ctx context.Context, workload *models.Workload, metrics *models.Metrics) (*models.WorkloadCost, error) {
	costInfo, err := g.GetCostInfo(ctx, g.region, "")
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
