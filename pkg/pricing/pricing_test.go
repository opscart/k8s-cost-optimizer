package pricing

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// Helper to load recorded responses
func loadRecording(t *testing.T, filename string) *models.CostInfo {
	t.Helper()
	data, err := os.ReadFile("../../testdata/pricing/" + filename)
	if err != nil {
		t.Fatalf("Failed to load recording: %v", err)
	}
	
	var costInfo models.CostInfo
	if err := json.Unmarshal(data, &costInfo); err != nil {
		t.Fatalf("Failed to parse recording: %v", err)
	}
	
	return &costInfo
}

func TestDefaultProvider(t *testing.T) {
	provider := NewDefaultProvider(23.0, 3.0)
	
	if provider.Name() != "default" {
		t.Errorf("Expected provider name 'default', got %s", provider.Name())
	}
	
	ctx := context.Background()
	costInfo, err := provider.GetCostInfo(ctx, "", "")
	
	if err != nil {
		t.Fatalf("GetCostInfo failed: %v", err)
	}
	
	if costInfo.CPUCostPerCore != 23.0 {
		t.Errorf("Expected CPU cost 23.0, got %.2f", costInfo.CPUCostPerCore)
	}
	
	if costInfo.MemoryCostPerGiB != 3.0 {
		t.Errorf("Expected memory cost 3.0, got %.2f", costInfo.MemoryCostPerGiB)
	}
	
	if costInfo.Provider != "default" {
		t.Errorf("Expected provider 'default', got %s", costInfo.Provider)
	}
}

// Contract test - uses recorded Azure response
func TestAzureProviderContract(t *testing.T) {
	// Load recorded response to verify data structure
	recording := loadRecording(t, "azure_eastus.json")
	
	// Verify structure matches expectations
	if recording.Provider != "azure" {
		t.Errorf("Expected provider 'azure', got %s", recording.Provider)
	}
	
	if recording.Region != "eastus" {
		t.Errorf("Expected region 'eastus', got %s", recording.Region)
	}
	
	if recording.CPUCostPerCore <= 0 {
		t.Error("CPU cost should be positive")
	}
	
	if recording.MemoryCostPerGiB <= 0 {
		t.Error("Memory cost should be positive")
	}
	
	if recording.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got %s", recording.Currency)
	}
	
	// Azure is typically more expensive than default
	if recording.CPUCostPerCore < 30.0 {
		t.Errorf("Azure CPU cost seems low: %.2f", recording.CPUCostPerCore)
	}
}

func TestAWSProviderContract(t *testing.T) {
	recording := loadRecording(t, "aws_us-east-1.json")
	
	if recording.Provider != "aws" {
		t.Errorf("Expected provider 'aws', got %s", recording.Provider)
	}
	
	if recording.CPUCostPerCore <= 0 {
		t.Error("CPU cost should be positive")
	}
	
	// AWS should be in reasonable range
	if recording.CPUCostPerCore < 20.0 || recording.CPUCostPerCore > 50.0 {
		t.Errorf("AWS CPU cost out of expected range: %.2f", recording.CPUCostPerCore)
	}
}

func TestGCPProviderContract(t *testing.T) {
	recording := loadRecording(t, "gcp_us-central1.json")
	
	if recording.Provider != "gcp" {
		t.Errorf("Expected provider 'gcp', got %s", recording.Provider)
	}
	
	if recording.CPUCostPerCore <= 0 {
		t.Error("CPU cost should be positive")
	}
	
	// GCP should be cheapest
	azureRecording := loadRecording(t, "azure_eastus.json")
	if recording.CPUCostPerCore >= azureRecording.CPUCostPerCore {
		t.Error("Expected GCP to be cheaper than Azure")
	}
}

func TestPriceCache(t *testing.T) {
	cache := NewPriceCache(100 * time.Millisecond)
	
	// Test empty cache
	result := cache.Get("test-key")
	if result != nil {
		t.Error("Expected nil for non-existent key")
	}
	
	// Test set and get
	testCost := &models.CostInfo{
		Provider:         "test",
		CPUCostPerCore:   10.0,
		MemoryCostPerGiB: 5.0,
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}
	
	cache.Set("test-key", testCost)
	
	result = cache.Get("test-key")
	if result == nil {
		t.Fatal("Expected cached value, got nil")
	}
	
	if result.CPUCostPerCore != 10.0 {
		t.Errorf("Expected CPU cost 10.0, got %.2f", result.CPUCostPerCore)
	}
	
	// Test expiration
	time.Sleep(150 * time.Millisecond)
	result = cache.Get("test-key")
	if result != nil {
		t.Error("Expected nil for expired cache entry")
	}
}

func TestProviderPriceComparison(t *testing.T) {
	// Load all recordings
	azure := loadRecording(t, "azure_eastus.json")
	aws := loadRecording(t, "aws_us-east-1.json")
	gcp := loadRecording(t, "gcp_us-central1.json")
	
	// GCP should be cheapest
	if gcp.CPUCostPerCore >= azure.CPUCostPerCore {
		t.Errorf("Expected GCP (%.2f) < Azure (%.2f)", 
			gcp.CPUCostPerCore, azure.CPUCostPerCore)
	}
	
	// Azure should be most expensive
	if azure.CPUCostPerCore < gcp.CPUCostPerCore {
		t.Errorf("Expected Azure (%.2f) > GCP (%.2f)", 
			azure.CPUCostPerCore, gcp.CPUCostPerCore)
	}
	
	// All should be reasonable
	providers := []*models.CostInfo{azure, aws, gcp}
	for _, p := range providers {
		if p.CPUCostPerCore < 10.0 || p.CPUCostPerCore > 100.0 {
			t.Errorf("Provider %s has unreasonable CPU cost: %.2f", 
				p.Provider, p.CPUCostPerCore)
		}
	}
}
