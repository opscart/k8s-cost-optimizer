package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/datasource"
)

func main() {
	promURL := flag.String("prometheus-url", "http://localhost:9090", "Prometheus URL")
	namespace := flag.String("namespace", "default", "Namespace to analyze")
	podName := flag.String("pod", "", "Pod name to analyze")
	containerName := flag.String("container", "", "Container name to analyze")
	days := flag.Int("days", 7, "Number of days of history")

	flag.Parse()

	if *podName == "" || *containerName == "" {
		log.Fatal("Please provide --pod and --container flags")
	}

	fmt.Printf("Historical Analysis - Real Prometheus Data\n")
	fmt.Printf("==========================================\n\n")
	fmt.Printf("Prometheus: %s\n", *promURL)
	fmt.Printf("Namespace: %s\n", *namespace)
	fmt.Printf("Pod: %s\n", *podName)
	fmt.Printf("Container: %s\n", *containerName)
	fmt.Printf("Days: %d\n\n", *days)

	// Create Prometheus datasource
	promDS, err := datasource.NewPrometheusSource(*promURL)
	if err != nil {
		log.Fatalf("Failed to create Prometheus datasource: %v", err)
	}

	// Get client and create analyzer
	promClient := promDS.GetClient()
	historicalAnalyzer := analyzer.NewHistoricalAnalyzer(promClient)

	// Fetch historical metrics
	fmt.Printf("Fetching %d days of data...\n", *days)
	startTime := time.Now()

	metrics, err := historicalAnalyzer.GetHistoricalMetrics(
		context.Background(),
		*namespace,
		*podName,
		*containerName,
		*days,
	)

	if err != nil {
		log.Fatalf("Failed to get historical metrics: %v", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("✓ Fetched in %v\n\n", elapsed)

	// Display results
	fmt.Printf("Data Summary\n")
	fmt.Printf("------------\n")
	fmt.Printf("Period: %s to %s\n",
		metrics.StartTime.Format("2006-01-02 15:04"),
		metrics.EndTime.Format("2006-01-02 15:04"),
	)
	fmt.Printf("Resolution: %v\n", metrics.Resolution)
	fmt.Printf("CPU Samples: %d\n", len(metrics.CPUSamples))
	fmt.Printf("Memory Samples: %d\n\n", len(metrics.MemorySamples))

	// CPU Analysis
	if len(metrics.CPUSamples) > 0 {
		fmt.Printf("CPU Analysis\n")
		fmt.Printf("------------\n")

		cpuPercentiles, _ := analyzer.CalculatePercentiles(metrics.CPUSamples)
		fmt.Printf("Percentiles:\n")
		fmt.Printf("  Average: %.2f millicores\n", cpuPercentiles.Average)
		fmt.Printf("  P50: %.2f millicores\n", cpuPercentiles.P50)
		fmt.Printf("  P90: %.2f millicores\n", cpuPercentiles.P90)
		fmt.Printf("  P95: %.2f millicores\n", cpuPercentiles.P95)
		fmt.Printf("  P99: %.2f millicores\n", cpuPercentiles.P99)
		fmt.Printf("  Peak: %.2f millicores\n", cpuPercentiles.Peak)
		fmt.Printf("  Min: %.2f millicores\n\n", cpuPercentiles.Min)

		cpuPattern := analyzer.AnalyzeUsagePattern(metrics.CPUSamples)
		fmt.Printf("Pattern: %s (variation: %.2f%%)\n", cpuPattern.Type, cpuPattern.Variation*100)
		fmt.Printf("Confidence: %.0f%%\n\n", cpuPattern.Confidence*100)

		if len(metrics.CPUSamples) >= 100 {
			cpuGrowth, err := analyzer.CalculateGrowthTrend(metrics.CPUSamples)
			if err == nil {
				fmt.Printf("Growth Trend:\n")
				fmt.Printf("  Rate: %.2f%% per month\n", cpuGrowth.RatePerMonth)
				fmt.Printf("  Is Growing: %v\n", cpuGrowth.IsGrowing)
				if cpuGrowth.IsGrowing {
					fmt.Printf("  Predicted (3 months): %.2f millicores\n", cpuGrowth.Predicted3Month)
					fmt.Printf("  Predicted (6 months): %.2f millicores\n", cpuGrowth.Predicted6Month)
				}
				fmt.Printf("  Trend Confidence: %.0f%%\n\n", cpuGrowth.Confidence*100)
			}
		} else {
			fmt.Printf("Growth Trend: Not enough data (need 100+ samples)\n\n")
		}
	}

	// Memory Analysis
	if len(metrics.MemorySamples) > 0 {
		fmt.Printf("Memory Analysis\n")
		fmt.Printf("---------------\n")

		memPercentiles, _ := analyzer.CalculatePercentiles(metrics.MemorySamples)
		fmt.Printf("Percentiles:\n")
		fmt.Printf("  Average: %.2f MB\n", memPercentiles.Average/1024/1024)
		fmt.Printf("  P95: %.2f MB\n", memPercentiles.P95/1024/1024)
		fmt.Printf("  P99: %.2f MB\n", memPercentiles.P99/1024/1024)
		fmt.Printf("  Peak: %.2f MB\n\n", memPercentiles.Peak/1024/1024)

		memPattern := analyzer.AnalyzeUsagePattern(metrics.MemorySamples)
		fmt.Printf("Pattern: %s (variation: %.2f%%)\n\n", memPattern.Type, memPattern.Variation*100)
	}

	fmt.Printf("✓ Analysis complete!\n")
}

// Add this at the very end of main.go, after main() function:

/*
To run mock test instead of real Prometheus:
  go run cmd/test-historical/*.go --mock
*/
