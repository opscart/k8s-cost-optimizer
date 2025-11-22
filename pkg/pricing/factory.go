package pricing

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// NewProvider creates a pricing provider based on cloud detection or config
func NewProvider(ctx context.Context, clientset *kubernetes.Clientset, config *Config) (Provider, error) {
	var provider string
	var region string

	if config.Provider != "" {
		// Use configured provider
		provider = config.Provider
		region = config.Region
	} else {
		// Auto-detect from cluster
		var err error
		provider, region, err = DetectProvider(ctx, clientset)
		if err != nil {
			// Fallback to default
			provider = "default"
			region = "unknown"
		}
	}

	switch provider {
	case "azure":
		return NewAzureProvider(region), nil
	case "aws":
		return NewAWSProvider(region), nil
	case "gcp":
		return NewGCPProvider(region), nil
	case "default":
		return NewDefaultProvider(config.DefaultCPU, config.DefaultMemory), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}
