package recommender

import (
	"context"
	"fmt"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
)

type RecommendationType string

const (
	RightSize RecommendationType = "RIGHT_SIZE"
	ScaleDown RecommendationType = "SCALE_DOWN"
	NoAction  RecommendationType = "NO_ACTION"
)

type Recommendation struct {
	Type              RecommendationType
	DeploymentName    string
	Namespace         string
	CurrentCPU        int64
	CurrentMemory     int64
	RecommendedCPU    int64
	RecommendedMemory int64
	Reason            string
	Savings           float64
	Impact            string
	Risk              string
	Provider          string // Cloud provider used for pricing
}

type Recommender struct {
	pricingProvider pricing.Provider
	safetyBuffer    float64
}

// New creates a recommender with default pricing (backwards compatible)
func New() *Recommender {
	return &Recommender{
		pricingProvider: pricing.NewDefaultProvider(23.0, 3.0),
		safetyBuffer:    1.5,
	}
}

// NewWithPricing creates a recommender with a specific pricing provider
func NewWithPricing(provider pricing.Provider) *Recommender {
	return &Recommender{
		pricingProvider: provider,
		safetyBuffer:    1.5,
	}
}

func (r *Recommender) Analyze(analyses []analyzer.PodAnalysis, deploymentName string) *Recommendation {
	ctx := context.Background()

	if len(analyses) == 0 {
		return nil
	}

	rec := &Recommendation{
		DeploymentName: deploymentName,
		Namespace:      analyses[0].Namespace,
		Provider:       r.pricingProvider.Name(),
	}

	// Calculate average requested and actual usage
	var totalRequestedCPU, totalActualCPU int64
	var totalRequestedMem, totalActualMem int64

	for _, analysis := range analyses {
		totalRequestedCPU += analysis.RequestedCPU
		totalActualCPU += analysis.ActualCPU
		totalRequestedMem += analysis.RequestedMemory
		totalActualMem += analysis.ActualMemory
	}

	avgRequestedCPU := totalRequestedCPU / int64(len(analyses))
	avgRequestedMem := totalRequestedMem / int64(len(analyses))
	avgActualCPU := totalActualCPU / int64(len(analyses))
	avgActualMem := totalActualMem / int64(len(analyses))

	rec.CurrentCPU = avgRequestedCPU
	rec.CurrentMemory = avgRequestedMem

	// Check if workload is idle (< 5% CPU usage)
	cpuUtil := float64(avgActualCPU) / float64(avgRequestedCPU)
	if cpuUtil < 0.05 {
		rec.Type = ScaleDown
		rec.Reason = fmt.Sprintf("Workload appears idle (%.1f%% CPU utilization)", cpuUtil*100)
		rec.RecommendedCPU = 0
		rec.RecommendedMemory = 0
		rec.Impact = "HIGH"
		rec.Risk = "MEDIUM"

		rec.Savings = r.calculateMonthlyCost(ctx, avgRequestedCPU, avgRequestedMem) * float64(len(analyses))

		return rec
	}

	// Calculate recommended resources with safety buffer
	recCPU := int64(float64(avgActualCPU) * r.safetyBuffer)
	recMem := int64(float64(avgActualMem) * r.safetyBuffer)

	// Minimum thresholds
	if recCPU < 10 {
		recCPU = 10
	}
	if recMem < 10*1024*1024 {
		recMem = 10 * 1024 * 1024
	}

	// Check if right-sizing is beneficial
	cpuReduction := (float64(avgRequestedCPU) - float64(recCPU)) / float64(avgRequestedCPU) * 100
	memReduction := (float64(avgRequestedMem) - float64(recMem)) / float64(avgRequestedMem) * 100

	if cpuReduction > 25 || memReduction > 25 {
		rec.Type = RightSize
		rec.RecommendedCPU = recCPU
		rec.RecommendedMemory = recMem
		rec.Reason = fmt.Sprintf("Over-provisioned: %.0f%% CPU, %.0f%% memory reduction possible",
			cpuReduction, memReduction)

		currentCost := r.calculateMonthlyCost(ctx, avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		newCost := r.calculateMonthlyCost(ctx, recCPU, recMem) * float64(len(analyses))
		rec.Savings = currentCost - newCost

		// Skip if savings are negligible (<$1/month)
		if rec.Savings < 1.0 {
			rec.Type = NoAction
			rec.Reason = "Savings too small to justify change"
			rec.Impact = "NONE"
			rec.Risk = "NONE"
			return rec
		}

		if rec.Savings > 50 {
			rec.Impact = "HIGH"
		} else if rec.Savings > 20 {
			rec.Impact = "MEDIUM"
		} else {
			rec.Impact = "LOW"
		}

		// Risk assessment based on reduction percentage
		avgReduction := (cpuReduction + memReduction) / 2
		if avgReduction > 75 {
			rec.Risk = "HIGH"
		} else if avgReduction > 50 {
			rec.Risk = "MEDIUM"
		} else {
			rec.Risk = "LOW"
		}

		return rec
	}

	// No action needed
	rec.Type = NoAction
	rec.Reason = "Resource allocation is appropriate"
	rec.RecommendedCPU = avgRequestedCPU
	rec.RecommendedMemory = avgRequestedMem
	rec.Savings = 0
	rec.Impact = "NONE"
	rec.Risk = "NONE"

	return rec
}

func (r *Recommender) calculateMonthlyCost(ctx context.Context, cpuMillicores int64, memoryBytes int64) float64 {
	// Get cost info from provider
	costInfo, err := r.pricingProvider.GetCostInfo(ctx, "", "")
	if err != nil {
		// Fallback to defaults if provider fails
		cpuCores := float64(cpuMillicores) / 1000.0
		memoryGiB := float64(memoryBytes) / (1024.0 * 1024.0 * 1024.0)
		return (cpuCores * 23.0) + (memoryGiB * 3.0)
	}

	cpuCores := float64(cpuMillicores) / 1000.0
	memoryGiB := float64(memoryBytes) / (1024.0 * 1024.0 * 1024.0)

	return (cpuCores * costInfo.CPUCostPerCore) + (memoryGiB * costInfo.MemoryCostPerGiB)
}

func (r *Recommendation) String() string {
	if r.Type == NoAction {
		return fmt.Sprintf("[%s] %s: %s", r.Impact, r.DeploymentName, r.Reason)
	}

	if r.Type == ScaleDown {
		return fmt.Sprintf(
			"[%s] %s: %s\n"+
				"  Current: %dm CPU, %dMi memory\n"+
				"  Recommendation: Scale to 0 replicas\n"+
				"  Savings: $%.2f/month (%s pricing)\n"+
				"  Risk: %s",
			r.Impact,
			r.DeploymentName,
			r.Reason,
			r.CurrentCPU,
			r.CurrentMemory/(1024*1024),
			r.Savings,
			r.Provider,
			r.Risk,
		)
	}

	return fmt.Sprintf(
		"[%s] %s: %s\n"+
			"  Current: %dm CPU, %dMi memory\n"+
			"  Recommended: %dm CPU, %dMi memory (with 1.5x safety buffer)\n"+
			"  Savings: $%.2f/month (%s pricing)\n"+
			"  Risk: %s",
		r.Impact,
		r.DeploymentName,
		r.Reason,
		r.CurrentCPU,
		r.CurrentMemory/(1024*1024),
		r.RecommendedCPU,
		r.RecommendedMemory/(1024*1024),
		r.Savings,
		r.Provider,
		r.Risk,
	)
}
