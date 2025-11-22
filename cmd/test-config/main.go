package main

import (
	"fmt"
	"os"

	"github.com/opscart/k8s-cost-optimizer/pkg/config"
)

func main() {
	fmt.Println("=== Configuration Testing ===\n")

	// Test 1: Default configuration
	fmt.Println("[TEST 1] Default Configuration")
	cfg := config.NewConfig()
	fmt.Printf("  Lookback Days: %d\n", cfg.MetricsLookbackDays)
	fmt.Printf("  Duration: %v\n", cfg.MetricsDuration)
	fmt.Printf("  Safety Buffer: %.1fx\n", cfg.SafetyBuffer)
	fmt.Println()

	// Test 2: Environment variable override
	fmt.Println("[TEST 2] Environment Variable Override")
	os.Setenv("METRICS_LOOKBACK_DAYS", "15")
	os.Setenv("SAFETY_BUFFER", "2.0")
	cfg2 := config.NewConfig()
	fmt.Printf("  Lookback Days: %d (changed from 7 to 15)\n", cfg2.MetricsLookbackDays)
	fmt.Printf("  Duration: %v\n", cfg2.MetricsDuration)
	fmt.Printf("  Safety Buffer: %.1fx (changed from 1.5x to 2.0x)\n", cfg2.SafetyBuffer)
	fmt.Println()

	// Test 3: Presets
	fmt.Println("[TEST 3] Configuration Presets")
	
	devCfg := config.NewConfig()
	devCfg.UseDevPreset()
	fmt.Printf("  Dev Preset: %d days, %.1fx buffer (fast iteration)\n", 
		devCfg.MetricsLookbackDays, devCfg.SafetyBuffer)
	
	prodCfg := config.NewConfig()
	prodCfg.UseProductionPreset()
	fmt.Printf("  Production Preset: %d days, %.1fx buffer (balanced)\n", 
		prodCfg.MetricsLookbackDays, prodCfg.SafetyBuffer)
	
	criticalCfg := config.NewConfig()
	criticalCfg.UseCriticalPreset()
	fmt.Printf("  Critical Preset: %d days, %.1fx buffer (very safe)\n", 
		criticalCfg.MetricsLookbackDays, criticalCfg.SafetyBuffer)
	fmt.Println()

	// Test 4: Validation
	fmt.Println("[TEST 4] Configuration Validation")
	
	validCfg := config.NewConfig()
	if err := validCfg.Validate(); err != nil {
		fmt.Printf("  Valid config failed: %v\n", err)
	} else {
		fmt.Println("  Valid config passed")
	}
	
	invalidCfg := config.NewConfig()
	invalidCfg.MetricsLookbackDays = 100 // Too high
	if err := invalidCfg.Validate(); err != nil {
		fmt.Printf("  Invalid config caught: %v\n", err)
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Configuration Options ===")
	fmt.Println("\n1. Environment Variables:")
	fmt.Println("   export METRICS_LOOKBACK_DAYS=15")
	fmt.Println("   export SAFETY_BUFFER=2.0")
	fmt.Println("\n2. CLI Flags (coming next):")
	fmt.Println("   cost-scan --lookback=15d --buffer=2.0")
	fmt.Println("\n3. Presets:")
	fmt.Println("   cost-scan --preset=production")
	fmt.Println("\nEasy to change: Just update METRICS_LOOKBACK_DAYS from 7 to 15!")
}
