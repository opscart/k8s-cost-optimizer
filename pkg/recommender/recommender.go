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
	WorkloadType      string
	Environment       string
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

	// Get workload type and environment EARLY (needed by all paths)
	workloadType := analyses[0].WorkloadType
	if workloadType == "" {
		workloadType = "Deployment"
	}

	environment := string(analyses[0].Environment)
	if environment == "" {
		environment = string(analyzer.EnvironmentUnknown)
	}

	workloadConfig := analyzer.GetWorkloadConfig(analyzer.WorkloadType(workloadType))

	// Use COMBINED safety buffer (workload × environment)
	safetyBuffer := analyzer.GetCombinedSafetyBuffer(
		analyzer.WorkloadType(workloadType),
		analyzer.Environment(environment),
	)

	// Check if workload has HPA enabled - skip optimization
	if analyses[0].HasHPA {
		rec := &Recommendation{
			Type:              NoAction,
			DeploymentName:    deploymentName,
			Namespace:         analyses[0].Namespace,
			WorkloadType:      workloadType,
			Environment:       environment,
			Provider:          r.pricingProvider.Name(),
			CurrentCPU:        0,
			CurrentMemory:     0,
			RecommendedCPU:    0,
			RecommendedMemory: 0,
			Reason:            fmt.Sprintf("⚠️  Workload managed by HPA '%s' - Auto-scaling enabled, manual optimization not recommended", analyses[0].HPAName),
			Savings:           0,
			Impact:            "N/A",
			Risk:              "N/A",
		}
		return rec
	}

	rec := &Recommendation{
		DeploymentName: deploymentName,
		Namespace:      analyses[0].Namespace,
		WorkloadType:   workloadType,
		Environment:    environment,
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

	// Check if workload type should be optimized
	if !workloadConfig.OptimizeEnabled {
		rec.Type = NoAction
		rec.Environment = environment
		rec.Reason = fmt.Sprintf("Workload type %s (%s) - optimization disabled for safety",
			workloadType, workloadConfig.Description)
		rec.RecommendedCPU = avgRequestedCPU
		rec.RecommendedMemory = avgRequestedMem
		rec.Risk = workloadConfig.RiskLevel
		rec.Impact = "N/A"
		rec.Savings = 0
		return rec
	}

	// Check if workload is idle (< 5% CPU usage)
	cpuUtil := float64(avgActualCPU) / float64(avgRequestedCPU)
	if cpuUtil < 0.05 {
		rec.Type = ScaleDown
		rec.Environment = environment
		rec.Reason = fmt.Sprintf("Workload appears idle (%.1f%% CPU utilization) - Workload: %s, Environment: %s",
			cpuUtil*100, workloadType, environment)
		rec.RecommendedCPU = 0
		rec.RecommendedMemory = 0
		rec.Impact = "HIGH"
		rec.Risk = workloadConfig.RiskLevel

		rec.Savings = r.calculateMonthlyCost(ctx, avgRequestedCPU, avgRequestedMem) * float64(len(analyses))

		return rec
	}

	// Calculate recommended resources with combined safety buffer
	recCPU := int64(float64(avgActualCPU) * safetyBuffer)
	recMem := int64(float64(avgActualMem) * safetyBuffer)

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
		rec.Environment = environment

		// Include environment context in reason
		rec.Reason = fmt.Sprintf("Over-provisioned: CPU %.0f%% under-utilized, Memory %.0f%% under-utilized (Workload: %s, Safety: %.1fx, Env: %s)",
			cpuReduction, memReduction, workloadType, safetyBuffer, environment)

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

		// Risk assessment
		avgReduction := (cpuReduction + memReduction) / 2
		if avgReduction > 75 {
			rec.Risk = "HIGH"
		} else if avgReduction > 50 {
			rec.Risk = "MEDIUM"
		} else {
			rec.Risk = workloadConfig.RiskLevel
		}

		return rec
	}

	// No action needed
	rec.Type = NoAction
	rec.Environment = environment
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
