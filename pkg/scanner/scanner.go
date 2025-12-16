package scanner

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
	"github.com/prometheus/client_golang/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Scanner struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	analyzer      *analyzer.Analyzer
	recommender   *recommender.Recommender
	verbose       bool
}

func New(kubeconfigPath string, verbose bool) (*Scanner, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first (for pods running in Kubernetes)
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		var kubeconfig string
		if kubeconfigPath != "" {
			kubeconfig = kubeconfigPath
		} else if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	metricsClient, err := metricsv.NewForConfig(config)

	return &Scanner{
		clientset:     clientset,
		metricsClient: metricsClient,
		analyzer:      analyzer.New(clientset, metricsClient),
		recommender:   recommender.New(),
		verbose:       verbose,
	}, nil
}

func (s *Scanner) ScanAndRecommend(namespace string, allNamespaces bool) ([]*recommender.Recommendation, error) {
	ctx := context.Background()

	version, err := s.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	fmt.Printf("[INFO] Connected to cluster (version: %s)\n", version.GitVersion)

	namespaces := []string{namespace}
	if allNamespaces {
		nsList, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}
		namespaces = []string{}
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, ns.Name)
		}
		fmt.Printf("[INFO] Scanning %d namespaces\n", len(namespaces))
	} else {
		fmt.Printf("[INFO] Scanning namespace: %s\n", namespace)
	}

	var allRecommendations []*recommender.Recommendation

	for _, ns := range namespaces {
		recommendations, err := s.scanNamespace(ctx, ns)
		if err != nil {
			fmt.Printf("[WARN] Error scanning namespace %s: %v\n", ns, err)
			continue
		}
		allRecommendations = append(allRecommendations, recommendations...)
	}

	return allRecommendations, nil
}

