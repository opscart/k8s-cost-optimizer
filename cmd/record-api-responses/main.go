package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/opscart/k8s-cost-optimizer/pkg/pricing"
)

func main() {
	fmt.Println("Recording real API responses for contract testing...")
	
	ctx := context.Background()
	
	// Record Azure response
	fmt.Print("Recording Azure (eastus)... ")
	azureProvider := pricing.NewAzureProvider("eastus")
	azureCost, err := azureProvider.GetCostInfo(ctx, "eastus", "")
	if err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		data, _ := json.MarshalIndent(azureCost, "", "  ")
		os.WriteFile("testdata/pricing/azure_eastus.json", data, 0644)
		fmt.Println("✓ saved to testdata/pricing/azure_eastus.json")
	}
	
	// Record AWS response
	fmt.Print("Recording AWS (us-east-1)... ")
	awsProvider := pricing.NewAWSProvider("us-east-1")
	awsCost, err := awsProvider.GetCostInfo(ctx, "us-east-1", "")
	if err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		data, _ := json.MarshalIndent(awsCost, "", "  ")
		os.WriteFile("testdata/pricing/aws_us-east-1.json", data, 0644)
		fmt.Println("✓ saved to testdata/pricing/aws_us-east-1.json")
	}
	
	// Record GCP response
	fmt.Print("Recording GCP (us-central1)... ")
	gcpProvider := pricing.NewGCPProvider("us-central1")
	gcpCost, err := gcpProvider.GetCostInfo(ctx, "us-central1", "")
	if err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		data, _ := json.MarshalIndent(gcpCost, "", "  ")
		os.WriteFile("testdata/pricing/gcp_us-central1.json", data, 0644)
		fmt.Println("✓ saved to testdata/pricing/gcp_us-central1.json")
	}
	
	fmt.Println("\nRecordings complete! These will be used for fast contract tests.")
	fmt.Println("Re-run this command if cloud provider APIs change.")
}
