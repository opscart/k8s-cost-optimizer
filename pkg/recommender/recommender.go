package recommender

import (
	"context"
	"fmt"
	"strings"

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
	Provider          string
	Confidence        string
	DataQuality       float64
	PatternInfo       string
	HasSufficientData bool
}

type Recommender struct {
	pricingProvider pricing.Provider
	safetyBuffer    float64
}

func New() *Recommender {
	return &Recommender{
		pricingProvider: pricing.NewDefaultProvider(23.0, 3.0),
		safetyBuffer:    1.5,
	}
}

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

	// Get workload type and environment
	workloadType := analyses[0].WorkloadType
	if workloadType == "" {
		workloadType = "Deployment"
	}

	environment := string(analyses[0].Environment)
	if environment == "" {
		environment = string(analyzer.EnvironmentUnknown)
	}

	workloadConfig := analyzer.GetWorkloadConfig(analyzer.WorkloadType(workloadType))

	// Calculate combined safety buffer
	baseSafetyBuffer := analyzer.GetCombinedSafetyBuffer(
		analyzer.WorkloadType(workloadType),
		analyzer.Environment(environment),
	)

	// Adjust safety buffer based on pattern analysis (Week 9)
	safetyBuffer := baseSafetyBuffer
	if analyses[0].HasSufficientData {
		safetyBuffer = adjustSafetyBufferForPattern(
			baseSafetyBuffer,
			analyses[0].CPUPattern,
			analyses[0].MemoryPattern,
		)
	}

	// Calculate confidence and pattern info (Week 9 Day 2)
	confidence := calculateConfidence(
		analyses[0].DataQuality,
		analyses[0].HasSufficientData,
		analyses[0].CPUPattern.Type,
	)
	patternInfo := buildPatternInfo(
		analyses[0].CPUPattern,
		analyses[0].MemoryPattern,
		analyses[0].CPUGrowth,
	)

	// Check if workload has HPA - skip optimization
	if analyses[0].HasHPA {
		return &Recommendation{
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
			Confidence:        "N/A",
			DataQuality:       0,
			PatternInfo:       "HPA-managed",
			HasSufficientData: false,
		}
	}

	// Calculate averages
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

	rec := &Recommendation{
		DeploymentName: deploymentName,
		Namespace:      analyses[0].Namespace,
		WorkloadType:   workloadType,
		Environment:    environment,
		Provider:       r.pricingProvider.Name(),
		CurrentCPU:     avgRequestedCPU,
		CurrentMemory:  avgRequestedMem,
	}

	// Check if workload type should be optimized
	if !workloadConfig.OptimizeEnabled {
		rec.Type = NoAction
		rec.Reason = fmt.Sprintf("Workload type %s (%s) - optimization disabled for safety",
			workloadType, workloadConfig.Description)
		rec.RecommendedCPU = avgRequestedCPU
		rec.RecommendedMemory = avgRequestedMem
		rec.Risk = workloadConfig.RiskLevel
		rec.Impact = "N/A"
		rec.Savings = 0
		rec.Confidence = confidence
		rec.DataQuality = analyses[0].DataQuality
		rec.PatternInfo = patternInfo
		rec.HasSufficientData = analyses[0].HasSufficientData
		return rec
	}

	// Check if workload is idle
	cpuUtil := float64(avgActualCPU) / float64(avgRequestedCPU)
	if cpuUtil < 0.05 {
		rec.Type = ScaleDown

		// Build reason with pattern context
		reasonParts := []string{
			fmt.Sprintf("Workload appears idle (%.1f%% CPU utilization)", cpuUtil*100),
		}

		// Add data quality warning if confidence is low
		if !analyses[0].HasSufficientData {
			reasonParts = append(reasonParts, "⚠️ Limited historical data (<3 days)")
		} else if analyses[0].CPUPattern.Type != "" {
			reasonParts = append(reasonParts, fmt.Sprintf("Pattern: %s", analyses[0].CPUPattern.Type))
		}

		reasonParts = append(reasonParts, fmt.Sprintf("Workload: %s, Environment: %s", workloadType, environment))

		rec.Reason = strings.Join(reasonParts, " - ")
		rec.RecommendedCPU = 0
		rec.RecommendedMemory = 0
		rec.Impact = "HIGH"
		rec.Risk = workloadConfig.RiskLevel
		rec.Savings = r.calculateMonthlyCost(ctx, avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		rec.Confidence = confidence
		rec.DataQuality = analyses[0].DataQuality
		rec.PatternInfo = patternInfo
		rec.HasSufficientData = analyses[0].HasSufficientData
		return rec
	}

	// Calculate recommended resources
	recCPU := int64(float64(avgActualCPU) * safetyBuffer)
	recMem := int64(float64(avgActualMem) * safetyBuffer)

	// Adjust for growth trends (Week 9)
	if analyses[0].HasSufficientData {
		recCPU = adjustForGrowthTrend(recCPU, analyses[0].CPUGrowth)
		recMem = adjustForGrowthTrend(recMem, analyses[0].MemoryGrowth)
	}

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

		// Build reason with pattern and growth context (Week 9)
		reasonParts := []string{
			fmt.Sprintf("Over-provisioned: CPU %.0f%% under-utilized, Memory %.0f%% under-utilized", cpuReduction, memReduction),
		}

		if analyses[0].HasSufficientData {
			reasonParts = append(reasonParts,
				fmt.Sprintf("Pattern: CPU %s (CV: %.2f)", analyses[0].CPUPattern.Type, analyses[0].CPUPattern.Variation))

			if analyses[0].CPUGrowth.IsGrowing && analyses[0].CPUGrowth.RatePerMonth > 5.0 {
				reasonParts = append(reasonParts,
					fmt.Sprintf("⚠️ Growing %.1f%%/month", analyses[0].CPUGrowth.RatePerMonth))
			}
		}

		reasonParts = append(reasonParts,
			fmt.Sprintf("Workload: %s, Safety: %.1fx, Env: %s", workloadType, safetyBuffer, environment))

		rec.Reason = strings.Join(reasonParts, " | ")

		currentCost := r.calculateMonthlyCost(ctx, avgRequestedCPU, avgRequestedMem) * float64(len(analyses))
		newCost := r.calculateMonthlyCost(ctx, recCPU, recMem) * float64(len(analyses))
		rec.Savings = currentCost - newCost

		// Skip if savings negligible
		if rec.Savings < 1.0 {
			rec.Type = NoAction
			rec.Reason = fmt.Sprintf("Savings too small to justify change ($%.2f/month) - Change overhead not worth minimal benefit", rec.Savings)
			rec.Impact = "NONE"
			rec.Risk = "NONE"
			rec.Confidence = confidence
			rec.DataQuality = analyses[0].DataQuality
			rec.PatternInfo = patternInfo
			rec.HasSufficientData = analyses[0].HasSufficientData
			return rec
		}

		// Set impact
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

		rec.Confidence = confidence
		rec.DataQuality = analyses[0].DataQuality
		rec.PatternInfo = patternInfo
		rec.HasSufficientData = analyses[0].HasSufficientData

		return rec
	}

	// No action needed
	rec.Type = NoAction
	reasonParts := []string{"Resource allocation is appropriate"}
	cpuUtil = float64(avgActualCPU) / float64(avgRequestedCPU) * 100
	memUtil := float64(avgActualMem) / float64(avgRequestedMem) * 100
	reasonParts = append(reasonParts, fmt.Sprintf("CPU utilization: %.0f%%, Memory utilization: %.0f%%", cpuUtil, memUtil))

	// Add pattern info if high confidence
	if confidence == "HIGH" && analyses[0].CPUPattern.Type != "" {
		reasonParts = append(reasonParts, fmt.Sprintf("Pattern: %s (consistent)", analyses[0].CPUPattern.Type))
	}

	rec.Reason = strings.Join(reasonParts, " - ")
	rec.RecommendedCPU = avgRequestedCPU
	rec.RecommendedMemory = avgRequestedMem
	rec.Savings = 0
	rec.Impact = "NONE"
	rec.Risk = "NONE"
	rec.Confidence = confidence
	rec.DataQuality = analyses[0].DataQuality
	rec.PatternInfo = patternInfo
	rec.HasSufficientData = analyses[0].HasSufficientData

	return rec
}

