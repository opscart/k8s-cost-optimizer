package storage

import (
	"context"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// Store defines the interface for persistent storage
type Store interface {
	SaveRecommendation(ctx context.Context, rec *models.Recommendation) error
	GetRecommendation(ctx context.Context, id string) (*models.Recommendation, error)
	ListRecommendations(ctx context.Context, namespace string, limit int) ([]*models.Recommendation, error)
	UpdateRecommendation(ctx context.Context, rec *models.Recommendation) error
	
	LogAction(ctx context.Context, entry *models.AuditEntry) error
	GetAuditLog(ctx context.Context, recommendationID string) ([]*models.AuditEntry, error)
	
	CacheMetrics(ctx context.Context, workload *models.Workload, metrics *models.Metrics) error
	GetCachedMetrics(ctx context.Context, workload *models.Workload) (*models.Metrics, error)
	
	Ping(ctx context.Context) error
	Close() error
}

type Config struct {
	Type    string
	Path    string
	URL     string
	Timeout int
}
