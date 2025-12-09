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
	verbose bool
}

// NewHistoricalAnalyzer creates a new analyzer with Prometheus API client
// Pass the client from datasource, but don't import datasource package
func NewHistoricalAnalyzer(promClient api.Client, verbose bool) *HistoricalAnalyzer {
	return &HistoricalAnalyzer{
		promAPI: v1.NewAPI(promClient),
		verbose: verbose, // ADD THIS
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

	// Calculate CPU pattern analysis
	if len(cpuSamples) > 0 {
		metrics.CPUPattern = AnalyzeUsagePattern(cpuSamples)
	}

	// Calculate Memory pattern analysis
	if len(memorySamples) > 0 {
		metrics.MemoryPattern = AnalyzeUsagePattern(memorySamples)
	}

	// Calculate CPU growth trend
	if len(cpuSamples) >= 10 { // Need minimum samples for trend
		cpuGrowth, err := CalculateGrowthTrend(cpuSamples)
		if err == nil {
			metrics.CPUGrowth = *cpuGrowth
		}
	}

	// Calculate Memory growth trend
	if len(memorySamples) >= 10 {
		memGrowth, err := CalculateGrowthTrend(memorySamples)
		if err == nil {
			metrics.MemoryGrowth = *memGrowth
		}
	}

	// Calculate data quality and confidence
	metrics.DataQuality = calculateDataQuality(len(cpuSamples), endTime.Sub(startTime))
	metrics.HasSufficientData = len(cpuSamples) >= 864 // ~3 days at 5-min intervals

	// Week 9 Day 3: Split samples by weekday/weekend and calculate separate P95
	if len(cpuSamples) > 0 {
		weekdayCPU, weekendCPU := SplitSamplesByWeekday(cpuSamples)

		if len(weekdayCPU) > 0 {
			weekdayPercentiles, err := CalculatePercentiles(weekdayCPU)
			if err == nil {
				metrics.WeekdayCPUP95 = weekdayPercentiles.P95
			}
		}

		if len(weekendCPU) > 0 {
			weekendPercentiles, err := CalculatePercentiles(weekendCPU)
			if err == nil {
				metrics.WeekendCPUP95 = weekendPercentiles.P95
			}
		}
	}

	if len(memorySamples) > 0 {
		weekdayMem, weekendMem := SplitSamplesByWeekday(memorySamples)

		if len(weekdayMem) > 0 {
			weekdayPercentiles, err := CalculatePercentiles(weekdayMem)
			if err == nil {
				metrics.WeekdayMemoryP95 = uint64(weekdayPercentiles.P95)
			}
		}

		if len(weekendMem) > 0 {
			weekendPercentiles, err := CalculatePercentiles(weekendMem)
			if err == nil {
				metrics.WeekendMemoryP95 = uint64(weekendPercentiles.P95)
			}
		}
	}

	if h.verbose {
		fmt.Printf("[DEBUG] Pattern Analysis - CPU: %s (CV: %.2f), Memory: %s (CV: %.2f)\n",
			metrics.CPUPattern.Type, metrics.CPUPattern.Variation,
			metrics.MemoryPattern.Type, metrics.MemoryPattern.Variation)

		if metrics.CPUGrowth.IsGrowing {
			fmt.Printf("[DEBUG] Growth Detected - CPU: %.1f%%/month, Predicted 3mo: %.0fm\n",
				metrics.CPUGrowth.RatePerMonth, metrics.CPUGrowth.Predicted3Month)
		}
		// Week 9 Day 3: Show weekday/weekend split
		if metrics.WeekdayCPUP95 > 0 || metrics.WeekendCPUP95 > 0 {
			fmt.Printf("[DEBUG] Weekday/Weekend Split - CPU: Weekday P95: %.0fm, Weekend P95: %.0fm\n",
				metrics.WeekdayCPUP95, metrics.WeekendCPUP95)
			fmt.Printf("[DEBUG] Weekday/Weekend Split - Memory: Weekday P95: %dMi, Weekend P95: %dMi\n",
				metrics.WeekdayMemoryP95/(1024*1024), metrics.WeekendMemoryP95/(1024*1024))
		}
	}

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
		`container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!="POD"}`,
		namespace, podName,
	)

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	// Conditional debug output
	if h.verbose {
		fmt.Printf("[DEBUG] Prometheus CPU query: %s\n", query)
		fmt.Printf("[DEBUG] Time range: %s to %s (step: %s)\n",
			startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), step)
	}

	result, warnings, err := h.promAPI.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 && h.verbose {
		fmt.Printf("[DEBUG] Prometheus warnings: %v\n", warnings)
	}

	if h.verbose {
		fmt.Printf("[DEBUG] Result type: %T\n", result)
		if matrix, ok := result.(model.Matrix); ok {
			fmt.Printf("[DEBUG] Matrix length: %d\n", len(matrix))
			if len(matrix) > 0 {
				fmt.Printf("[DEBUG] First series has %d values\n", len(matrix[0].Values))
			}
		}
	}

	samples, err := parsePrometheusResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CPU results: %w", err)
	}

	// Convert counter to rate (millicores)
	if len(samples) > 0 {
		samples = calculateRateFromCounter(samples)
	}

	if len(samples) == 0 {
		return []MetricSample{}, nil
	}

	if h.verbose {
		fmt.Printf("[DEBUG] Parsed %d CPU samples\n", len(samples))
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
		`container_memory_working_set_bytes{namespace="%s",pod="%s",container!="POD"}`,
		namespace, podName,
	)

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	if h.verbose {
		fmt.Printf("[DEBUG] Prometheus Memory query: %s\n", query)
	}

	result, warnings, err := h.promAPI.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 && h.verbose {
		fmt.Printf("[DEBUG] Prometheus warnings: %v\n", warnings)
	}

	samples, err := parsePrometheusResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory results: %w", err)
	}

	if len(samples) == 0 {
		return []MetricSample{}, nil
	}

	if h.verbose {
		fmt.Printf("[DEBUG] Parsed %d memory samples\n", len(samples))
	}
	return samples, nil
}

