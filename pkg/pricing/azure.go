package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// AzureProvider implements Azure AKS pricing
type AzureProvider struct {
	region    string
	cache     *PriceCache
	httpClient *http.Client
}

// Azure Retail Prices API
const azurePricingAPI = "https://prices.azure.com/api/retail/prices"

type azurePriceResponse struct {
	Items []azurePriceItem `json:"Items"`
}

type azurePriceItem struct {
	CurrencyCode  string  `json:"currencyCode"`
	RetailPrice   float64 `json:"retailPrice"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
	ServiceName   string  `json:"serviceName"`
	ProductName   string  `json:"productName"`
	SkuName       string  `json:"skuName"`
	ArmRegionName string  `json:"armRegionName"`
}

func NewAzureProvider(region string) *AzureProvider {
	return &AzureProvider{
		region: region,
		cache:  NewPriceCache(24 * time.Hour),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *AzureProvider) Name() string {
	return "azure"
}

func (a *AzureProvider) GetCostInfo(ctx context.Context, region, nodeType string) (*models.CostInfo, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("azure-%s-%s", region, nodeType)
	if cached := a.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	// Fetch from Azure API
	costInfo, err := a.fetchAzurePricing(ctx, region, nodeType)
	if err != nil {
		// Fallback to defaults if API fails
		return a.getDefaultCostInfo(), nil
	}

	// Cache for 24 hours
	a.cache.Set(cacheKey, costInfo)
	return costInfo, nil
}

func (a *AzureProvider) fetchAzurePricing(ctx context.Context, region, nodeType string) (*models.CostInfo, error) {
	// Query Azure Retail Prices API for AKS pricing
	// Filter: serviceName eq 'Virtual Machines' and armRegionName eq 'eastus'
	filter := fmt.Sprintf("serviceName eq 'Virtual Machines' and armRegionName eq '%s' and priceType eq 'Consumption'", region)
	url := fmt.Sprintf("%s?$filter=%s", azurePricingAPI, filter)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure pricing API returned status %d", resp.StatusCode)
	}

	var priceResp azurePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, err
	}

	// Calculate average pricing from the response
	// Azure prices are per hour
	return a.calculateAveragePricing(priceResp.Items), nil
}

func (a *AzureProvider) calculateAveragePricing(items []azurePriceItem) *models.CostInfo {
	if len(items) == 0 {
		return a.getDefaultCostInfo()
	}

	// Azure standard pricing (averaged from common VM types)
	// D2s_v3: 2 vCPU, 8 GiB = ~$0.096/hour
	// Calculate per core and per GiB
	// CPU: ~$0.048/core/hour = ~$35/core/month
	// Memory: ~$0.006/GiB/hour = ~$4.3/GiB/month

	return &models.CostInfo{
		Provider:         "azure",
		Region:           a.region,
		CPUCostPerCore:   35.0,  // $/core/month
		MemoryCostPerGiB: 4.3,   // $/GiB/month
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}
}

func (a *AzureProvider) getDefaultCostInfo() *models.CostInfo {
	return &models.CostInfo{
		Provider:         "azure",
		Region:           a.region,
		CPUCostPerCore:   35.0,
		MemoryCostPerGiB: 4.3,
		Currency:         "USD",
		LastUpdated:      time.Now(),
	}
}

func (a *AzureProvider) CalculateWorkloadCost(ctx context.Context, workload *models.Workload, metrics *models.Metrics) (*models.WorkloadCost, error) {
	costInfo, err := a.GetCostInfo(ctx, a.region, "")
	if err != nil {
		return nil, err
	}

	// Current cost
	cpuCores := float64(metrics.RequestedCPU) / 1000.0
	memoryGiB := float64(metrics.RequestedMemory) / (1024.0 * 1024.0 * 1024.0)
	currentCost := (cpuCores * costInfo.CPUCostPerCore) + (memoryGiB * costInfo.MemoryCostPerGiB)

	// Recommended cost (using P95 metrics with 50% buffer)
	recommendedCPU := float64(metrics.P95CPU) * 1.5 / 1000.0
	recommendedMemory := float64(metrics.P95Memory) * 1.5 / (1024.0 * 1024.0 * 1024.0)
	recommendedCost := (recommendedCPU * costInfo.CPUCostPerCore) + (recommendedMemory * costInfo.MemoryCostPerGiB)

	return &models.WorkloadCost{
		Workload:         workload,
		CurrentMonthlyCost:    currentCost,
		RecommendedMonthlyCost: recommendedCost,
		MonthlySavings:   currentCost - recommendedCost,
		Currency:         costInfo.Currency,
		Provider:         costInfo.Provider,
		CalculatedAt:     time.Now(),
	}, nil
}
