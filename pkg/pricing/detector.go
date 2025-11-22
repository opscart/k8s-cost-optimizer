package pricing

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DetectProvider attempts to detect the cloud provider from Kubernetes node labels
func DetectProvider(ctx context.Context, clientset *kubernetes.Clientset) (string, string, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return "default", "unknown", err
	}

	if len(nodes.Items) == 0 {
		return "default", "unknown", nil
	}

	node := nodes.Items[0]
	labels := node.Labels

	// Check provider ID first
	if providerID := node.Spec.ProviderID; providerID != "" {
		if strings.HasPrefix(providerID, "azure://") {
			region := extractAzureRegion(labels)
			return "azure", region, nil
		}
		if strings.HasPrefix(providerID, "aws://") {
			region := extractAWSRegion(labels)
			return "aws", region, nil
		}
		if strings.HasPrefix(providerID, "gce://") {
			region := extractGCPRegion(labels)
			return "gcp", region, nil
		}
	}

	// Check common labels
	if _, exists := labels["kubernetes.azure.com/cluster"]; exists {
		region := extractAzureRegion(labels)
		return "azure", region, nil
	}

	if _, exists := labels["eks.amazonaws.com/nodegroup"]; exists {
		region := extractAWSRegion(labels)
		return "aws", region, nil
	}

	if _, exists := labels["cloud.google.com/gke-nodepool"]; exists {
		region := extractGCPRegion(labels)
		return "gcp", region, nil
	}

	return "default", "unknown", nil
}

func extractAzureRegion(labels map[string]string) string {
	if region, exists := labels["topology.kubernetes.io/region"]; exists {
		return region
	}
	if region, exists := labels["failure-domain.beta.kubernetes.io/region"]; exists {
		return region
	}
	return "eastus" // default
}

func extractAWSRegion(labels map[string]string) string {
	if region, exists := labels["topology.kubernetes.io/region"]; exists {
		return region
	}
	if region, exists := labels["failure-domain.beta.kubernetes.io/region"]; exists {
		return region
	}
	return "us-east-1" // default
}

func extractGCPRegion(labels map[string]string) string {
	if region, exists := labels["topology.kubernetes.io/region"]; exists {
		return region
	}
	if region, exists := labels["failure-domain.beta.kubernetes.io/region"]; exists {
		return region
	}
	return "us-central1" // default
}
