package recommender

import (
	"testing"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
)

func TestNew(t *testing.T) {
	rec := New()
	
	if rec == nil {
		t.Fatal("New() returned nil")
	}
	
	if rec.safetyBuffer != 1.5 {
		t.Errorf("Expected default safety buffer 1.5, got %.1f", rec.safetyBuffer)
	}
}

func TestNewWithPricing(t *testing.T) {
	provider := pricing.NewDefaultProvider(23.0, 3.0)
	rec := NewWithPricing(provider)
	
	if rec == nil {
		t.Fatal("NewWithPricing() returned nil")
	}
	
	if rec.pricingProvider == nil {
		t.Fatal("Pricing provider not set")
	}
}

func TestRightSizeRecommendation(t *testing.T) {
	rec := New()
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "overprovision-pod",
			Namespace:         "default",
			RequestedCPU:      1000,
			RequestedMemory:   1024 * 1024 * 1024,
			ActualCPU:         200,
			ActualMemory:      256 * 1024 * 1024,
			CPUUtilization:    20.0,
			MemoryUtilization: 25.0,
		},
	}
	
	recommendation := rec.Analyze(analyses, "overprovision-deployment")
	
	if recommendation == nil {
		t.Fatal("Expected recommendation, got nil")
	}
	
	if recommendation.Type != RightSize {
		t.Errorf("Expected RIGHT_SIZE, got %s", recommendation.Type)
	}
	
	if recommendation.Savings <= 0 {
		t.Errorf("Expected positive savings, got %.2f", recommendation.Savings)
	}
}

func TestPricingProviderIntegration(t *testing.T) {
	providers := map[string]pricing.Provider{
		"default": pricing.NewDefaultProvider(23.0, 3.0),
		"azure":   pricing.NewAzureProvider("eastus"),
		"aws":     pricing.NewAWSProvider("us-east-1"),
		"gcp":     pricing.NewGCPProvider("us-central1"),
	}
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "test-pod",
			Namespace:         "default",
			RequestedCPU:      1000,
			RequestedMemory:   1024 * 1024 * 1024,
			ActualCPU:         200,
			ActualMemory:      256 * 1024 * 1024,
			CPUUtilization:    20.0,
			MemoryUtilization: 25.0,
		},
	}
	
	for name, provider := range providers {
		rec := NewWithPricing(provider)
		recommendation := rec.Analyze(analyses, "test-deployment")
		
		if recommendation == nil {
			t.Fatalf("Provider %s returned nil", name)
		}
		
		if recommendation.Provider != name {
			t.Errorf("Expected provider %s, got %s", name, recommendation.Provider)
		}
		
		t.Logf("Provider %s: Savings = $%.2f/month", name, recommendation.Savings)
	}
}

func TestEmptyAnalyses(t *testing.T) {
	rec := New()
	
	recommendation := rec.Analyze([]analyzer.PodAnalysis{}, "empty-deployment")
	
	if recommendation != nil {
		t.Error("Expected nil for empty analyses")
	}
}

func TestScaleDownRecommendation(t *testing.T) {
	rec := New()
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "idle-pod",
			Namespace:         "default",
			RequestedCPU:      1000,
			RequestedMemory:   1024 * 1024 * 1024,
			ActualCPU:         20,
			ActualMemory:      50 * 1024 * 1024,
			CPUUtilization:    2.0,
			MemoryUtilization: 5.0,
		},
	}
	
	recommendation := rec.Analyze(analyses, "idle-deployment")
	
	if recommendation == nil {
		t.Fatal("Expected recommendation, got nil")
	}
	
	if recommendation.Type != ScaleDown {
		t.Errorf("Expected SCALE_DOWN, got %s", recommendation.Type)
	}
	
	if recommendation.RecommendedCPU != 0 {
		t.Errorf("Expected recommended CPU 0, got %d", recommendation.RecommendedCPU)
	}
}

func TestNoActionRecommendation(t *testing.T) {
	rec := New()
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "well-sized-pod",
			Namespace:         "default",
			RequestedCPU:      100,
			RequestedMemory:   128 * 1024 * 1024,
			ActualCPU:         75,
			ActualMemory:      100 * 1024 * 1024,
			CPUUtilization:    75.0,
			MemoryUtilization: 78.0,
		},
	}
	
	recommendation := rec.Analyze(analyses, "well-sized-deployment")
	
	if recommendation == nil {
		t.Fatal("Expected recommendation, got nil")
	}
	
	if recommendation.Type != NoAction {
		t.Errorf("Expected NO_ACTION, got %s", recommendation.Type)
	}
}

func TestMinimumThresholds(t *testing.T) {
	rec := New()
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "tiny-pod",
			Namespace:         "default",
			RequestedCPU:      50,
			RequestedMemory:   64 * 1024 * 1024,
			ActualCPU:         5,
			ActualMemory:      5 * 1024 * 1024,
			CPUUtilization:    10.0,
			MemoryUtilization: 8.0,
		},
	}
	
	recommendation := rec.Analyze(analyses, "tiny-deployment")
	
	if recommendation.RecommendedCPU < 10 {
		t.Errorf("Expected minimum CPU 10m, got %d", recommendation.RecommendedCPU)
	}
}

func TestMultiplePods(t *testing.T) {
	rec := New()
	
	analyses := []analyzer.PodAnalysis{
		{
			Name:              "pod-1",
			Namespace:         "default",
			RequestedCPU:      500,
			RequestedMemory:   512 * 1024 * 1024,
			ActualCPU:         100,
			ActualMemory:      128 * 1024 * 1024,
			CPUUtilization:    20.0,
			MemoryUtilization: 25.0,
		},
		{
			Name:              "pod-2",
			Namespace:         "default",
			RequestedCPU:      500,
			RequestedMemory:   512 * 1024 * 1024,
			ActualCPU:         120,
			ActualMemory:      140 * 1024 * 1024,
			CPUUtilization:    24.0,
			MemoryUtilization: 27.0,
		},
	}
	
	recommendation := rec.Analyze(analyses, "multi-pod-deployment")
	
	if recommendation == nil {
		t.Fatal("Expected recommendation, got nil")
	}
	
	if recommendation.Savings <= 0 {
		t.Errorf("Expected positive savings, got %.2f", recommendation.Savings)
	}
}
