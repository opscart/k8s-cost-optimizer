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
	fmt.Println(strings.Repeat("=", 80) + "\n")

	for _, podName := range testPods {
		workload := &models.Workload{
			Namespace: "cost-test",
			Pod:       podName,
		}

		fmt.Printf("Pod: %s\n", podName)
		fmt.Println(strings.Repeat("-", 40))

		metrics, err := source.GetMetrics(ctx, workload, 7*24*time.Hour)
		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			continue
		}

		fmt.Printf("  CPU:\n")
		fmt.Printf("    Current:   %dm\n", metrics.AvgCPU)
		fmt.Printf("    Requested: %dm\n", metrics.RequestedCPU)
		if metrics.RequestedCPU > 0 {
			util := float64(metrics.AvgCPU) / float64(metrics.RequestedCPU) * 100
			fmt.Printf("    Utilization: %.1f%%\n", util)
		}

		fmt.Printf("  Memory:\n")
		fmt.Printf("    Current:   %dMi\n", metrics.AvgMemory/(1024*1024))
		fmt.Printf("    Requested: %dMi\n", metrics.RequestedMemory/(1024*1024))
		if metrics.RequestedMemory > 0 {
			util := float64(metrics.AvgMemory) / float64(metrics.RequestedMemory) * 100
			fmt.Printf("    Utilization: %.1f%%\n", util)
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("[INFO] Test complete!")
}
