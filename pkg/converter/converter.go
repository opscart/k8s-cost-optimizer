package converter

import (
	"fmt"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
)

// OldToNew converts old Recommendation to new models.Recommendation
func OldToNew(old *recommender.Recommendation, clusterID string) *models.Recommendation {
	// Convert type
	var recType models.RecommendationType
	switch old.Type {
	case recommender.RightSize:
		recType = models.RecommendationRightSize
	case recommender.ScaleDown:
		recType = models.RecommendationScaleDown
	case recommender.NoAction:
		recType = models.RecommendationNoAction
	default:
		recType = models.RecommendationNoAction
	}

	// Convert risk
	var risk models.RiskLevel
	switch old.Risk {
	case "LOW":
		risk = models.RiskLow
	case "MEDIUM":
		risk = models.RiskMedium
	case "HIGH":
		risk = models.RiskHigh
	default:
		risk = models.RiskNone
	}

	// Create workload
	workload := &models.Workload{
		ClusterID:  clusterID,
		Namespace:  old.Namespace,
		Deployment: old.DeploymentName,
		Pod:        old.DeploymentName, // Use deployment name as pod identifier for now
	}

	return &models.Recommendation{
		Type:              recType,
		Workload:          workload,
		CurrentCPU:        old.CurrentCPU,
		CurrentMemory:     old.CurrentMemory,
		RecommendedCPU:    old.RecommendedCPU,
		RecommendedMemory: old.RecommendedMemory,
		Reason:            old.Reason,
		SavingsMonthly:    old.Savings,
		Impact:            old.Impact,
		Risk:              risk,
		Command:           generateCommand(old),
	}
}

// generateCommand creates kubectl command from recommendation
func generateCommand(rec *recommender.Recommendation) string {
	if rec.Type == recommender.NoAction {
		return ""
	}

	cpuStr := fmt.Sprintf("%dm", rec.RecommendedCPU)
	memStr := fmt.Sprintf("%dMi", rec.RecommendedMemory/(1024*1024))

	return fmt.Sprintf(
		"kubectl set resources deployment %s -n %s --requests=cpu=%s,memory=%s",
		rec.DeploymentName,
		rec.Namespace,
		cpuStr,
		memStr,
	)
}

// PodAnalysisToWorkload converts analyzer.PodAnalysis to models.Workload
func PodAnalysisToWorkload(pa analyzer.PodAnalysis, clusterID string) *models.Workload {
	return &models.Workload{
		ClusterID: clusterID,
		Namespace: pa.Namespace,
		Pod:       pa.Name,
		Container: pa.ContainerName,
	}
}
