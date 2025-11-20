package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/datasource"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

func main() {
	prometheusURL := "http://localhost:9090"
	if url := os.Getenv("PROMETHEUS_URL"); url != "" {
		prometheusURL = url
	}

	fmt.Println("[INFO] Connecting to Prometheus:", prometheusURL)
	
	source, err := datasource.NewPrometheusSource(prometheusURL)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create Prometheus source: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if !source.IsAvailable(ctx) {
		fmt.Println("[ERROR] Prometheus is not available")
		os.Exit(1)
	}
	fmt.Println("[INFO] Prometheus is available")

	testPods := []string{
		"idle-app-666df6866b-k8zlr",
		"overprovision-app-846db48d89-kstp8",
		"normal-app-594df877fc-b86vj",
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Testing PrometheusSource with cost-test namespace pods")
	fmt.Println("NOTE: P95/P99 require 24-48h of data for accuracy")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	// Test with different durations
	durations := []struct {
		name string
		dur  time.Duration
	}{
		{"1 hour", 1 * time.Hour},
		{"12 hours", 12 * time.Hour},
		{"7 days", 7 * 24 * time.Hour},
	}

	for _, podName := range testPods {
		workload := &models.Workload{
			Namespace: "cost-test",
			Pod:       podName,
		}

		fmt.Printf("Pod: %s\n", podName)
		fmt.Println(strings.Repeat("-", 80))

		for _, d := range durations {
			fmt.Printf("  [%s]\n", d.name)
			
			metrics, err := source.GetMetrics(ctx, workload, d.dur)
			if err != nil {
				fmt.Printf("    ERROR: %v\n", err)
				continue
			}

			// Show all metrics
			fmt.Printf("    CPU:  Avg=%dm  P95=%dm  P99=%dm  Max=%dm\n",
				metrics.AvgCPU, metrics.P95CPU, metrics.P99CPU, metrics.MaxCPU)
			fmt.Printf("    Mem:  Avg=%dMi P95=%dMi P99=%dMi Max=%dMi\n",
				metrics.AvgMemory/(1024*1024),
				metrics.P95Memory/(1024*1024),
				metrics.P99Memory/(1024*1024),
				metrics.MaxMemory/(1024*1024))
			
			// Show requests
			fmt.Printf("    Requested: CPU=%dm Memory=%dMi\n",
				metrics.RequestedCPU, metrics.RequestedMemory/(1024*1024))
			
			// Calculate utilization from P95
			if metrics.RequestedCPU > 0 && metrics.P95CPU > 0 {
				util := float64(metrics.P95CPU) / float64(metrics.RequestedCPU) * 100
				fmt.Printf("    P95 Utilization: CPU=%.1f%%", util)
			}
			if metrics.RequestedMemory > 0 && metrics.P95Memory > 0 {
				util := float64(metrics.P95Memory) / float64(metrics.RequestedMemory) * 100
				fmt.Printf(" Memory=%.1f%%", util)
			}
			fmt.Println()
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("[INFO] Test complete!")
	fmt.Println("[NOTE] If CPU shows 0m, there's insufficient historical data.")
	fmt.Println("[NOTE] Pods need 24-48h runtime for accurate P95/P99 metrics.")
}