// parsePrometheusResult converts Prometheus query result to MetricSample array
func parsePrometheusResult(result model.Value) ([]MetricSample, error) {
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Allow empty matrix - return empty samples instead of error
	if len(matrix) == 0 {
		return []MetricSample{}, nil // Changed from error to empty slice
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

// calculateRateFromCounter converts CPU counter values to per-second rate in millicores
func calculateRateFromCounter(samples []MetricSample) []MetricSample {
	if len(samples) < 2 {
		return samples
	}

	rates := make([]MetricSample, 0, len(samples)-1)

	for i := 1; i < len(samples); i++ {
		timeDiff := samples[i].Timestamp.Sub(samples[i-1].Timestamp).Seconds()
		if timeDiff > 0 {
			valueDiff := samples[i].Value - samples[i-1].Value
			// Convert to millicores (rate per second * 1000)
			rate := (valueDiff / timeDiff) * 1000

			rates = append(rates, MetricSample{
				Timestamp: samples[i].Timestamp,
				Value:     rate,
			})
		}
	}

	return rates
}

// calculateDataQuality returns a quality score (0.0-1.0) based on sample count and time span
func calculateDataQuality(sampleCount int, timeSpan time.Duration) float64 {
	// Ideal: 7 days * 288 samples/day (5-min intervals) = ~2000 samples
	idealSamples := 2000.0
	sampleScore := float64(sampleCount) / idealSamples
	if sampleScore > 1.0 {
		sampleScore = 1.0
	}

	// Time span quality (ideal: 7 days)
	idealDays := 7.0
	actualDays := timeSpan.Hours() / 24.0
	timeScore := actualDays / idealDays
	if timeScore > 1.0 {
		timeScore = 1.0
	}

	// Combined score (weighted average: 60% samples, 40% time)
	quality := (sampleScore * 0.6) + (timeScore * 0.4)
	return quality
}
