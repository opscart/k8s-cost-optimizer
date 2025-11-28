package analyzer

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvironmentProduction  Environment = "production"
	EnvironmentStaging     Environment = "staging"
	EnvironmentDevelopment Environment = "development"
	EnvironmentUnknown     Environment = "unknown"
)

// EnvironmentConfig holds configuration for different environments
type EnvironmentConfig struct {
	SafetyBufferMultiplier float64 // Additional multiplier on top of workload buffer
	MinDataDays            int     // Minimum days of metrics required
	RiskTolerance          string  // HIGH, MEDIUM, LOW
	Description            string
	IsProduction           bool
}

// GetEnvironmentConfig returns configuration for a given environment
func GetEnvironmentConfig(env Environment) EnvironmentConfig {
	configs := map[Environment]EnvironmentConfig{
		EnvironmentProduction: {
			SafetyBufferMultiplier: 1.3,  // 30% extra on top of workload buffer
			MinDataDays:            7,
			RiskTolerance:          "LOW",
			Description:            "Production environment - conservative optimization",
			IsProduction:           true,
		},
		EnvironmentStaging: {
			SafetyBufferMultiplier: 1.0,  // Standard workload buffer
			MinDataDays:            5,
			RiskTolerance:          "MEDIUM",
			Description:            "Staging environment - balanced optimization",
			IsProduction:           false,
		},
		EnvironmentDevelopment: {
			SafetyBufferMultiplier: 0.85, // 15% less than workload buffer (more aggressive)
			MinDataDays:            3,
			RiskTolerance:          "HIGH",
			Description:            "Development environment - aggressive optimization",
			IsProduction:           false,
		},
		EnvironmentUnknown: {
			SafetyBufferMultiplier: 1.2,  // Conservative for unknown
			MinDataDays:            7,
			RiskTolerance:          "MEDIUM",
			Description:            "Unknown environment - cautious optimization",
			IsProduction:           false,
		},
	}

	if config, exists := configs[env]; exists {
		return config
	}

	return configs[EnvironmentUnknown]
}

// ClassifyNamespace determines the environment type of a namespace
func ClassifyNamespace(ctx context.Context, clientset *kubernetes.Clientset, namespace string) Environment {
	// Try to get namespace object to check labels
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil && ns.Labels != nil {
		// Check for environment label
		if env, exists := ns.Labels["environment"]; exists {
			return normalizeEnvironment(env)
		}

		// Check for tier label
		if tier, exists := ns.Labels["tier"]; exists {
			if tier == "prod" || tier == "production" {
				return EnvironmentProduction
			}
			if tier == "staging" || tier == "stage" {
				return EnvironmentStaging
			}
			if tier == "dev" || tier == "development" {
				return EnvironmentDevelopment
			}
		}
	}

	// Fallback to name-based detection
	return detectEnvironmentFromName(namespace)
}

// normalizeEnvironment converts label value to Environment type
func normalizeEnvironment(label string) Environment {
	label = strings.ToLower(strings.TrimSpace(label))

	switch label {
	case "production", "prod", "prd":
		return EnvironmentProduction
	case "staging", "stage", "stg":
		return EnvironmentStaging
	case "development", "dev", "test", "testing":
		return EnvironmentDevelopment
	default:
		return EnvironmentUnknown
	}
}

// detectEnvironmentFromName tries to detect environment from namespace name
func detectEnvironmentFromName(namespace string) Environment {
	name := strings.ToLower(namespace)

	// Production patterns
	prodPatterns := []string{"prod", "production", "prd"}
	for _, pattern := range prodPatterns {
		if strings.Contains(name, pattern) {
			return EnvironmentProduction
		}
	}

	// Staging patterns
	stagingPatterns := []string{"staging", "stage", "stg", "uat"}
	for _, pattern := range stagingPatterns {
		if strings.Contains(name, pattern) {
			return EnvironmentStaging
		}
	}

	// Development patterns
	devPatterns := []string{"dev", "develop", "test", "sandbox", "demo"}
	for _, pattern := range devPatterns {
		if strings.Contains(name, pattern) {
			return EnvironmentDevelopment
		}
	}

	// Default to unknown for ambiguous namespaces
	return EnvironmentUnknown
}

// GetCombinedSafetyBuffer combines workload and environment safety buffers
func GetCombinedSafetyBuffer(workloadType WorkloadType, environment Environment) float64 {
	workloadConfig := GetWorkloadConfig(workloadType)
	envConfig := GetEnvironmentConfig(environment)

	// Multiply workload buffer by environment multiplier
	combinedBuffer := workloadConfig.SafetyBuffer * envConfig.SafetyBufferMultiplier

	// Ensure minimum buffer of 1.2x
	if combinedBuffer < 1.2 {
		combinedBuffer = 1.2
	}

	return combinedBuffer
}
