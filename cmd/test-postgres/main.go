package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	"github.com/opscart/k8s-cost-optimizer/pkg/storage"
)

func main() {
	// Database connection string
	dsn := "host=localhost port=5432 user=costuser password=devpassword dbname=costoptimizer sslmode=disable"
	if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		dsn = envDSN
	}

	fmt.Println("[INFO] Connecting to PostgreSQL...")
	store, err := storage.NewPostgresStore(dsn)
	if err != nil {
		fmt.Printf("[ERROR] Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Test connection
	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		fmt.Printf("[ERROR] Ping failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Connected to PostgreSQL")

	// Test 1: Save a recommendation
	fmt.Println("\n[TEST 1] Saving recommendation...")
	rec := &models.Recommendation{
		Type: models.RecommendationRightSize,
		Workload: &models.Workload{
			ClusterID:  "test-cluster",
			Namespace:  "cost-test",
			Pod:        "idle-app-666df6866b-k8zlr",
			Deployment: "idle-app",
			Container:  "sleeper",
		},
		CurrentCPU:        500,
		CurrentMemory:     536870912, // 512Mi
		RecommendedCPU:    100,
		RecommendedMemory: 134217728, // 128Mi
		Reason:            "Pod is using only 20% of requested CPU",
		SavingsMonthly:    23.50,
		Impact:            "LOW",
		Risk:              models.RiskLow,
		Command:           "kubectl set resources deployment idle-app -n cost-test --requests=cpu=100m,memory=128Mi",
	}

	if err := store.SaveRecommendation(ctx, rec); err != nil {
		fmt.Printf("[ERROR] Save failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Saved recommendation: %s\n", rec.ID)

	// Test 2: Retrieve the recommendation
	fmt.Println("\n[TEST 2] Retrieving recommendation...")
	retrieved, err := store.GetRecommendation(ctx, rec.ID)
	if err != nil {
		fmt.Printf("[ERROR] Get failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Retrieved: %s (Pod: %s, Savings: $%.2f/mo)\n",
		retrieved.ID, retrieved.Workload.Pod, retrieved.SavingsMonthly)

	// Test 3: List recommendations by namespace
	fmt.Println("\n[TEST 3] Listing recommendations...")
	recommendations, err := store.ListRecommendations(ctx, "cost-test", 10)
	if err != nil {
		fmt.Printf("[ERROR] List failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Found %d recommendation(s) in cost-test namespace\n", len(recommendations))
	for i, r := range recommendations {
		fmt.Printf("  %d. %s - Save $%.2f/mo\n", i+1, r.Workload.Pod, r.SavingsMonthly)
	}

	// Test 4: Update recommendation (mark as applied)
	fmt.Println("\n[TEST 4] Updating recommendation (marking as applied)...")
	now := time.Now()
	rec.AppliedAt = &now
	rec.AppliedBy = "test-user"
	if err := store.UpdateRecommendation(ctx, rec); err != nil {
		fmt.Printf("[ERROR] Update failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Updated: marked as applied")

	// Test 5: Audit log
	fmt.Println("\n[TEST 5] Creating audit log entry...")
	auditEntry := &models.AuditEntry{
		RecommendationID: rec.ID,
		Action:           "APPLIED",
		Status:           "SUCCESS",
		ExecutedBy:       "test-user",
	}
	if err := store.LogAction(ctx, auditEntry); err != nil {
		fmt.Printf("[ERROR] Audit log failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Audit log entry created")

	// Test 6: Retrieve audit log
	fmt.Println("\n[TEST 6] Retrieving audit log...")
	auditLogs, err := store.GetAuditLog(ctx, rec.ID)
	if err != nil {
		fmt.Printf("[ERROR] Get audit log failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Found %d audit log entries\n", len(auditLogs))
	for i, log := range auditLogs {
		fmt.Printf("  %d. %s - %s by %s\n", i+1, log.Action, log.Status, log.ExecutedBy)
	}

	// Test 7: Cache metrics
	fmt.Println("\n[TEST 7] Caching metrics...")
	metrics := &models.Metrics{
		P95CPU:          100,
		P99CPU:          120,
		MaxCPU:          150,
		AvgCPU:          80,
		P95Memory:       134217728,
		P99Memory:       140000000,
		MaxMemory:       150000000,
		AvgMemory:       120000000,
		RequestedCPU:    500,
		RequestedMemory: 536870912,
		CollectedAt:     time.Now(),
		Duration:        7 * 24 * time.Hour,
		SampleCount:     10080,
	}
	if err := store.CacheMetrics(ctx, rec.Workload, metrics); err != nil {
		fmt.Printf("[ERROR] Cache metrics failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Metrics cached")

	// Test 8: Retrieve cached metrics
	fmt.Println("\n[TEST 8] Retrieving cached metrics...")
	cachedMetrics, err := store.GetCachedMetrics(ctx, rec.Workload)
	if err != nil {
		fmt.Printf("[ERROR] Get cached metrics failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Retrieved cached metrics: P95 CPU=%dm, P95 Memory=%dMi\n",
		cachedMetrics.P95CPU, cachedMetrics.P95Memory/(1024*1024))

	// Summary
	fmt.Println("\n" + "============================================================")
	fmt.Println("All tests passed!")
	fmt.Println("============================================================")
	fmt.Println("\nPostgreSQL Store is working correctly!")
	fmt.Println("  - Recommendations: Save, Get, List, Update [OK]")
	fmt.Println("  - Audit Log: Create, Retrieve [OK]")
	fmt.Println("  - Metrics Cache: Save, Retrieve [OK]")
}
