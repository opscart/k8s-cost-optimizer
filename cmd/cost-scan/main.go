package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
	"github.com/opscart/k8s-cost-optimizer/pkg/config"
	"github.com/opscart/k8s-cost-optimizer/pkg/converter"
	"github.com/opscart/k8s-cost-optimizer/pkg/executor"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
	"github.com/opscart/k8s-cost-optimizer/pkg/scanner"
	"github.com/opscart/k8s-cost-optimizer/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	// Scan flags
	namespace     string
	allNamespaces bool
	outputFormat  string
	saveResults   bool
	clusterID     string
	usePrometheus bool

	// Global config
	cfg   *config.Config
	store storage.Store
	
	// History command vars
	historyLimit int
)

func main() {
	// Initialize config
	cfg = config.NewConfig()

	var rootCmd = &cobra.Command{
		Use:   "cost-scan",
		Short: "Kubernetes cost optimization scanner",
		Long:  `Scan Kubernetes clusters for cost optimization opportunities with P95/P99 metrics from Prometheus.`,
		Run:   runScan,
	}

	// Scan flags
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scan")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Scan all namespaces")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, commands")
	rootCmd.Flags().BoolVar(&saveResults, "save", false, "Save recommendations to database")
	rootCmd.Flags().StringVar(&clusterID, "cluster-id", "default", "Cluster identifier")
	rootCmd.Flags().BoolVar(&usePrometheus, "use-prometheus", true, "Use Prometheus for P95/P99 metrics (default: true)")

	// History command
	historyCmd := &cobra.Command{
		Use:   "history <namespace>",
		Short: "View past recommendations",
		Args:  cobra.ExactArgs(1),
		Run:   runHistory,
	}
	historyCmd.Flags().IntVar(&historyLimit, "limit", 10, "Number of recommendations to show")

	// Audit command
	auditCmd := &cobra.Command{
		Use:   "audit <recommendation-id>",
		Short: "View audit log",
		Args:  cobra.ExactArgs(1),
		Run:   runAudit,
	}

	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(auditCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initStorage() error {
	if !cfg.StorageEnabled || !saveResults {
		return nil
	}

	var err error
	store, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	return nil
}

func initStorageForced() error {
	var err error
	store, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	return nil
}

func runScan(cmd *cobra.Command, args []string) {
	if namespace == "" && !allNamespaces {
		fmt.Fprintln(os.Stderr, "Error: either --namespace or --all-namespaces must be specified")
		os.Exit(1)
	}

	if outputFormat != "text" && outputFormat != "json" && outputFormat != "commands" {
		fmt.Fprintln(os.Stderr, "Error: output must be text, json, or commands")
		os.Exit(1)
	}

	// Initialize storage if --save flag is used
	if saveResults {
		if err := initStorage(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()
	}

	if outputFormat != "commands" {
		fmt.Println("[INFO] K8s Cost Optimizer - Starting scan")
		if saveResults {
			fmt.Println("[INFO] Results will be saved to database")
		}
	}

	// Initialize scanner
	scan, err := scanner.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing scanner: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Try to use Prometheus if enabled
	var promAnalyzer *analyzer.PrometheusAnalyzer
	metricsSource := "metrics-server (instant snapshots)"
	
	if usePrometheus && cfg.PrometheusURL != "" {
		promAnalyzer, err = analyzer.NewPrometheusAnalyzer(
			scan.GetClientset(),
			cfg.PrometheusURL,
			cfg,
		)
		
		if err != nil {
			if outputFormat != "commands" {
				fmt.Printf("[WARN] Prometheus initialization failed: %v\n", err)
				fmt.Println("[INFO] Falling back to metrics-server")
			}
			promAnalyzer = nil
		} else if promAnalyzer.IsAvailable(ctx) {
			metricsSource = fmt.Sprintf("Prometheus P95/P99 (%d days lookback)", cfg.MetricsLookbackDays)
			if outputFormat != "commands" {
				fmt.Printf("[INFO] Using Prometheus at %s\n", cfg.PrometheusURL)
				fmt.Printf("[INFO] Metrics window: %d days, Safety buffer: %.1fx\n", 
					cfg.MetricsLookbackDays, cfg.SafetyBuffer)
			}
		} else {
			if outputFormat != "commands" {
				fmt.Println("[WARN] Prometheus not reachable, falling back to metrics-server")
			}
			promAnalyzer = nil
		}
	} else if usePrometheus && cfg.PrometheusURL == "" {
		if outputFormat != "commands" {
			fmt.Println("[INFO] Prometheus URL not configured, using metrics-server")
			fmt.Println("[INFO] Set PROMETHEUS_URL environment variable to enable Prometheus")
		}
	}

	// Auto-detect cloud provider for pricing
	detectedProvider := "default"
	detectedRegion := "unknown"
	
	if clientset := scan.GetClientset(); clientset != nil {
		detectedProvider, detectedRegion, err = pricing.DetectProvider(ctx, clientset)
		if err != nil && outputFormat != "commands" {
			fmt.Printf("[WARN] Cloud detection failed: %v, using default pricing\n", err)
			detectedProvider = "default"
		}
	}

	// Create pricing provider
	pricingConfig := &pricing.Config{
		Provider:      detectedProvider,
		Region:        detectedRegion,
		DefaultCPU:    23.0,
		DefaultMemory: 3.0,
	}

	priceProvider, err := pricing.NewProvider(ctx, scan.GetClientset(), pricingConfig)
	if err != nil {
		if outputFormat != "commands" {
			fmt.Printf("[WARN] Pricing provider failed: %v, using defaults\n", err)
		}
		priceProvider = pricing.NewDefaultProvider(23.0, 3.0)
	}

	// Get version info
	versionInfo, err := scan.GetClientset().Discovery().ServerVersion()
	if err == nil && outputFormat != "commands" {
		fmt.Printf("[INFO] Connected to cluster (version: %s)\n", versionInfo.String())
	}

	if outputFormat != "commands" {
		fmt.Printf("[INFO] Cloud provider: %s (region: %s)\n", detectedProvider, detectedRegion)
		fmt.Printf("[INFO] Metrics source: %s\n", metricsSource)
		fmt.Printf("[INFO] Scanning namespace: %s\n", namespace)
	}

	// Scan using Prometheus or fallback to metrics-server
	var oldRecommendations []*recommender.Recommendation
	
	if promAnalyzer != nil {
		// Use Prometheus P95/P99 metrics
		analyses, err := promAnalyzer.AnalyzePodsWithPrometheus(ctx, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing with Prometheus: %v\n", err)
			os.Exit(1)
		}

		if len(analyses) == 0 {
			if outputFormat != "commands" {
				fmt.Println("[INFO] No pods found in namespace")
			}
			return
		}

		// Group by deployment and generate recommendations
		deploymentPods := groupPodsByDeployment(analyses)
		rec := recommender.NewWithPricing(priceProvider)
		
		for deploymentName, pods := range deploymentPods {
			recommendation := rec.Analyze(pods, deploymentName)
			if recommendation != nil {
				oldRecommendations = append(oldRecommendations, recommendation)
			}
		}
	} else {
		// Fallback to old scanner method with metrics-server
		oldRecommendations, err = scan.ScanAndRecommend(namespace, allNamespaces)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning cluster: %v\n", err)
			os.Exit(1)
		}
	}

	if len(oldRecommendations) == 0 {
		if outputFormat != "commands" {
			fmt.Println("[INFO] No optimization opportunities found")
		}
		return
	}

	if outputFormat != "commands" {
		fmt.Printf("[INFO] Found %d recommendation(s)\n\n", len(oldRecommendations))
	}

	// Convert to new models and save if requested
	var recommendations []*models.Recommendation
	totalSavings := 0.0

	for _, oldRec := range oldRecommendations {
		// Convert to new model
		newRec := converter.OldToNew(oldRec, clusterID)
		recommendations = append(recommendations, newRec)
		totalSavings += newRec.SavingsMonthly

		// Save to database if requested
		if saveResults && store != nil {
			if err := store.SaveRecommendation(ctx, newRec); err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to save recommendation: %v\n", err)
			} else if outputFormat != "commands" {
				fmt.Printf("[INFO] Saved recommendation for %s/%s (ID: %s)\n",
					newRec.Workload.Namespace, newRec.Workload.Deployment, newRec.ID)
			}
		}
	}

	// Output results
	switch outputFormat {
	case "json":
		outputJSON(recommendations, totalSavings)
	case "commands":
		outputCommands(oldRecommendations)
	default:
		outputText(recommendations, totalSavings)
	}
}

