package analyzer

// WorkloadType represents different Kubernetes workload types
type WorkloadType string

const (
	WorkloadDeployment  WorkloadType = "Deployment"
	WorkloadStatefulSet WorkloadType = "StatefulSet"
	WorkloadDaemonSet   WorkloadType = "DaemonSet"
	WorkloadJob         WorkloadType = "Job"
	WorkloadCronJob     WorkloadType = "CronJob"
	WorkloadReplicaSet  WorkloadType = "ReplicaSet"
	WorkloadUnknown     WorkloadType = "Unknown"
)

// WorkloadConfig holds configuration for different workload types
type WorkloadConfig struct {
	SafetyBuffer    float64 // Multiplier for resource recommendations
	MinDataDays     int     // Minimum days of data required
	Description     string  // Human-readable description
	RiskLevel       string  // LOW, MEDIUM, HIGH
	OptimizeEnabled bool    // Whether to optimize this type
}

// GetWorkloadConfig returns configuration for a given workload type
func GetWorkloadConfig(workloadType WorkloadType) WorkloadConfig {
	configs := map[WorkloadType]WorkloadConfig{
		WorkloadDeployment: {
			SafetyBuffer:    1.5,
			MinDataDays:     5,
			Description:     "Stateless application",
			RiskLevel:       "LOW",
			OptimizeEnabled: true,
		},
		WorkloadStatefulSet: {
			SafetyBuffer:    2.0,
			MinDataDays:     7,
			Description:     "Stateful application (databases, queues)",
			RiskLevel:       "MEDIUM",
			OptimizeEnabled: true,
		},
		WorkloadDaemonSet: {
			SafetyBuffer:    2.5,
			MinDataDays:     7,
			Description:     "Node-critical service (monitoring, logging)",
			RiskLevel:       "HIGH",
			OptimizeEnabled: false, // Usually don't optimize DaemonSets
		},
		WorkloadJob: {
			SafetyBuffer:    1.2,
			MinDataDays:     3,
			Description:     "Batch job workload",
			RiskLevel:       "LOW",
			OptimizeEnabled: true,
		},
		WorkloadCronJob: {
			SafetyBuffer:    1.2,
			MinDataDays:     3,
			Description:     "Scheduled batch workload",
			RiskLevel:       "LOW",
			OptimizeEnabled: true,
		},
		WorkloadReplicaSet: {
			SafetyBuffer:    1.5,
			MinDataDays:     5,
			Description:     "Standalone ReplicaSet",
			RiskLevel:       "MEDIUM",
			OptimizeEnabled: true,
		},
	}

	if config, exists := configs[workloadType]; exists {
		return config
	}

	// Default for unknown workloads - conservative
	return WorkloadConfig{
		SafetyBuffer:    2.0,
		MinDataDays:     7,
		Description:     "Unknown workload type",
		RiskLevel:       "HIGH",
		OptimizeEnabled: false,
	}
}

// GetSafetyBuffer returns the safety buffer for a workload type
func GetSafetyBuffer(workloadType string) float64 {
	return GetWorkloadConfig(WorkloadType(workloadType)).SafetyBuffer
}
