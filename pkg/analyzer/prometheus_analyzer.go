package analyzer

import (
	"context"
	"fmt"

	"github.com/opscart/k8s-cost-optimizer/pkg/config"
	"github.com/opscart/k8s-cost-optimizer/pkg/datasource"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PrometheusAnalyzer uses Prometheus for historical metrics
type PrometheusAnalyzer struct {
	clientset  *kubernetes.Clientset
	prometheus *datasource.PrometheusSource
	config     *config.Config
}

// NewPrometheusAnalyzer creates an analyzer that uses Prometheus
func NewPrometheusAnalyzer(clientset *kubernetes.Clientset, prometheusURL string, cfg *config.Config) (*PrometheusAnalyzer, error) {
	prom, err := datasource.NewPrometheusSource(prometheusURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusAnalyzer{
		clientset:  clientset,
		prometheus: prom,
		config:     cfg,
	}, nil
}

// AnalyzePodsWithPrometheus analyzes pods using Prometheus P95/P99 metrics
func (a *PrometheusAnalyzer) AnalyzePodsWithPrometheus(ctx context.Context, namespace string) ([]PodAnalysis, error) {
	// Get all pods in namespace
	pods, err := a.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	var analyses []PodAnalysis

	for _, pod := range pods.Items {
		// Skip pods without containers
		if len(pod.Spec.Containers) == 0 {
			continue
		}

		// For each container in the pod
		for _, container := range pod.Spec.Containers {
			// Get resource requests
			cpuReq := container.Resources.Requests.Cpu().MilliValue()
			memReq := container.Resources.Requests.Memory().Value()

			// Create workload model for Prometheus query
			workload := &models.Workload{
				Namespace: namespace,
				Pod:       pod.Name,
				Container: container.Name,
			}

			// Get P95/P99 metrics from Prometheus
			metrics, err := a.prometheus.GetMetrics(ctx, workload, a.config.MetricsDuration)
			if err != nil {
				fmt.Printf("[WARN] Failed to get Prometheus metrics for %s: %v\n", pod.Name, err)
				// Continue with zero metrics - will be handled later
				metrics = &models.Metrics{
					P95CPU:          0,
					P95Memory:       0,
					RequestedCPU:    cpuReq,
					RequestedMemory: memReq,
				}
			}

			// Calculate utilization based on P95 (not instant)
			cpuUtil := 0.0
			if cpuReq > 0 {
				cpuUtil = float64(metrics.P95CPU) / float64(cpuReq) * 100
			}

			memUtil := 0.0
			if memReq > 0 {
				memUtil = float64(metrics.P95Memory) / float64(memReq) * 100
			}

			analysis := PodAnalysis{
				Name:              pod.Name,
				Namespace:         namespace,
				ContainerName:     container.Name,
				RequestedCPU:      cpuReq,
				RequestedMemory:   memReq,
				ActualCPU:         metrics.P95CPU, // Use P95 instead of instant
				ActualMemory:      metrics.P95Memory,
				CPUUtilization:    cpuUtil,
				MemoryUtilization: memUtil,
			}

			analyses = append(analyses, analysis)
		}
	}

	return analyses, nil
}

// IsAvailable checks if Prometheus is reachable
func (a *PrometheusAnalyzer) IsAvailable(ctx context.Context) bool {
	return a.prometheus.IsAvailable(ctx)
}