// Helper function to group pod analyses by deployment
func groupPodsByDeployment(analyses []analyzer.PodAnalysis) map[string][]analyzer.PodAnalysis {
	deploymentPods := make(map[string][]analyzer.PodAnalysis)
	
	for _, analysis := range analyses {
		// Extract deployment name from pod name (before last dash and hash)
		deploymentName := extractDeploymentName(analysis.Name)
		deploymentPods[deploymentName] = append(deploymentPods[deploymentName], analysis)
	}
	
	return deploymentPods
}

// Extract deployment name from pod name (e.g., "api-server-7d9f8b-xyz" -> "api-server")
func extractDeploymentName(podName string) string {
	// Simple heuristic: remove last two dash-separated segments (replicaset hash + pod hash)
	parts := []rune(podName)
	dashCount := 0
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '-' {
			dashCount++
			if dashCount == 2 {
				return string(parts[:i])
			}
		}
	}
	return podName
}

func runHistory(cmd *cobra.Command, args []string) {
	namespace := args[0]

	// Force initialize storage
	if err := initStorageForced(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx := context.Background()
	recommendations, err := store.ListRecommendations(ctx, namespace, historyLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(recommendations) == 0 {
		fmt.Printf("No recommendations found for namespace: %s\n", namespace)
		return
	}

	fmt.Printf("Recent recommendations for namespace '%s':\n\n", namespace)
	for i, rec := range recommendations {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, rec.Workload.Deployment, rec.ID)
		fmt.Printf("   Type: %s\n", rec.Type)
		fmt.Printf("   Savings: $%.2f/mo\n", rec.SavingsMonthly)
		fmt.Printf("   Status: %s\n", "pending")
		fmt.Printf("   Created: %s\n", rec.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

func runAudit(cmd *cobra.Command, args []string) {
	recommendationID := args[0]

	// Force initialize storage for audit command
	if err := initStorageForced(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx := context.Background()

	// Get recommendation details
	rec, err := store.GetRecommendation(ctx, recommendationID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Recommendation: %s\n", rec.ID)
	fmt.Printf("Deployment: %s (Namespace: %s)\n", rec.Workload.Deployment, rec.Workload.Namespace)
	fmt.Printf("Type: %s\n", rec.Type)
	fmt.Printf("Savings: $%.2f/mo\n", rec.SavingsMonthly)
	fmt.Printf("Created: %s\n\n", rec.CreatedAt.Format("2006-01-02 15:04:05"))

	// Get audit log
	auditLogs, err := store.GetAuditLog(ctx, recommendationID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(auditLogs) == 0 {
		fmt.Println("No audit log entries found")
		return
	}

	fmt.Println("Audit Log:")
	for i, log := range auditLogs {
		fmt.Printf("%d. %s - %s\n", i+1, log.Action, log.Status)
		fmt.Printf("   Executed: %s\n", log.ExecutedAt.Format("2006-01-02 15:04:05"))
		if log.ExecutedBy != "" {
			fmt.Printf("   By: %s\n", log.ExecutedBy)
		}
		if log.ErrorMessage != "" {
			fmt.Printf("   Error: %s\n", log.ErrorMessage)
		}
		fmt.Println()
	}
}

func outputText(recommendations []*models.Recommendation, totalSavings float64) {
	if len(recommendations) == 0 {
		fmt.Println("[INFO] No optimization opportunities found")
		return
	}

	fmt.Println("=== Optimization Recommendations ===\n")

	for i, rec := range recommendations {
		fmt.Printf("%d. %s/%s\n", i+1, rec.Workload.Namespace, rec.Workload.Deployment)
		fmt.Printf("   Type: %s\n", rec.Type)
		fmt.Printf("   Current:  CPU=%dm Memory=%dMi\n",
			rec.CurrentCPU, rec.CurrentMemory/(1024*1024))
		fmt.Printf("   Recommended: CPU=%dm Memory=%dMi\n",
			rec.RecommendedCPU, rec.RecommendedMemory/(1024*1024))
		fmt.Printf("   Savings: $%.2f/month\n", rec.SavingsMonthly)
		fmt.Printf("   Risk: %s\n", rec.Risk)
		fmt.Printf("   Command: %s\n", rec.Command)
		fmt.Println()
	}

	fmt.Printf("Total potential savings: $%.2f/month\n", totalSavings)
}

func outputJSON(recommendations []*models.Recommendation, totalSavings float64) {
	output := map[string]interface{}{
		"recommendations": recommendations,
		"total_savings":   totalSavings,
		"count":           len(recommendations),
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputCommands(recommendations []*recommender.Recommendation) {
	for _, rec := range recommendations {
		cmd := executor.GenerateCommand(rec)
		if cmd != "" {
			fmt.Println(cmd)
		}
	}
}
