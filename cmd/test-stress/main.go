package main

import (
	"context"
	"fmt"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/datasource"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

func main() {
	source, _ := datasource.NewPrometheusSource("http://localhost:9090")
	
	workload := &models.Workload{
		Namespace: "cost-test",
		Pod:       "cpu-stress-test",
	}
	
	fmt.Println("Testing with CPU stress pod...")
	
	for i := 0; i < 10; i++ {
		metrics, err := source.GetMetrics(context.Background(), workload, 1*time.Hour)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		
		fmt.Printf("[%ds] CPU: Avg=%dm P95=%dm P99=%dm Max=%dm\n",
			i*5, metrics.AvgCPU, metrics.P95CPU, metrics.P99CPU, metrics.MaxCPU)
		
		if i < 9 {
			time.Sleep(5 * time.Second)
		}
	}
}
