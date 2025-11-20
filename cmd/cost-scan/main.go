package main

import (
	"fmt"
	"os"

	"github.com/opscart/k8s-cost-optimizer/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	namespace string
	allNamespaces bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cost-scan",
		Short: "Kubernetes cost optimization scanner",
		Long:  `Scan Kubernetes clusters for cost optimization opportunities and generate actionable recommendations.`,
		Run:   runScan,
	}

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scan (required unless --all-namespaces)")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Scan all namespaces")

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

	fmt.Println("[INFO] K8s Cost Optimizer - Starting scan")
	
	s, err := scanner.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing scanner: %v\n", err)
		os.Exit(1)
	}

	if err := s.Scan(namespace, allNamespaces); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Scan complete")
}
