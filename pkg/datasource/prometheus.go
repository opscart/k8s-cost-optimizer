package datasource

import (
	"context"
	"fmt"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusSource struct {
	apiClient api.Client
	client    v1.API
	url       string
}

func NewPrometheusSource(url string) (*PrometheusSource, error) {
	apiClient, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusSource{
		apiClient: apiClient,            // Store base client
		client:    v1.NewAPI(apiClient), // Create v1 API from base client
		url:       url,
	}, nil
}

// GetMetrics retrieves comprehensive metrics for a workload
func (p *PrometheusSource) GetMetrics(ctx context.Context, workload *models.Workload, duration time.Duration) (*models.Metrics, error) {
	now := time.Now()

	// Try historical queries first, fall back to instant if needed
	p95CPU, p99CPU, maxCPU, avgCPU := p.getCPUMetrics(ctx, workload, duration)
	p95Mem, p99Mem, maxMem, avgMem := p.getMemoryMetrics(ctx, workload, duration)

	// Get requests from kube-state-metrics
	reqCPU, reqMem, err := p.queryRequests(ctx, workload)
	if err != nil {
		fmt.Printf("[WARN] Failed to query resource requests: %v\n", err)
	}

	return &models.Metrics{
		P95CPU:          int64(p95CPU * 1000), // Convert to millicores
		P99CPU:          int64(p99CPU * 1000),
		MaxCPU:          int64(maxCPU * 1000),
		AvgCPU:          int64(avgCPU * 1000),
		P95Memory:       int64(p95Mem),
		P99Memory:       int64(p99Mem),
		MaxMemory:       int64(maxMem),
		AvgMemory:       int64(avgMem),
		RequestedCPU:    int64(reqCPU * 1000),
		RequestedMemory: int64(reqMem),
		CollectedAt:     now,
		Duration:        duration,
	}, nil
}

// getCPUMetrics tries historical queries with automatic fallback
func (p *PrometheusSource) getCPUMetrics(ctx context.Context, workload *models.Workload, duration time.Duration) (p95, p99, max, avg float64) {
	// Try P95 from historical data
	p95, err := p.queryP95CPU(ctx, workload, duration)
	if err != nil || p95 == 0 {
		// Fallback to instant rate
		fmt.Printf("[WARN] P95 CPU unavailable, using instant rate for %s\n", workload.Pod)
		instant, _ := p.queryInstantCPU(ctx, workload)
		p95 = instant
	}

	// Try P99
	p99, err = p.queryP99CPU(ctx, workload, duration)
	if err != nil || p99 == 0 {
		p99 = p95 * 1.05 // 5% higher than P95
	}

	// Try Max
	max, err = p.queryMaxCPU(ctx, workload, duration)
	if err != nil || max == 0 {
		max = p99 * 1.1 // 10% higher than P99
	}

	// Try Avg
	avg, err = p.queryAvgCPU(ctx, workload, duration)
	if err != nil || avg == 0 {
		avg = p95 * 0.7 // Typically 70% of P95
	}

	return p95, p99, max, avg
}

// getMemoryMetrics tries historical queries with automatic fallback
func (p *PrometheusSource) getMemoryMetrics(ctx context.Context, workload *models.Workload, duration time.Duration) (p95, p99, max, avg float64) {
	// Try P95 from historical data
	p95, err := p.queryP95Memory(ctx, workload, duration)
	if err != nil || p95 == 0 {
		// Fallback to instant
		fmt.Printf("[WARN] P95 Memory unavailable, using instant value for %s\n", workload.Pod)
		instant, _ := p.queryInstantMemory(ctx, workload)
		p95 = instant
	}

	// Try P99
	p99, err = p.queryP99Memory(ctx, workload, duration)
	if err != nil || p99 == 0 {
		p99 = p95 * 1.05
	}

	// Try Max
	max, err = p.queryMaxMemory(ctx, workload, duration)
	if err != nil || max == 0 {
		max = p99 * 1.1
	}

	// Try Avg
	avg, err = p.queryAvgMemory(ctx, workload, duration)
	if err != nil || avg == 0 {
		avg = p95 * 0.8
	}

	return p95, p99, max, avg
}

// queryP95CPU uses quantile_over_time for 95th percentile CPU
func (p *PrometheusSource) queryP95CPU(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`quantile_over_time(0.95, rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryP99CPU uses quantile_over_time for 99th percentile CPU
func (p *PrometheusSource) queryP99CPU(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`quantile_over_time(0.99, rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryMaxCPU gets maximum CPU over duration
func (p *PrometheusSource) queryMaxCPU(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`max_over_time(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryAvgCPU gets average CPU over duration
func (p *PrometheusSource) queryAvgCPU(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`avg_over_time(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryP95Memory uses quantile_over_time for 95th percentile memory
func (p *PrometheusSource) queryP95Memory(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`quantile_over_time(0.95, container_memory_working_set_bytes{namespace="%s",pod="%s"}[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryP99Memory uses quantile_over_time for 99th percentile memory
func (p *PrometheusSource) queryP99Memory(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`quantile_over_time(0.99, container_memory_working_set_bytes{namespace="%s",pod="%s"}[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryMaxMemory gets maximum memory over duration
func (p *PrometheusSource) queryMaxMemory(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`max_over_time(container_memory_working_set_bytes{namespace="%s",pod="%s"}[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryAvgMemory gets average memory over duration
func (p *PrometheusSource) queryAvgMemory(ctx context.Context, workload *models.Workload, duration time.Duration) (float64, error) {
	query := fmt.Sprintf(
		`avg_over_time(container_memory_working_set_bytes{namespace="%s",pod="%s"}[%s:1m])`,
		workload.Namespace,
		workload.Pod,
		formatDuration(duration),
	)

	return p.querySingleSum(ctx, query)
}

// queryInstantCPU gets current CPU rate (fallback)
func (p *PrometheusSource) queryInstantCPU(ctx context.Context, workload *models.Workload) (float64, error) {
	query := fmt.Sprintf(
		`rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])`,
		workload.Namespace,
		workload.Pod,
	)

	return p.querySingleSum(ctx, query)
}

// queryInstantMemory gets current memory (fallback)
func (p *PrometheusSource) queryInstantMemory(ctx context.Context, workload *models.Workload) (float64, error) {
	query := fmt.Sprintf(
		`container_memory_working_set_bytes{namespace="%s",pod="%s"}`,
		workload.Namespace,
		workload.Pod,
	)

	return p.querySingleSum(ctx, query)
}

// queryRequests gets resource requests from kube-state-metrics
func (p *PrometheusSource) queryRequests(ctx context.Context, workload *models.Workload) (cpu, memory float64, err error) {
	// CPU requests
	cpuQuery := fmt.Sprintf(
		`kube_pod_container_resource_requests{namespace="%s",pod="%s",resource="cpu"}`,
		workload.Namespace,
		workload.Pod,
	)
	cpu, err = p.querySingleSum(ctx, cpuQuery)
	if err != nil {
		return 0, 0, err
	}

	// Memory requests
	memQuery := fmt.Sprintf(
		`kube_pod_container_resource_requests{namespace="%s",pod="%s",resource="memory"}`,
		workload.Namespace,
		workload.Pod,
	)
	memory, err = p.querySingleSum(ctx, memQuery)
	if err != nil {
		return 0, 0, err
	}

	return cpu, memory, nil
}

// querySingleSum executes a query and sums all results
func (p *PrometheusSource) querySingleSum(ctx context.Context, query string) (float64, error) {
	result, warnings, err := p.client.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}

	if len(warnings) > 0 {
		fmt.Printf("[WARN] Prometheus: %v\n", warnings)
	}

	vector, ok := result.(model.Vector)
	if !ok || len(vector) == 0 {
		return 0, nil // Return 0, not error - let caller handle
	}

	// Sum all values (for pods with multiple containers)
	sum := 0.0
	for _, sample := range vector {
		sum += float64(sample.Value)
	}

	return sum, nil
}

// formatDuration converts Go duration to Prometheus duration format
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours >= 24 {
		days := hours / 24
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dh", hours)
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

// This allows other packages to create their own analyzers without circular imports
func (p *PrometheusSource) GetClient() api.Client {
	return p.apiClient // Return base client, not v1 API
}

// GetHistoricalAnalyzer returns an analyzer for historical queries
func (p *PrometheusSource) GetHistoricalAnalyzer() *analyzer.HistoricalAnalyzer {
	return analyzer.NewHistoricalAnalyzer(p.apiClient) // Use base client
}