func (r *Recommender) calculateMonthlyCost(ctx context.Context, cpuMillicores int64, memoryBytes int64) float64 {
	costInfo, err := r.pricingProvider.GetCostInfo(ctx, "", "")
	if err != nil {
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

// Week 9: Pattern-based safety buffer adjustment
func adjustSafetyBufferForPattern(baseSafetyBuffer float64, cpuPattern analyzer.UsagePattern, memPattern analyzer.UsagePattern) float64 {
	adjustedBuffer := baseSafetyBuffer

	switch cpuPattern.Type {
	case "steady":
		adjustedBuffer *= 0.90 // -10%
	case "spiky":
		adjustedBuffer *= 1.15 // +15%
	case "highly-variable":
		adjustedBuffer *= 1.25 // +25%
	case "moderate":
		adjustedBuffer *= 1.0
	}

	if cpuPattern.Variation > 0.5 {
		adjustedBuffer *= 1.10
	}

	if adjustedBuffer < 1.2 {
		adjustedBuffer = 1.2
	}
	if adjustedBuffer > 3.0 {
		adjustedBuffer = 3.0
	}

	return adjustedBuffer
}

// Week 9: Growth-aware recommendations
func adjustForGrowthTrend(baseValue int64, growth analyzer.GrowthTrend) int64 {
	if !growth.IsGrowing {
		return baseValue
	}

	if growth.RatePerMonth > 5.0 {
		growthBuffer := int64(growth.Predicted3Month * 0.5)
		return baseValue + growthBuffer
	}

	return baseValue
}

// Week 9 Day 2: Confidence scoring
func calculateConfidence(dataQuality float64, hasSufficientData bool, patternType string) string {
	if !hasSufficientData {
		return "LOW"
	}

	if dataQuality >= 0.8 && patternType == "steady" {
		return "HIGH"
	} else if dataQuality >= 0.6 {
		return "MEDIUM"
	}
	return "LOW"
}

// Week 9 Day 2: Pattern info for display
func buildPatternInfo(cpuPattern analyzer.UsagePattern, memPattern analyzer.UsagePattern, cpuGrowth analyzer.GrowthTrend) string {
	parts := []string{}

	if cpuPattern.Type != "" {
		parts = append(parts, fmt.Sprintf("CPU: %s", cpuPattern.Type))
	}

	if cpuGrowth.IsGrowing && cpuGrowth.RatePerMonth > 5.0 {
		parts = append(parts, fmt.Sprintf("Growing %.0f%%/mo", cpuGrowth.RatePerMonth))
	}

	if len(parts) == 0 {
		return "Insufficient data"
	}

	return strings.Join(parts, ", ")
}
