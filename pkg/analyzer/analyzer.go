package analyzer

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodAnalysis struct {
	Name              string
	Namespace         string
	ContainerName     string
	RequestedCPU      int64   // in millicores
	RequestedMemory   int64   // in bytes
	ActualCPU         int64   // in millicores
	ActualMemory      int64   // in bytes
	CPUUtilization    float64 // percentage
	MemoryUtilization float64 // percentage
	HasHPA            bool    // NEW: indicates if workload has HPA
	HPAName           string  // NEW: name of the HPA
	WorkloadType      string  // NEW: Deployment, StatefulSet, etc.
	WorkloadName      string  // NEW: name of the parent workload
	Environment       Environment
}

type Analyzer struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
}

func New(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset) *Analyzer {
	return &Analyzer{
		clientset:     clientset,
		metricsClient: metricsClient,
	}
}

// getTopLevelOwner extracts the top-level workload (Deployment/StatefulSet) from pod
func getTopLevelOwner(pod corev1.Pod) (kind string, name string) {
	if len(pod.OwnerReferences) == 0 {
		return "", ""
	}

	owner := pod.OwnerReferences[0]

	// If owner is ReplicaSet, extract Deployment name
	if owner.Kind == "ReplicaSet" {
		rsName := owner.Name
		lastDash := strings.LastIndex(rsName, "-")
		if lastDash > 0 {
			return "Deployment", rsName[:lastDash]
		}
	}

	return owner.Kind, owner.Name
}

// checkHPA checks if a pod's workload has an HPA configured
func (a *Analyzer) checkHPA(ctx context.Context, pod corev1.Pod) (bool, string) {
	ownerKind, ownerName := getTopLevelOwner(pod)

	if ownerName == "" {
		return false, ""
	}

	hpaList, err := a.clientset.AutoscalingV2().HorizontalPodAutoscalers(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// Log error but don't fail - just assume no HPA
		return false, ""
	}

	for _, hpa := range hpaList.Items {
		if hpa.Spec.ScaleTargetRef.Name == ownerName &&
			hpa.Spec.ScaleTargetRef.Kind == ownerKind {
			return true, hpa.Name
		}
	}
	return false, ""
}

func (a *Analyzer) AnalyzePods(ctx context.Context, namespace string) ([]PodAnalysis, error) {
	// Get pods
	pods, err := a.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	// Classify namespace environment ONCE for all pods
	environment := ClassifyNamespace(ctx, a.clientset, namespace)
	// Get pod metrics
	podMetrics, err := a.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	// Create metrics lookup
	metricsMap := make(map[string]map[string]struct {
		cpu    resource.Quantity
		memory resource.Quantity
	})

	for _, pm := range podMetrics.Items {
		metricsMap[pm.Name] = make(map[string]struct {
			cpu    resource.Quantity
			memory resource.Quantity
		})
		for _, container := range pm.Containers {
			metricsMap[pm.Name][container.Name] = struct {
				cpu    resource.Quantity
				memory resource.Quantity
			}{
				cpu:    container.Usage[corev1.ResourceCPU],
				memory: container.Usage[corev1.ResourceMemory],
			}
		}
	}

	var analyses []PodAnalysis

	// Analyze each pod
	for _, pod := range pods.Items {
		// Check HPA once per pod (not per container)
		hasHPA, hpaName := a.checkHPA(ctx, pod)
		workloadKind, workloadName := getTopLevelOwner(pod)

		for _, container := range pod.Spec.Containers {
			analysis := PodAnalysis{
				Name:          pod.Name,
				Namespace:     pod.Namespace,
				ContainerName: container.Name,
				HasHPA:        hasHPA,
				HPAName:       hpaName,
				WorkloadType:  workloadKind,
				WorkloadName:  workloadName,
				Environment:   environment,
			}

			// Get requested resources
			if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				analysis.RequestedCPU = cpu.MilliValue()
			}
			if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				analysis.RequestedMemory = mem.Value()
			}

			// Get actual usage from metrics
			if podMetrics, ok := metricsMap[pod.Name]; ok {
				if containerMetrics, ok := podMetrics[container.Name]; ok {
					analysis.ActualCPU = containerMetrics.cpu.MilliValue()
					analysis.ActualMemory = containerMetrics.memory.Value()
				}
			}

			// Calculate utilization
			if analysis.RequestedCPU > 0 {
				analysis.CPUUtilization = float64(analysis.ActualCPU) / float64(analysis.RequestedCPU) * 100
			}
			if analysis.RequestedMemory > 0 {
				analysis.MemoryUtilization = float64(analysis.ActualMemory) / float64(analysis.RequestedMemory) * 100
			}

			analyses = append(analyses, analysis)
		}
	}

	return analyses, nil
}
