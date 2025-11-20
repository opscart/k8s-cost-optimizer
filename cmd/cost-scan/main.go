package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opscart/k8s-cost-optimizer/pkg/executor"
	"github.com/opscart/k8s-cost-optimizer/pkg/recommender"
	"github.com/opscart/k8s-cost-optimizer/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	namespace     string
	allNamespaces bool
	outputFormat  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cost-scan",
		Short: "Kubernetes cost optimization scanner",
		Long:  `Scan Kubernetes clusters for cost optimization opportunities and generate actionable recommendations.`,
		Run:   runScan,
	}

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scan")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Scan all namespaces")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, commands")

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

	if outputFormat != "text" && outputFormat != "json" && outputFormat != "commands" {
		fmt.Fprintln(os.Stderr, "Error: output must be text, json, or commands")
		os.Exit(1)
	}

	if outputFormat != "commands" {
		fmt.Println("[INFO] K8s Cost Optimizer - Starting scan")
	}
	
	s, err := scanner.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing scanner: %v\n", err)
		os.Exit(1)
	}

	recommendations, err := s.ScanAndRecommend(namespace, allNamespaces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning cluster: %v\n", err)
		os.Exit(1)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		outputJSON(recommendations)
	case "commands":
		outputCommands(recommendations)
	default:
		outputText(recommendations)
	}

	if outputFormat != "commands" {
		fmt.Println("[INFO] Scan complete")
	}
}

func outputText(recommendations []*recommender.Recommendation) {
	fmt.Println("\n================================================================================")
	fmt.Println("OPTIMIZATION OPPORTUNITIES\n")
	
	totalSavings := 0.0
	actionableCount := 0
	
	for _, rec := range recommendations {
		if rec.Type != recommender.NoAction {
			fmt.Println(rec.String())
			fmt.Println()
			totalSavings += rec.Savings
			actionableCount++
		}
	}

	if actionableCount == 0 {
		fmt.Println("No optimization opportunities found.")
	} else {
		fmt.Printf("Found %d optimization opportunities\n", actionableCount)
		fmt.Printf("Total potential savings: $%.2f/month\n\n", totalSavings)
		fmt.Println("To see kubectl commands, run with: --output commands")
	}
}

func outputJSON(recommendations []*recommender.Recommendation) {
	data, err := json.MarshalIndent(recommendations, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func outputCommands(recommendations []*recommender.Recommendation) {
	script := executor.GenerateScript(recommendations)
	fmt.Print(script)
}
