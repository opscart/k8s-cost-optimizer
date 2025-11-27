package main

import (
	"fmt"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
)

func main() {
	fmt.Printf("Complete Historical Analysis - Mock Data Test\n")
	fmt.Printf("==============================================\n\n")

	metrics := createMockHistoricalMetrics()

	fmt.Printf("Step 1: Data Collection\n")
	fmt.Printf("-----------------------\n")
	fmt.Printf("Pod: %s/%s\n", metrics.Namespace, metrics.PodName)
	fmt.Printf("Container: %s\n", metrics.ContainerName)
	fmt.Printf("Period: %s to %s (7 days)\n",
		metrics.StartTime.Format("2006-01-02"),
		metrics.EndTime.Format("2006-01-02"),
	)
	fmt.Printf("Samples: %d CPU, %d Memory\n\n",
		len(metrics.CPUSamples), len(metrics.MemorySamples))

	// CPU Analysis
	fmt.Printf("Step 2: CPU Analysis\n")
	fmt.Printf("--------------------\n")

	cpuPercentiles, _ := analyzer.CalculatePercentiles(metrics.CPUSamples)
	fmt.Printf("Percentiles:\n")
	fmt.Printf("  Average: %.2f millicores\n", cpuPercentiles.Average)
	fmt.Printf("  P50: %.2f millicores\n", cpuPercentiles.P50)
	fmt.Printf("  P90: %.2f millicores\n", cpuPercentiles.P90)
	fmt.Printf("  P95: %.2f millicores\n", cpuPercentiles.P95)
	fmt.Printf("  P99: %.2f millicores\n", cpuPercentiles.P99)
	fmt.Printf("  Peak: %.2f millicores\n\n", cpuPercentiles.Peak)

	cpuPattern := analyzer.AnalyzeUsagePattern(metrics.CPUSamples)
	fmt.Printf("Usage Pattern: %s (variation: %.2f%%)\n", cpuPattern.Type, cpuPattern.Variation*100)
	fmt.Printf("Confidence: %.0f%%\n\n", cpuPattern.Confidence*100)

	cpuGrowth, err := analyzer.CalculateGrowthTrend(metrics.CPUSamples)
	if err == nil {
		fmt.Printf("Growth Trend:\n")
		fmt.Printf("  Rate: %.2f%% per month\n", cpuGrowth.RatePerMonth)
		fmt.Printf("  Is Growing: %v\n", cpuGrowth.IsGrowing)
		fmt.Printf("  Predicted (3 months): %.2f millicores\n", cpuGrowth.Predicted3Month)
		fmt.Printf("  Predicted (6 months): %.2f millicores\n", cpuGrowth.Predicted6Month)
		fmt.Printf("  Confidence: %.0f%%\n\n", cpuGrowth.Confidence*100)
	}

	// Memory Analysis
	fmt.Printf("Step 3: Memory Analysis\n")
	fmt.Printf("-----------------------\n")

	memPercentiles, _ := analyzer.CalculatePercentiles(metrics.MemorySamples)
	fmt.Printf("Percentiles:\n")
	fmt.Printf("  Average: %.2f MB\n", memPercentiles.Average/1024/1024)
	fmt.Printf("  P95: %.2f MB\n", memPercentiles.P95/1024/1024)
	fmt.Printf("  P99: %.2f MB\n", memPercentiles.P99/1024/1024)
	fmt.Printf("  Peak: %.2f MB\n\n", memPercentiles.Peak/1024/1024)

	// Recommendation
	fmt.Printf("Step 4: Smart Recommendation\n")
	fmt.Printf("----------------------------\n")

	currentRequest := 1000.0
	recommendedRequest := cpuPercentiles.P99 * 1.15

	fmt.Printf("Current CPU Request: %.0f millicores\n", currentRequest)
	fmt.Printf("Recommended: %.0f millicores\n", recommendedRequest)
	fmt.Printf("Reasoning: Based on P99 (%.0fm) + 15%% safety buffer\n", cpuPercentiles.P99)

	if recommendedRequest < currentRequest {
		savings := ((currentRequest - recommendedRequest) / currentRequest) * 100
		fmt.Printf("Potential Savings: %.0f%% reduction\n", savings)
	} else {
		increase := ((recommendedRequest - currentRequest) / currentRequest) * 100
		fmt.Printf("⚠️  Recommendation: Increase by %.0f%% (currently under-provisioned)\n", increase)
	}

	fmt.Printf("\n✓ Complete analysis finished!\n")
}

func createMockHistoricalMetrics() *analyzer.HistoricalMetrics {
	now := time.Now()
	startTime := now.Add(-7 * 24 * time.Hour)
	numSamples := 2016

	metrics := &analyzer.HistoricalMetrics{
		PodName:       "test-api-xyz",
		Namespace:     "production",
		ContainerName: "api",
		StartTime:     startTime,
		EndTime:       now,
		Resolution:    5 * time.Minute,
		CPUSamples:    make([]analyzer.MetricSample, numSamples),
		MemorySamples: make([]analyzer.MetricSample, numSamples),
	}

	for i := 0; i < numSamples; i++ {
		timestamp := startTime.Add(time.Duration(i) * 5 * time.Minute)
		hour := timestamp.Hour()

		// CPU with business hours pattern
		cpuBase := 150.0
		if hour >= 9 && hour <= 17 {
			cpuBase += 100.0
		}
		cpuGrowth := float64(i) * 0.02
		cpuVariation := (float64(i%20) - 10) * 5.0
		cpuSpike := 0.0
		if i%100 < 5 {
			cpuSpike = 200.0
		}

		metrics.CPUSamples[i] = analyzer.MetricSample{
			Timestamp: timestamp,
			Value:     cpuBase + cpuGrowth + cpuVariation + cpuSpike,
		}

		// Memory: slow growth
		memBase := 512.0 * 1024 * 1024
		memGrowth := float64(i) * 500.0
		memVariation := (float64(i%5) - 2.5) * 1024 * 1024

		metrics.MemorySamples[i] = analyzer.MetricSample{
			Timestamp: timestamp,
			Value:     memBase + memGrowth + memVariation,
		}
	}

	return metrics
}
