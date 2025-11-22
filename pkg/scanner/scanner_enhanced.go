package scanner

import (
	"context"

	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
)

// NewWithPricing creates a scanner with a specific pricing provider
func (s *Scanner) WithPricing(provider pricing.Provider) *Scanner {
	s.recommender = recommender.NewWithPricing(provider)
	return s
}

// GetPricingProvider returns the current pricing provider
func (s *Scanner) GetPricingProvider() pricing.Provider {
	// Try to auto-detect if not already set
	if s.recommender == nil {
		return nil
	}
	
	ctx := context.Background()
	provider, region, err := pricing.DetectProvider(ctx, s.clientset)
	if err != nil {
		return pricing.NewDefaultProvider(23.0, 3.0)
	}

	config := &pricing.Config{
		Provider:      provider,
		Region:        region,
		DefaultCPU:    23.0,
		DefaultMemory: 3.0,
	}

	pricingProvider, err := pricing.NewProvider(ctx, s.clientset, config)
	if err != nil {
		return pricing.NewDefaultProvider(23.0, 3.0)
	}

	return pricingProvider
}
