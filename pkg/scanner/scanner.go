package scanner

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Scanner struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	analyzer      *analyzer.Analyzer
	recommender   *recommender.Recommender
}

func New() (*Scanner, error) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	return &Scanner{
		clientset:     clientset,
		metricsClient: metricsClient,
		analyzer:      analyzer.New(clientset, metricsClient),
		recommender:   recommender.New(),
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
	deployments, err := s.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments.Items) == 0 {
		return nil, nil
	}

	analyses, err := s.analyzer.AnalyzePods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze pods: %w", err)
	}

	deploymentPods := make(map[string][]analyzer.PodAnalysis)
	for _, analysis := range analyses {
		for _, deploy := range deployments.Items {
			if len(analysis.Name) > len(deploy.Name) && analysis.Name[:len(deploy.Name)] == deploy.Name {
				deploymentPods[deploy.Name] = append(deploymentPods[deploy.Name], analysis)
				break
			}
		}
	}

	var recommendations []*recommender.Recommendation

	for deployName, pods := range deploymentPods {
		if len(pods) == 0 {
			continue
		}
		
		rec := s.recommender.Analyze(pods, deployName)
		if rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations, nil
}

// GetAnalyzer returns the analyzer for direct use
func (s *Scanner) GetAnalyzer() *analyzer.Analyzer {
	return s.analyzer
}

