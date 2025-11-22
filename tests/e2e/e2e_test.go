//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func getKubernetesClient(t *testing.T) *kubernetes.Clientset {
	t.Helper()
	
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create clientset: %v", err)
	}
	
	return clientset
}

func TestRealClusterConnection(t *testing.T) {
	clientset := getKubernetesClient(t)
	
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	
	if len(nodes.Items) == 0 {
		t.Fatal("No nodes found in cluster")
	}
	
	t.Logf("✓ Connected to cluster with %d node(s)", len(nodes.Items))
	for _, node := range nodes.Items {
		t.Logf("  Node: %s", node.Name)
	}
}

func TestCostTestNamespace(t *testing.T) {
	clientset := getKubernetesClient(t)
	
	ctx := context.Background()
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, "cost-test", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("cost-test namespace not found: %v\nRun: kubectl apply -f examples/test-workloads/", err)
	}
	
	t.Logf("✓ Found namespace: %s", ns.Name)
}

func TestRealPods(t *testing.T) {
	clientset := getKubernetesClient(t)
	
	ctx := context.Background()
	pods, err := clientset.CoreV1().Pods("cost-test").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}
	
	if len(pods.Items) == 0 {
		t.Fatal("No pods found. Deploy: kubectl apply -f examples/test-workloads/")
	}
	
	t.Logf("✓ Found %d real pods:", len(pods.Items))
	for _, pod := range pods.Items {
		t.Logf("  - %s (Phase: %s)", pod.Name, pod.Status.Phase)
	}
}

func TestCostScanCLIExecution(t *testing.T) {
	// Build CLI
	t.Log("Building cost-scan...")
	build := exec.Command("go", "build", "-o", "../../bin/cost-scan", "../../cmd/cost-scan")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\n%s", err, output)
	}
	t.Log("✓ Built CLI")
	
	// Run against REAL cluster
	t.Log("Running cost-scan against REAL cluster...")
	cmd := exec.Command("../../bin/cost-scan", "-n", "cost-test")
	output, err := cmd.CombinedOutput()
	
	outputStr := string(output)
	t.Logf("Output:\n%s", outputStr)
	
	if err != nil {
		t.Fatalf("CLI failed: %v", err)
	}
	
	// Verify it found real pods
	if !strings.Contains(outputStr, "cost-test") {
		t.Error("Output should mention cost-test namespace")
	}
	
	t.Log("✓ Successfully scanned real cluster!")
}
