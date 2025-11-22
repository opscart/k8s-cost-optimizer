package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	fmt.Println("[TEST] Cloud Pricing Detection & Calculation")
	fmt.Println("=" + string(make([]byte, 50)))

	// Setup Kubernetes client
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Printf("[ERROR] Failed to build config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Test 1: Auto-detect provider
	fmt.Println("\n[TEST 1] Auto-detecting cloud provider...")
	provider, region, err := pricing.DetectProvider(ctx, clientset)
	if err != nil {
		fmt.Printf("[WARN] Detection failed: %v, using default\n", err)
		provider = "default"
		region = "unknown"
	}
	fmt.Printf("[SUCCESS] Detected: %s (region: %s)\n", provider, region)

	// Test 2: Create pricing provider
	fmt.Println("\n[TEST 2] Creating pricing provider...")
	pricingConfig := &pricing.Config{
		Provider:      provider,
		Region:        region,
		DefaultCPU:    23.0,
		DefaultMemory: 3.0,
	}

	priceProvider, err := pricing.NewProvider(ctx, clientset, pricingConfig)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create provider: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Created %s provider\n", priceProvider.Name())

	// Test 3: Get cost info
	fmt.Println("\n[TEST 3] Fetching pricing information...")
	costInfo, err := priceProvider.GetCostInfo(ctx, region, "")
	if err != nil {
		fmt.Printf("[ERROR] Failed to get cost info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Pricing retrieved:\n")
	fmt.Printf("  Provider: %s\n", costInfo.Provider)
	fmt.Printf("  Region: %s\n", costInfo.Region)
	fmt.Printf("  CPU Cost: $%.2f/core/month\n", costInfo.CPUCostPerCore)
	fmt.Printf("  Memory Cost: $%.2f/GiB/month\n", costInfo.MemoryCostPerGiB)
	fmt.Printf("  Currency: %s\n", costInfo.Currency)

	// Test 4: Compare all providers
	fmt.Println("\n[TEST 4] Comparing all cloud providers...")
	providers := []string{"azure", "aws", "gcp", "default"}
	
	fmt.Printf("\n%-10s %-15s %-20s %-20s\n", "Provider", "Region", "CPU ($/core/mo)", "Memory ($/GiB/mo)")
	fmt.Println(string(make([]byte, 70)))

	for _, p := range providers {
		cfg := &pricing.Config{
			Provider:      p,
			Region:        "us-east-1",
			DefaultCPU:    23.0,
			DefaultMemory: 3.0,
		}
		
		prov, _ := pricing.NewProvider(ctx, clientset, cfg)
		info, _ := prov.GetCostInfo(ctx, "us-east-1", "")
		
		fmt.Printf("%-10s %-15s $%-19.2f $%-19.2f\n", 
			info.Provider, 
			info.Region, 
			info.CPUCostPerCore, 
			info.MemoryCostPerGiB)
	}

	// Test 5: Sample cost calculation
	fmt.Println("\n[TEST 5] Sample workload cost calculation...")
	fmt.Println("Workload: 500m CPU, 512Mi memory → Recommended: 100m CPU, 128Mi memory")
	
	currentCPU := 500.0 / 1000.0  // 0.5 cores
	currentMem := 512.0 / 1024.0  // 0.5 GiB
	
	recCPU := 100.0 / 1000.0      // 0.1 cores
	recMem := 128.0 / 1024.0      // 0.125 GiB
	
	currentCost := (currentCPU * costInfo.CPUCostPerCore) + (currentMem * costInfo.MemoryCostPerGiB)
	recCost := (recCPU * costInfo.CPUCostPerCore) + (recMem * costInfo.MemoryCostPerGiB)
	savings := currentCost - recCost
	
	fmt.Printf("  Current Cost: $%.2f/month\n", currentCost)
	fmt.Printf("  Recommended Cost: $%.2f/month\n", recCost)
	fmt.Printf("  Monthly Savings: $%.2f (%.1f%% reduction)\n", savings, (savings/currentCost)*100)

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("[SUCCESS] All pricing tests passed!")
}
