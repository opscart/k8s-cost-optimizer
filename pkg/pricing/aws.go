package pricing

import (
	"context"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// AWSProvider implements AWS EKS pricing
type AWSProvider struct {
	region string
	cache  *PriceCache
}

func NewAWSProvider(region string) *AWSProvider {
	return &AWSProvider{
		region: region,
		cache:  NewPriceCache(24 * time.Hour),
	}
}

func (a *AWSProvider) Name() string {
	return "aws"
}

func (a *AWSProvider) GetCostInfo(ctx context.Context, region, nodeType string) (*models.CostInfo, error) {
	// For now, return typical AWS pricing
	// TODO: Integrate with AWS Pricing API in future
	return &models.CostInfo{
		Provider:         "aws",
		Region:           region,
		CPUCostPerCore:   33.0,  // $/core/month (t3.medium average)
		MemoryCostPerGiB: 4.5,   // $/GiB/month
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}, nil
}

func (a *AWSProvider) CalculateWorkloadCost(ctx context.Context, workload *models.Workload, metrics *models.Metrics) (*models.WorkloadCost, error) {
	costInfo, err := a.GetCostInfo(ctx, a.region, "")
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
