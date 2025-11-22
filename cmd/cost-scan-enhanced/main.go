package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
	"github.com/opscart/k8s-cost-optimizer/pkg/scanner"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	namespace     string
	allNamespaces bool
	provider      string
	region        string
	autoDetect    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cost-scan-enhanced",
		Short: "Enhanced cost scanner with cloud-specific pricing",
		Run:   runScan,
	}

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scan")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Scan all namespaces")
	rootCmd.Flags().StringVar(&provider, "provider", "", "Cloud provider: azure, aws, gcp, default")
	rootCmd.Flags().StringVar(&region, "region", "", "Cloud region")
	rootCmd.Flags().BoolVar(&autoDetect, "auto-detect", true, "Auto-detect cloud provider from cluster")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) {
	if namespace == "" && !allNamespaces {
		fmt.Fprintln(os.Stderr, "Error: either --namespace or --all-namespaces must be specified")
		os.Exit(1)
	}

	// Setup Kubernetes client
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Setup pricing provider
	var pricingProvider pricing.Provider

	if autoDetect && provider == "" {
		fmt.Println("[INFO] Auto-detecting cloud provider...")
		detectedProvider, detectedRegion, err := pricing.DetectProvider(ctx, clientset)
		if err != nil {
			fmt.Printf("[WARN] Auto-detection failed: %v\n", err)
			provider = "default"
			region = "unknown"
		} else {
			provider = detectedProvider
			region = detectedRegion
		}
		fmt.Printf("[INFO] Detected: %s (region: %s)\n", provider, region)
	}

	// Create pricing provider
	pricingConfig := &pricing.Config{
		Provider:      provider,
		Region:        region,
		DefaultCPU:    23.0,
		DefaultMemory: 3.0,
	}

	pricingProvider, err = pricing.NewProvider(ctx, clientset, pricingConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating pricing provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[INFO] Using %s pricing\n", pricingProvider.Name())

	// Get cost info to show user
	costInfo, _ := pricingProvider.GetCostInfo(ctx, region, "")
	fmt.Printf("[INFO] Pricing: CPU=$%.2f/core/mo, Memory=$%.2f/GiB/mo\n\n",
		costInfo.CPUCostPerCore, costInfo.MemoryCostPerGiB)

	// Initialize scanner with pricing
	scan, err := scanner.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create recommender with pricing provider
	rec := recommender.NewWithPricing(pricingProvider)

	// Scan cluster
	var recommendations []*recommender.Recommendation
	if allNamespaces {
		fmt.Println("[INFO] Scanning all namespaces...")
		// For now, just scan specific namespace
		fmt.Println("[WARN] All namespaces not fully implemented, scanning cost-test")
		analyses, err := scan.GetAnalyzer().AnalyzePods(ctx, "cost-test")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Group by deployment
		deploymentPods := make(map[string][]interface{})
		for _, analysis := range analyses {
			// Simple grouping
			deploymentPods["deployment"] = append(deploymentPods["deployment"], analysis)
		}

		for _, pods := range deploymentPods {
			if podList, ok := pods.([]interface{}); ok && len(podList) > 0 {
				// Convert to PodAnalysis slice
				// This is simplified - in production, group by actual deployment name
			}
		}
	} else {
		fmt.Printf("[INFO] Scanning namespace: %s\n", namespace)
		analyses, err := scan.GetAnalyzer().AnalyzePods(ctx, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// For each pod, create recommendation
		// Simplified: treat each pod as its own deployment
		for _, analysis := range analyses {
			recommendation := rec.Analyze([]interface{}{analysis}, analysis.Name)
			if recommendation != nil && recommendation.Type != recommender.NoAction {
				recommendations = append(recommendations, recommendation)
			}
		}
	}

	// Output results
	if len(recommendations) == 0 {
		fmt.Println("[INFO] No optimization opportunities found")
		return
	}

	fmt.Printf("[INFO] Found %d recommendation(s)\n\n", len(recommendations))
	fmt.Println("=== Optimization Recommendations ===\n")

	totalSavings := 0.0
	for i, r := range recommendations {
		fmt.Printf("%d. %s\n", i+1, r.String())
		fmt.Println()
		totalSavings += r.Savings
	}

	fmt.Printf("Total potential savings: $%.2f/month (using %s pricing)\n", totalSavings, provider)
}