func (s *Scanner) scanNamespace(ctx context.Context, namespace string) ([]*recommender.Recommendation, error) {
	// Get all workload types
	deployments, err := s.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	statefulSets, err := s.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}

	daemonSets, err := s.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

	replicaSets, err := s.clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list replicasets: %w", err)
	}

	// Get pod analyses
	analyses, err := s.analyzer.AnalyzePods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze pods: %w", err)
	}

	// Group pods by their parent workload
	workloadPods := make(map[string][]analyzer.PodAnalysis)

	for _, analysis := range analyses {
		// Use the workload name from the analysis (already extracted in analyzer)
		workloadKey := analysis.WorkloadName
		if workloadKey == "" {
			// Fallback: try to extract from pod name
			workloadKey = extractWorkloadName(analysis.Name)
		}

		if workloadKey != "" {
			workloadPods[workloadKey] = append(workloadPods[workloadKey], analysis)
		}
	}

	var recommendations []*recommender.Recommendation

	// Generate recommendations for Deployments
	for _, deploy := range deployments.Items {
		if pods, exists := workloadPods[deploy.Name]; exists && len(pods) > 0 {
			rec := s.recommender.Analyze(pods, deploy.Name)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Generate recommendations for StatefulSets
	for _, sts := range statefulSets.Items {
		if pods, exists := workloadPods[sts.Name]; exists && len(pods) > 0 {
			rec := s.recommender.Analyze(pods, sts.Name)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Generate recommendations for DaemonSets
	for _, ds := range daemonSets.Items {
		if pods, exists := workloadPods[ds.Name]; exists && len(pods) > 0 {
			rec := s.recommender.Analyze(pods, ds.Name)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Generate recommendations for standalone ReplicaSets (not owned by Deployments)
	for _, rs := range replicaSets.Items {
		// Skip ReplicaSets owned by Deployments (already handled above)
		if len(rs.OwnerReferences) > 0 && rs.OwnerReferences[0].Kind == "Deployment" {
			continue
		}

		if pods, exists := workloadPods[rs.Name]; exists && len(pods) > 0 {
			rec := s.recommender.Analyze(pods, rs.Name)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	return recommendations, nil
}

// extractWorkloadName extracts workload name from pod name
// Handles formats like: "workload-name-7d9f8b-xyz" (Deployment) or "workload-name-0" (StatefulSet)
func extractWorkloadName(podName string) string {
	// For StatefulSets: "postgres-test-0" -> "postgres-test"
	// For Deployments: "api-server-7d9f8b-xyz" -> "api-server"

	// Try StatefulSet pattern first (ends with -<number>)
	if len(podName) > 2 && podName[len(podName)-2] == '-' {
		// Check if last char is a digit
		lastChar := podName[len(podName)-1]
		if lastChar >= '0' && lastChar <= '9' {
			return podName[:len(podName)-2]
		}
	}

	// Try Deployment pattern (remove last two dash-separated segments)
	dashCount := 0
	for i := len(podName) - 1; i >= 0; i-- {
		if podName[i] == '-' {
			dashCount++
			if dashCount == 2 {
				return podName[:i]
			}
		}
	}

	return podName
}

// GetAnalyzer returns the analyzer for direct use
func (s *Scanner) GetAnalyzer() *analyzer.Analyzer {
	return s.analyzer
}

// GetClientset returns the Kubernetes clientset for direct access
func (s *Scanner) GetClientset() *kubernetes.Clientset {
	return s.clientset
}

// Add this new method to Scanner struct
func (s *Scanner) ScanAndRecommendWithHistory(
	ctx context.Context,
	namespace string,
	allNamespaces bool,
	promClient api.Client,
	lookbackDays int,
) ([]*recommender.Recommendation, error) {

	// Get list of namespaces to scan
	namespaces := []string{namespace}
	if allNamespaces {
		nsList, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}
		namespaces = []string{}
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, ns.Name)
		}
	}

	var allRecommendations []*recommender.Recommendation

	// Create historical analyzer
	histAnalyzer := analyzer.NewHistoricalAnalyzer(promClient, s.verbose)

	for _, ns := range namespaces {
		recommendations, err := s.scanNamespaceWithHistory(ctx, ns, histAnalyzer, lookbackDays)
		if err != nil {
			fmt.Printf("[WARN] Error scanning namespace %s: %v\n", ns, err)
			continue
		}
		allRecommendations = append(allRecommendations, recommendations...)
	}

	return allRecommendations, nil
}

// scanNamespaceWithHistory scans a namespace using historical data
func (s *Scanner) scanNamespaceWithHistory(
	ctx context.Context,
	namespace string,
	histAnalyzer *analyzer.HistoricalAnalyzer,
	lookbackDays int,
) ([]*recommender.Recommendation, error) {

	// Get all workload types
	deployments, _ := s.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	statefulSets, _ := s.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	daemonSets, _ := s.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})

	// Get current pod analyses (for workload type, environment, etc.)
	currentAnalyses, err := s.analyzer.AnalyzePods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze pods: %w", err)
	}

	// Group by workload
	workloadPods := make(map[string][]analyzer.PodAnalysis)
	for _, analysis := range currentAnalyses {
		workloadKey := analysis.WorkloadName
		if workloadKey == "" {
			workloadKey = extractWorkloadName(analysis.Name)
		}
		if workloadKey != "" {
			workloadPods[workloadKey] = append(workloadPods[workloadKey], analysis)
		}
	}

	var recommendations []*recommender.Recommendation

	// Process deployments
	for _, deploy := range deployments.Items {
		if pods, exists := workloadPods[deploy.Name]; exists && len(pods) > 0 {
			rec := s.generateHistoricalRecommendation(ctx, deploy.Name, pods, histAnalyzer, lookbackDays)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Process StatefulSets
	for _, sts := range statefulSets.Items {
		if pods, exists := workloadPods[sts.Name]; exists && len(pods) > 0 {
			rec := s.generateHistoricalRecommendation(ctx, sts.Name, pods, histAnalyzer, lookbackDays)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Process DaemonSets
	for _, ds := range daemonSets.Items {
		if pods, exists := workloadPods[ds.Name]; exists && len(pods) > 0 {
			rec := s.generateHistoricalRecommendation(ctx, ds.Name, pods, histAnalyzer, lookbackDays)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	return recommendations, nil
}

// generateHistoricalRecommendation creates recommendation using historical data
func (s *Scanner) generateHistoricalRecommendation(
	ctx context.Context,
	workloadName string,
	pods []analyzer.PodAnalysis,
	histAnalyzer *analyzer.HistoricalAnalyzer,
	lookbackDays int,
) *recommender.Recommendation {

	// Use first pod as representative (they should have similar patterns)
	pod := pods[0]

	// Get historical metrics for this pod
	histMetrics, err := histAnalyzer.GetHistoricalMetrics(
		ctx,
		pod.Namespace,
		pod.Name,
		pod.ContainerName,
		lookbackDays,
	)

	// Check for errors or insufficient data
	if err != nil || len(histMetrics.CPUSamples) == 0 || len(histMetrics.MemorySamples) == 0 {
		// Fallback to instant metrics
		if err != nil {
			fmt.Printf("[DEBUG] Historical data unavailable for %s/%s: %v\n",
				pod.Namespace, workloadName, err)
		} else {
			fmt.Printf("[DEBUG] Insufficient historical data for %s/%s (CPU samples: %d, Memory samples: %d)\n",
				pod.Namespace, workloadName, len(histMetrics.CPUSamples), len(histMetrics.MemorySamples))
		}
		return s.recommender.Analyze(pods, workloadName)
	}

	// Calculate P95/P99 from historical data
	cpuPercentiles, err := analyzer.CalculatePercentiles(histMetrics.CPUSamples)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to calculate CPU percentiles for %s/%s: %v\n",
			pod.Namespace, workloadName, err)
		return s.recommender.Analyze(pods, workloadName)
	}

	memPercentiles, err := analyzer.CalculatePercentiles(histMetrics.MemorySamples)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to calculate memory percentiles for %s/%s: %v\n",
			pod.Namespace, workloadName, err)
		return s.recommender.Analyze(pods, workloadName)
	}

	// Log success with data points
	fmt.Printf("[INFO] Using %d-day historical analysis for %s/%s (%d CPU samples, %d memory samples)\n",
		lookbackDays, pod.Namespace, workloadName, len(histMetrics.CPUSamples), len(histMetrics.MemorySamples))

	// Update pod analyses with historical P95 values AND pattern analysis
	for i := range pods {
		// Week 9 Day 3: Use higher of weekday/weekend P95 for safer recommendations
		cpuP95 := cpuPercentiles.P95
		if histMetrics.WeekdayCPUP95 > 0 || histMetrics.WeekendCPUP95 > 0 {
			// Use the higher value to ensure we handle peak load
			if histMetrics.WeekdayCPUP95 > histMetrics.WeekendCPUP95 {
				cpuP95 = histMetrics.WeekdayCPUP95
			} else {
				cpuP95 = histMetrics.WeekendCPUP95
			}
		}

		memP95 := memPercentiles.P95
		if histMetrics.WeekdayMemoryP95 > 0 || histMetrics.WeekendMemoryP95 > 0 {
			// Use the higher value
			if histMetrics.WeekdayMemoryP95 > histMetrics.WeekendMemoryP95 {
				memP95 = float64(histMetrics.WeekdayMemoryP95)
			} else {
				memP95 = float64(histMetrics.WeekendMemoryP95)
			}
		}

		pods[i].ActualCPU = int64(cpuP95)
		pods[i].ActualMemory = int64(memP95)

		// Add pattern and growth analysis (Week 9)
		pods[i].CPUPattern = histMetrics.CPUPattern
		pods[i].MemoryPattern = histMetrics.MemoryPattern
		pods[i].CPUGrowth = histMetrics.CPUGrowth
		pods[i].MemoryGrowth = histMetrics.MemoryGrowth
		pods[i].DataQuality = histMetrics.DataQuality
		pods[i].HasSufficientData = histMetrics.HasSufficientData
	}

	// Generate recommendation with historical data
	rec := s.recommender.Analyze(pods, workloadName)

	if rec != nil {
		// Update reason to show historical context
		reasonContext := fmt.Sprintf("Based on %d-day P95: CPU %.0fm, Memory %.0fMi",
			lookbackDays,
			cpuPercentiles.P95,
			memPercentiles.P95/(1024*1024),
		)

		// Week 9 Day 3: Show weekday/weekend split if they differ significantly (>20%)
		if histMetrics.WeekdayCPUP95 > 0 && histMetrics.WeekendCPUP95 > 0 {
			cpuDiff := math.Abs(histMetrics.WeekdayCPUP95 - histMetrics.WeekendCPUP95)
			avgCPU := (histMetrics.WeekdayCPUP95 + histMetrics.WeekendCPUP95) / 2
			if cpuDiff/avgCPU > 0.2 { // >20% difference
				reasonContext = fmt.Sprintf("%s (Weekday: %.0fm, Weekend: %.0fm)",
					reasonContext,
					histMetrics.WeekdayCPUP95,
					histMetrics.WeekendCPUP95,
				)
			}
		}
		// Also check memory difference
		if histMetrics.WeekdayMemoryP95 > 0 && histMetrics.WeekendMemoryP95 > 0 {
			memDiff := math.Abs(float64(histMetrics.WeekdayMemoryP95) - float64(histMetrics.WeekendMemoryP95))
			avgMem := float64(histMetrics.WeekdayMemoryP95+histMetrics.WeekendMemoryP95) / 2
			if memDiff/avgMem > 0.2 { // >20% difference
				reasonContext = fmt.Sprintf("%s, Memory (Weekday: %dMi, Weekend: %dMi)",
					reasonContext,
					histMetrics.WeekdayMemoryP95/(1024*1024),
					histMetrics.WeekendMemoryP95/(1024*1024),
				)
			}
		}

		rec.Reason = fmt.Sprintf("%s (%s)", rec.Reason, reasonContext)

		return rec
	}
	return nil
}
