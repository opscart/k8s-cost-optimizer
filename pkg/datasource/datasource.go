package datasource

import (
	"context"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// DataSource defines the interface for collecting metrics
type DataSource interface {
	GetMetrics(ctx context.Context, workload *models.Workload, duration time.Duration) (*models.Metrics, error)
	GetTimeseries(ctx context.Context, workload *models.Workload, duration time.Duration, metric string) ([]models.Sample, error)
	ListWorkloads(ctx context.Context, namespace string) ([]*models.Workload, error)
	IsAvailable(ctx context.Context) bool
	Name() string
}

type Config struct {
	PrometheusURL    string
	UseMetricsServer bool
	Timeout          time.Duration
}
