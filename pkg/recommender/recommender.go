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
	Savings           float64
	Impact            string
	Risk              string
}

type Recommender struct {
	cpuCostPerCore    float64
	memoryCostPerGiB  float64
	safetyBuffer      float64
}

func New() *Recommender {
	return &Recommender{
		cpuCostPerCore:   23.0,
		memoryCostPerGiB: 3.0,
		safetyBuffer:     2.0,
	}
}

func (r *Recommender) Analyze(analyses []analyzer.PodAnalysis, deploymentName string) *Recommendation {
	if len(analyses) == 0 {
		return nil
	}

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

	// IMPROVED LOGIC: Only suggest scale down if BOTH CPU and memory are extremely low
	// AND actual usage is near zero (not just low utilization)
	if avgActualCPU < 1 && avgActualMem < 5*1024*1024 { // <1m CPU, <5Mi memory
		rec.Type = ScaleDown
		rec.Reason = "Extremely low resource usage - workload appears idle"
		rec.RecommendedCPU = 0
		rec.RecommendedMemory = 0
		rec.Impact = "HIGH"
		rec.Risk = "MEDIUM"
		
		rec.Savings = r.calculateMonthlyCost(avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		
		return rec
	}

	// Calculate recommended resources with safety buffer
	recCPU := int64(float64(avgActualCPU) * r.safetyBuffer)
	recMem := int64(float64(avgActualMem) * r.safetyBuffer)

	// Set minimum values
	minCPU := int64(25) // 25m minimum
	minMem := int64(50 * 1024 * 1024) // 50Mi minimum
	
	if recCPU < minCPU {
		recCPU = minCPU
	}
	if recMem < minMem {
		recMem = minMem
	}

	// Check if right-sizing would save significant resources
	cpuReduction := float64(avgRequestedCPU-recCPU) / float64(avgRequestedCPU) * 100
	memReduction := float64(avgRequestedMem-recMem) / float64(avgRequestedMem) * 100

	// Only recommend if reduction is >20% AND savings are meaningful
	if (cpuReduction > 20 || memReduction > 20) {
		rec.Type = RightSize
		rec.RecommendedCPU = recCPU
		rec.RecommendedMemory = recMem
		rec.Reason = fmt.Sprintf("Over-provisioned: %.0f%% CPU, %.0f%% memory reduction possible", 
			cpuReduction, memReduction)
		
		currentCost := r.calculateMonthlyCost(avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		newCost := r.calculateMonthlyCost(recCPU, recMem) * float64(len(analyses))
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
		
		rec.Risk = "LOW"
		
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
			"  Recommendation: Scale to 0 replicas\n"+
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
		"  Recommended: %dm CPU, %dMi memory (with 2x safety buffer)\n"+
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
