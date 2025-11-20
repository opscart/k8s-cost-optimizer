package pricing

import (
	"context"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// Provider defines the interface for cloud pricing data
type Provider interface {
	GetCostInfo(ctx context.Context, region, nodeType string) (*models.CostInfo, error)
	CalculateWorkloadCost(ctx context.Context, workload *models.Workload, metrics *models.Metrics) (*models.WorkloadCost, error)
	Name() string
}

type Config struct {
	Provider      string
	Region        string
	CacheTTL      int
	DefaultCPU    float64
	DefaultMemory float64
}
