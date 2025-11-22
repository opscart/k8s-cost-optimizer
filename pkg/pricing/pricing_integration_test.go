//go:build integration
// +build integration

package pricing

import (
	"context"
	"testing"
	"time"
)

// These tests make REAL API calls
// Run with: go test -tags=integration ./pkg/pricing -v

func TestAzureRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	provider := NewAzureProvider("eastus")
	
	ctx := context.Background()
	costInfo, err := provider.GetCostInfo(ctx, "eastus", "")
	
	if err != nil {
		t.Fatalf("Real Azure API failed: %v", err)
	}
	
	// Validate real response
	if costInfo.Provider != "azure" {
		t.Errorf("Expected provider 'azure', got %s", costInfo.Provider)
	}
	
	if costInfo.CPUCostPerCore <= 0 {
		t.Error("Azure returned zero or negative CPU cost")
	}
	
	if costInfo.MemoryCostPerGiB <= 0 {
		t.Error("Azure returned zero or negative memory cost")
	}
	
	// Check timestamp is recent (within 1 minute)
	if time.Since(costInfo.LastUpdated) > time.Minute {
		t.Errorf("Timestamp seems stale: %v", costInfo.LastUpdated)
	}
	
	t.Logf("Azure API returned: CPU=$%.2f/core, Memory=$%.2f/GiB", 
		costInfo.CPUCostPerCore, costInfo.MemoryCostPerGiB)
}

func TestAWSRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	provider := NewAWSProvider("us-east-1")
	
	ctx := context.Background()
	costInfo, err := provider.GetCostInfo(ctx, "us-east-1", "")
	
	if err != nil {
		t.Fatalf("Real AWS API failed: %v", err)
	}
	
	if costInfo.CPUCostPerCore <= 0 {
		t.Error("AWS returned zero or negative CPU cost")
	}
	
	t.Logf("AWS API returned: CPU=$%.2f/core, Memory=$%.2f/GiB", 
		costInfo.CPUCostPerCore, costInfo.MemoryCostPerGiB)
}

func TestGCPRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	provider := NewGCPProvider("us-central1")
	
	ctx := context.Background()
	costInfo, err := provider.GetCostInfo(ctx, "us-central1", "")
	
	if err != nil {
		t.Fatalf("Real GCP API failed: %v", err)
	}
	
	if costInfo.CPUCostPerCore <= 0 {
		t.Error("GCP returned zero or negative CPU cost")
	}
	
	t.Logf("GCP API returned: CPU=$%.2f/core, Memory=$%.2f/GiB", 
		costInfo.CPUCostPerCore, costInfo.MemoryCostPerGiB)
}

// Test that real APIs return consistent pricing
func TestRealAPIPriceConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	
	azure := NewAzureProvider("eastus")
	aws := NewAWSProvider("us-east-1")
	gcp := NewGCPProvider("us-central1")
	
	azureCost, _ := azure.GetCostInfo(ctx, "eastus", "")
	awsCost, _ := aws.GetCostInfo(ctx, "us-east-1", "")
	gcpCost, _ := gcp.GetCostInfo(ctx, "us-central1", "")
	
	// GCP should still be cheapest
	if gcpCost.CPUCostPerCore >= azureCost.CPUCostPerCore {
		t.Errorf("Pricing changed! GCP (%.2f) should be < Azure (%.2f)",
			gcpCost.CPUCostPerCore, azureCost.CPUCostPerCore)
	}
	
	t.Logf("Real pricing comparison:")
	t.Logf("  Azure: $%.2f/core", azureCost.CPUCostPerCore)
	t.Logf("  AWS:   $%.2f/core", awsCost.CPUCostPerCore)
	t.Logf("  GCP:   $%.2f/core", gcpCost.CPUCostPerCore)
}
