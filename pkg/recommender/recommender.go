package recommender

import (
	"fmt"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
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
	Savings           float64 // Monthly savings in USD
	Impact            string  // HIGH, MEDIUM, LOW
	Risk              string  // LOW, MEDIUM, HIGH
}

type Recommender struct {
	// Default pricing per month
	cpuCostPerCore    float64 // $ per core per month
	memoryCostPerGiB  float64 // $ per GiB per month
	safetyBuffer      float64 // multiplier for safety (e.g., 2.0 = 2x usage)
}

func New() *Recommender {
	return &Recommender{
		cpuCostPerCore:   23.0,  // Default: ~$23/month per core
		memoryCostPerGiB: 3.0,   // Default: ~$3/month per GiB
		safetyBuffer:     2.0,   // 2x actual usage for safety
	}
}

func (r *Recommender) Analyze(analyses []analyzer.PodAnalysis, deploymentName string) *Recommendation {
	if len(analyses) == 0 {
		return nil
	}

	// Calculate average usage across all pods in deployment
	var totalRequestedCPU, totalActualCPU int64
	var totalRequestedMem, totalActualMem int64
	
	for _, analysis := range analyses {
		totalRequestedCPU += analysis.RequestedCPU
		totalActualCPU += analysis.ActualCPU
		totalRequestedMem += analysis.RequestedMemory
		totalActualMem += analysis.ActualMemory
	}

	avgRequestedCPU := totalRequestedCPU / int64(len(analyses))
	avgActualCPU := totalActualCPU / int64(len(analyses))
	avgRequestedMem := totalRequestedMem / int64(len(analyses))
	avgActualMem := totalActualMem / int64(len(analyses))

	rec := &Recommendation{
		DeploymentName: deploymentName,
		Namespace:      analyses[0].Namespace,
		CurrentCPU:     avgRequestedCPU,
		CurrentMemory:  avgRequestedMem,
	}

	// Determine recommendation type
	cpuUtil := float64(avgActualCPU) / float64(avgRequestedCPU) * 100
	memUtil := float64(avgActualMem) / float64(avgRequestedMem) * 100

	// If utilization is extremely low, consider scaling down
	if cpuUtil < 1.0 && memUtil < 5.0 {
		rec.Type = ScaleDown
		rec.Reason = "Extremely low resource utilization - consider scaling to 0 or removing"
		rec.RecommendedCPU = 0
		rec.RecommendedMemory = 0
		rec.Impact = "HIGH"
		rec.Risk = "MEDIUM"
		
		// Calculate full savings (100% of current cost)
		rec.Savings = r.calculateMonthlyCost(avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		
		return rec
	}

	// Calculate recommended resources with safety buffer
	recCPU := int64(float64(avgActualCPU) * r.safetyBuffer)
	recMem := int64(float64(avgActualMem) * r.safetyBuffer)

	// Minimum values (don't recommend below 10m CPU, 10Mi memory)
	if recCPU < 10 {
		recCPU = 10
	}
	if recMem < 10*1024*1024 { // 10Mi in bytes
		recMem = 10 * 1024 * 1024
	}

	// Check if right-sizing would save significant resources (>20% reduction)
	cpuReduction := float64(avgRequestedCPU-recCPU) / float64(avgRequestedCPU) * 100
	memReduction := float64(avgRequestedMem-recMem) / float64(avgRequestedMem) * 100

	if cpuReduction > 20 || memReduction > 20 {
		rec.Type = RightSize
		rec.RecommendedCPU = recCPU
		rec.RecommendedMemory = recMem
		rec.Reason = fmt.Sprintf("Over-provisioned: %.0f%% CPU reduction, %.0f%% memory reduction possible", 
			cpuReduction, memReduction)
		
		// Calculate savings
		currentCost := r.calculateMonthlyCost(avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		newCost := r.calculateMonthlyCost(recCPU, recMem) * float64(len(analyses))
		rec.Savings = currentCost - newCost
		
		// Determine impact and risk
		if rec.Savings > 100 {
			rec.Impact = "HIGH"
		} else if rec.Savings > 50 {
			rec.Impact = "MEDIUM"
		} else {
			rec.Impact = "LOW"
		}
		
		rec.Risk = "LOW" // With 2x safety buffer, risk is low
		
		return rec
	}

	// No significant savings opportunity
	rec.Type = NoAction
	rec.Reason = "Resource allocation is appropriate"
	rec.RecommendedCPU = avgRequestedCPU
	rec.RecommendedMemory = avgRequestedMem
	rec.Savings = 0
	rec.Impact = "NONE"
	rec.Risk = "NONE"

	return rec
}

func (r *Recommender) calculateMonthlyCost(cpuMillicores int64, memoryBytes int64) float64 {
	cpuCores := float64(cpuMillicores) / 1000.0
	memoryGiB := float64(memoryBytes) / (1024.0 * 1024.0 * 1024.0)
	
	return (cpuCores * r.cpuCostPerCore) + (memoryGiB * r.memoryCostPerGiB)
}

func (r *Recommendation) String() string {
	if r.Type == NoAction {
		return fmt.Sprintf("[%s] %s: %s", r.Impact, r.DeploymentName, r.Reason)
	}

	if r.Type == ScaleDown {
		return fmt.Sprintf(
			"[%s] %s: %s\n"+
			"  Current: %dm CPU, %dMi memory\n"+
			"  Recommendation: Scale to 0 replicas or remove\n"+
			"  Savings: $%.2f/month\n"+
			"  Risk: %s",
			r.Impact,
			r.DeploymentName,
			r.Reason,
			r.CurrentCPU,
			r.CurrentMemory/(1024*1024),
			r.Savings,
			r.Risk,
		)
	}

	return fmt.Sprintf(
		"[%s] %s: %s\n"+
		"  Current: %dm CPU, %dMi memory\n"+
		"  Recommended: %dm CPU, %dMi memory\n"+
		"  Savings: $%.2f/month\n"+
		"  Risk: %s",
		r.Impact,
		r.DeploymentName,
		r.Reason,
		r.CurrentCPU,
		r.CurrentMemory/(1024*1024),
		r.RecommendedCPU,
		r.RecommendedMemory/(1024*1024),
		r.Savings,
		r.Risk,
	)
}
