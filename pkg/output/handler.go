package output

import (
	"context"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// Handler defines the interface for output formatting
type Handler interface {
	DisplayRecommendations(ctx context.Context, recommendations []*models.Recommendation) error
	DisplaySummary(ctx context.Context, totalSavings float64, count int) error
	Format() string
}
