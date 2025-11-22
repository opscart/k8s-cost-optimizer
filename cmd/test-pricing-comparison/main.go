package main

import (
	"context"
	"fmt"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
)

func main() {
	fmt.Println("=== Pricing Impact Comparison ===\n")

	// Sample workload data
	sampleAnalysis := []analyzer.PodAnalysis{
		{
			Name:              "api-server-xyz",
			Namespace:         "production",
			RequestedCPU:      1000, // 1 core
			ActualCPU:         250,  // 250m actual
			RequestedMemory:   2147483648, // 2 GiB
			ActualMemory:      536870912,  // 512 MiB actual
			CPUUtilization:    25.0,
			MemoryUtilization: 25.0,
		},
	}

	fmt.Println("Sample Workload:")
	fmt.Println("  Requested: 1000m CPU, 2048Mi memory")
	fmt.Println("  Actual: 250m CPU, 512Mi memory")
	fmt.Println("  Utilization: 25% CPU, 25% memory\n")

	providers := []struct {
		name     string
		provider pricing.Provider
	}{
		{"Default", pricing.NewDefaultProvider(23.0, 3.0)},
		{"Azure", pricing.NewAzureProvider("eastus")},
		{"AWS", pricing.NewAWSProvider("us-east-1")},
		{"GCP", pricing.NewGCPProvider("us-central1")},
	}

	fmt.Printf("%-10s %-15s %-20s %-15s\n", "Provider", "Current Cost", "Recommended Cost", "Savings")
	fmt.Println(string(make([]byte, 70)))

	for _, p := range providers {
		rec := recommender.NewWithPricing(p.provider)
		recommendation := rec.Analyze(sampleAnalysis, "api-server")

		if recommendation != nil {
			ctx := context.Background()
			costInfo, _ := p.provider.GetCostInfo(ctx, "", "")
			
			// Calculate current cost
			currentCPU := 1.0 // 1 core
			currentMem := 2.0 // 2 GiB
			currentCost := (currentCPU * costInfo.CPUCostPerCore) + (currentMem * costInfo.MemoryCostPerGiB)
			
			// Calculate recommended cost (375m CPU, 768Mi memory with 1.5x buffer)
			recCPU := float64(recommendation.RecommendedCPU) / 1000.0
			recMem := float64(recommendation.RecommendedMemory) / (1024.0 * 1024.0 * 1024.0)
			recCost := (recCPU * costInfo.CPUCostPerCore) + (recMem * costInfo.MemoryCostPerGiB)
			
			savings := currentCost - recCost
			
			fmt.Printf("%-10s $%-14.2f $%-19.2f $%-14.2f\n",
				p.name,
				currentCost,
				recCost,
				savings)
		}
	}

	fmt.Println("\nKey Insights:")
	fmt.Println("  1. Cloud provider pricing significantly impacts calculated savings")
	fmt.Println("  2. Azure shows highest savings due to higher base costs")
	fmt.Println("  3. Same optimization yields different ROI per cloud")
	fmt.Println("  4. Accurate pricing is critical for prioritization")
}
