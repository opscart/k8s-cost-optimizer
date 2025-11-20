package datasource

import (
	"context"
	"fmt"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusSource struct {
	client v1.API
	url    string
}

func NewPrometheusSource(url string) (*PrometheusSource, error) {
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusSource{
		client: v1.NewAPI(client),
		url:    url,
	}, nil
}

// GetMetrics retrieves metrics for a workload (pod-level for MVP)
func (p *PrometheusSource) GetMetrics(ctx context.Context, workload *models.Workload, duration time.Duration) (*models.Metrics, error) {
	// For MVP: Get current instant values (not P95 yet)
	// TODO: Week 2 - implement P95/P99 with quantile_over_time
	
	// Get CPU usage (instant counter value, we'll improve this)
	cpuQuery := fmt.Sprintf(`container_cpu_usage_seconds_total{namespace="%s",pod="%s"}`, 
		workload.Namespace, workload.Pod)
	cpu, err := p.querySingle(ctx, cpuQuery)
	if err != nil {
		return nil, fmt.Errorf("CPU query failed: %w", err)
	}
	
	// Get memory usage
	memQuery := fmt.Sprintf(`container_memory_working_set_bytes{namespace="%s",pod="%s"}`,
		workload.Namespace, workload.Pod)
	mem, err := p.querySingle(ctx, memQuery)
	if err != nil {
		return nil, fmt.Errorf("memory query failed: %w", err)
	}
	
	// Get requests from kube-state-metrics
	reqCPUQuery := fmt.Sprintf(`kube_pod_container_resource_requests{namespace="%s",pod="%s",resource="cpu"}`,
		workload.Namespace, workload.Pod)
	reqCPU, err := p.querySingle(ctx, reqCPUQuery)
	if err != nil {
		reqCPU = 0 // Fallback
	}
	
	reqMemQuery := fmt.Sprintf(`kube_pod_container_resource_requests{namespace="%s",pod="%s",resource="memory"}`,
		workload.Namespace, workload.Pod)
	reqMem, err := p.querySingle(ctx, reqMemQuery)
	if err != nil {
		reqMem = 0 // Fallback
	}
	
	// For MVP: use instant values as approximation
	// Convert CPU cores to millicores
	cpuMillicores := int64(cpu * 1000)
	
	return &models.Metrics{
		P95CPU:          cpuMillicores,  // TODO: Actual P95 in Week 2
		P99CPU:          cpuMillicores,
		MaxCPU:          cpuMillicores,
		AvgCPU:          cpuMillicores,
		P95Memory:       int64(mem),
		P99Memory:       int64(mem),
		MaxMemory:       int64(mem),
		AvgMemory:       int64(mem),
		RequestedCPU:    int64(reqCPU * 1000),
		RequestedMemory: int64(reqMem),
		CollectedAt:     time.Now(),
		Duration:        duration,
		SampleCount:     1,
	}, nil
}

func (p *PrometheusSource) querySingle(ctx context.Context, query string) (float64, error) {
	result, warnings, err := p.client.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	
	if len(warnings) > 0 {
		fmt.Printf("[WARN] Prometheus: %v\n", warnings)
	}
	
	vector, ok := result.(model.Vector)
	if !ok || len(vector) == 0 {
		return 0, fmt.Errorf("no data for query: %s", query)
	}
	
	// Sum all values (in case multiple containers per pod)
	sum := 0.0
	for _, sample := range vector {
		sum += float64(sample.Value)
	}
	
	return sum, nil
}

// Stub implementations
func (p *PrometheusSource) GetTimeseries(ctx context.Context, workload *models.Workload, duration time.Duration, metric string) ([]models.Sample, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PrometheusSource) ListWorkloads(ctx context.Context, namespace string) ([]*models.Workload, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PrometheusSource) IsAvailable(ctx context.Context) bool {
	_, _, err := p.client.Query(ctx, "up", time.Now())
	return err == nil
}

func (p *PrometheusSource) Name() string {
	return "Prometheus"
}
