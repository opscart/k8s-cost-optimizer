package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// HistoricalAnalyzer queries and analyzes historical metrics
type HistoricalAnalyzer struct {
	promAPI v1.API
}

// NewHistoricalAnalyzer creates a new analyzer with Prometheus API client
// Pass the client from datasource, but don't import datasource package
func NewHistoricalAnalyzer(promClient api.Client) *HistoricalAnalyzer {
	return &HistoricalAnalyzer{
		promAPI: v1.NewAPI(promClient),
	}
}

// GetHistoricalMetrics fetches 30 days of metrics for a pod
func (h *HistoricalAnalyzer) GetHistoricalMetrics(
	ctx context.Context,
	namespace, podName, containerName string,
	days int,
) (*HistoricalMetrics, error) {

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(days) * 24 * time.Hour)

	resolution := 5 * time.Minute

	metrics := &HistoricalMetrics{
		PodName:       podName,
		Namespace:     namespace,
		ContainerName: containerName,
		StartTime:     startTime,
		EndTime:       endTime,
		Resolution:    resolution,
	}

	// Query CPU usage
	cpuSamples, err := h.queryCPUUsage(ctx, namespace, podName, containerName, startTime, endTime, resolution)
	if err != nil {
		return nil, fmt.Errorf("failed to query CPU usage: %w", err)
	}
	metrics.CPUSamples = cpuSamples

	// Query Memory usage
	memorySamples, err := h.queryMemoryUsage(ctx, namespace, podName, containerName, startTime, endTime, resolution)
	if err != nil {
		return nil, fmt.Errorf("failed to query memory usage: %w", err)
	}
	metrics.MemorySamples = memorySamples

	metrics.SampleCount = len(cpuSamples)

	return metrics, nil
}

// queryCPUUsage queries historical CPU usage from Prometheus
func (h *HistoricalAnalyzer) queryCPUUsage(
	ctx context.Context,
	namespace, podName, containerName string,
	startTime, endTime time.Time,
	step time.Duration,
) ([]MetricSample, error) {

	query := fmt.Sprintf(
		`rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!=""}[5m]) * 1000`,

		namespace, podName,
	)

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	result, warnings, err := h.promAPI.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		fmt.Printf("Warnings from Prometheus: %v\n", warnings)
	}

	samples, err := parsePrometheusResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CPU results: %w", err)
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("no CPU data found for pod %s/%s", namespace, podName)
	}

	return samples, nil
}

// queryMemoryUsage queries historical memory usage from Prometheus
func (h *HistoricalAnalyzer) queryMemoryUsage(
	ctx context.Context,
	namespace, podName, containerName string,
	startTime, endTime time.Time,
	step time.Duration,
) ([]MetricSample, error) {

	query := fmt.Sprintf(
		`container_memory_working_set_bytes{namespace="%s",pod="%s",container!=""}`,
		namespace, podName,
	)

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	result, warnings, err := h.promAPI.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		fmt.Printf("Warnings from Prometheus: %v\n", warnings)
	}

	samples, err := parsePrometheusResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory results: %w", err)
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("no memory data found for pod %s/%s", namespace, podName)
	}

	return samples, nil
}

// parsePrometheusResult converts Prometheus query result to MetricSample array
func parsePrometheusResult(result model.Value) ([]MetricSample, error) {
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	if len(matrix) == 0 {
		return nil, fmt.Errorf("no data in prometheus result")
	}

	var samples []MetricSample

	// Aggregate all series (multiple containers in a pod)
	for _, series := range matrix {
		for _, value := range series.Values {
			samples = append(samples, MetricSample{
				Timestamp: value.Timestamp.Time(),
				Value:     float64(value.Value),
			})
		}
	}

	return samples, nil
}
