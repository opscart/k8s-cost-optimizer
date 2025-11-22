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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	fmt.Println("=== Real Cluster Pricing Test ===\n")

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

	// Auto-detect cloud provider
	fmt.Println("[INFO] Detecting cloud provider from cluster...")
	detectedProvider, detectedRegion, err := pricing.DetectProvider(ctx, clientset)
	if err != nil {
		fmt.Printf("[WARN] Detection failed: %v\n", err)
	}
	fmt.Printf("[INFO] Detected: %s (region: %s)\n\n", detectedProvider, detectedRegion)

	// Get real pods from cost-test namespace
	fmt.Println("[INFO] Getting real pods from cost-test namespace...")
	pods, err := clientset.CoreV1().Pods("cost-test").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[INFO] Found %d pods in cost-test\n\n", len(pods.Items))

	if len(pods.Items) == 0 {
		fmt.Println("[WARN] No pods found. Deploy test workloads first:")
		fmt.Println("  kubectl apply -f examples/test-workloads/")
		return
	}

	// Compare costs for each pod
	for _, pod := range pods.Items {
		if len(pod.Spec.Containers) == 0 {
			continue
		}
		
		fmt.Printf("Pod: %s\n", pod.Name)
		
		container := pod.Spec.Containers[0]
		cpuReq := container.Resources.Requests.Cpu().MilliValue()
		memReq := container.Resources.Requests.Memory().Value()
		
		fmt.Printf("  Requested: %dm CPU, %dMi memory\n", 
			cpuReq, memReq/(1024*1024))
		
		// Compare costs across providers
		providers := []struct {
			name string
			provider pricing.Provider
		}{
			{"Default", pricing.NewDefaultProvider(23.0, 3.0)},
			{"Azure", pricing.NewAzureProvider("eastus")},
			{"AWS", pricing.NewAWSProvider("us-east-1")},
			{"GCP", pricing.NewGCPProvider("us-central1")},
		}
		
		fmt.Println("\n  Monthly cost across clouds:")
		fmt.Printf("  %-10s %-15s\n", "Provider", "Cost/month")
		fmt.Println("  " + string(make([]byte, 30)))
		
		for _, p := range providers {
			costInfo, _ := p.provider.GetCostInfo(ctx, "", "")
			
			cpuCores := float64(cpuReq) / 1000.0
			memGiB := float64(memReq) / (1024.0 * 1024.0 * 1024.0)
			
			cost := (cpuCores * costInfo.CPUCostPerCore) + (memGiB * costInfo.MemoryCostPerGiB)
			
			fmt.Printf("  %-10s $%-14.2f\n", p.name, cost)
		}
		fmt.Println()
	}
	
	fmt.Println("=" + string(make([]byte, 50)))
	fmt.Println("[KEY INSIGHT] These are REAL pricing rates from cloud providers")
	fmt.Println("[KEY INSIGHT] Your pods have REAL resource requests from the cluster")
	fmt.Println("[KEY INSIGHT] Costs vary significantly across clouds!")
	fmt.Println("\nNext step: Add actual usage metrics for smarter recommendations")
}
