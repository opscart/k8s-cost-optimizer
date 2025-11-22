package main

import (
	"context"
	"fmt"
	"os"

	"github.com/opscart/k8s-cost-optimizer/pkg/config"
	"github.com/opscart/k8s-cost-optimizer/pkg/datasource"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

func main() {
	fmt.Println("=== Prometheus Lookback Period Test ===\n")

	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://localhost:9090"
	}

	// Test if Prometheus is available
	prom, err := datasource.NewPrometheusSource(prometheusURL)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create Prometheus client: %v\n", err)
		fmt.Println("\nPrometheus not available. This test shows how it would work:")
		showSimulation()
		return
	}

	ctx := context.Background()
	if !prom.IsAvailable(ctx) {
		fmt.Printf("[WARN] Prometheus not available at %s\n", prometheusURL)
		fmt.Println("\nShowing simulation of how lookback periods work:")
		showSimulation()
		return
	}

	fmt.Printf("[SUCCESS] Connected to Prometheus at %s\n\n", prometheusURL)

	// Test with different lookback periods
	testWorkload := &models.Workload{
		Namespace: "cost-test",
		Pod:       "idle-app",
	}

	periods := []struct {
		name string
		cfg  *config.Config
	}{
		{"Dev (3 days)", func() *config.Config { c := config.NewConfig(); c.UseDevPreset(); return c }()},
		{"Default (7 days)", config.NewConfig()},
		{"Production (14 days)", func() *config.Config { c := config.NewConfig(); c.UseProductionPreset(); return c }()},
		{"Custom (15 days)", func() *config.Config { c := config.NewConfig(); c.MetricsLookbackDays = 15; c.MetricsDuration = 360 * 60 * 60 * 1000000000; return c }()},
	}

	fmt.Println("Comparing different lookback periods:\n")
	for _, p := range periods {
		fmt.Printf("[%s]\n", p.name)
		
		metrics, err := prom.GetMetrics(ctx, testWorkload, p.cfg.MetricsDuration)
		if err != nil {
			fmt.Printf("  Error: %v\n\n", err)
			continue
		}

		fmt.Printf("  P95 CPU: %dm, P95 Memory: %dMi\n", 
			metrics.P95CPU, metrics.P95Memory/(1024*1024))
		fmt.Printf("  P99 CPU: %dm, P99 Memory: %dMi\n", 
			metrics.P99CPU, metrics.P99Memory/(1024*1024))
		
		// Calculate recommended with safety buffer
		recCPU := int64(float64(metrics.P95CPU) * p.cfg.SafetyBuffer)
		recMem := int64(float64(metrics.P95Memory) * p.cfg.SafetyBuffer)
		
		fmt.Printf("  Recommended (%.1fx buffer): %dm CPU, %dMi memory\n\n", 
			p.cfg.SafetyBuffer, recCPU, recMem/(1024*1024))
	}
}

func showSimulation() {
	fmt.Println("\n=== Simulation: How Lookback Periods Affect Recommendations ===\n")
	
	// Simulated metrics
	scenarios := []struct {
		period      string
		p95CPU      int64
		p95Memory   int64
		explanation string
	}{
		{
			"3 days (Dev)",
			180,
			100 * 1024 * 1024,
			"Fast response to changes, may miss weekly patterns",
		},
		{
			"7 days (Default)",
			150,
			90 * 1024 * 1024,
			"Captures weekly patterns (Mon-Sun), industry standard",
		},
		{
			"14 days (Production)",
			140,
			85 * 1024 * 1024,
			"More stable, captures edge cases, higher confidence",
		},
		{
			"15 days (Custom)",
			138,
			84 * 1024 * 1024,
			"Similar to 14 days, matches Kubecost default",
		},
	}

	fmt.Println("Sample workload: idle-app")
	fmt.Println("Current: 500m CPU, 512Mi memory\n")

	for _, s := range scenarios {
		fmt.Printf("[%s]\n", s.period)
		fmt.Printf("  P95: %dm CPU, %dMi memory\n", s.p95CPU, s.p95Memory/(1024*1024))
		
		// With 1.5x buffer
		recCPU := int64(float64(s.p95CPU) * 1.5)
		recMem := int64(float64(s.p95Memory) * 1.5)
		fmt.Printf("  Recommended (1.5x): %dm CPU, %dMi memory\n", recCPU, recMem/(1024*1024))
		
		// Savings
		currentCost := (0.5 * 35.0) + (0.5 * 4.3) // Azure pricing
		recCost := (float64(recCPU)/1000.0 * 35.0) + (float64(recMem)/(1024.0*1024.0*1024.0) * 4.3)
		savings := currentCost - recCost
		
		fmt.Printf("  Savings: $%.2f/mo (Azure)\n", savings)
		fmt.Printf("  Note: %s\n\n", s.explanation)
	}

	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println("\nKey Insight:")
	fmt.Println("  Longer lookback = More stable recommendations")
	fmt.Println("  Shorter lookback = Faster response to changes")
	fmt.Println("\nEasy to change:")
	fmt.Println("  export METRICS_LOOKBACK_DAYS=15  # Change from 7 to 15")
	fmt.Println("  cost-scan -n production --lookback=15d")
}
