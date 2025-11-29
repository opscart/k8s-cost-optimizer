package reporter

import (
	"encoding/csv"
	"fmt"
	"io"
)

// GenerateCSV creates a CSV report
func GenerateCSV(report *Report, writer io.Writer) error {
	w := csv.NewWriter(writer)
	defer w.Flush()

	// Write header
	header := []string{
		"Namespace",
		"Workload",
		"Environment",
		"Type",
		"Current CPU (m)",
		"Current Memory (Mi)",
		"Recommended CPU (m)",
		"Recommended Memory (Mi)",
		"Monthly Savings ($)",
		"Risk",
		"Impact",
		"Reason",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write recommendations
	for _, rec := range report.Recommendations {
		row := []string{
			rec.Workload.Namespace,
			rec.Workload.Deployment,
			rec.Environment,
			string(rec.Type),
			fmt.Sprintf("%d", rec.CurrentCPU),
			fmt.Sprintf("%d", rec.CurrentMemory/(1024*1024)),
			fmt.Sprintf("%d", rec.RecommendedCPU),
			fmt.Sprintf("%d", rec.RecommendedMemory/(1024*1024)),
			fmt.Sprintf("%.2f", rec.SavingsMonthly),
			string(rec.Risk),
			rec.Impact,
			rec.Reason,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Write summary rows
	w.Write([]string{}) // Empty row
	w.Write([]string{"SUMMARY"})
	w.Write([]string{"Total Workloads", fmt.Sprintf("%d", report.WorkloadCount)})
	w.Write([]string{"Optimization Opportunities", fmt.Sprintf("%d", report.OptimizableCount)})
	w.Write([]string{"Total Monthly Savings", fmt.Sprintf("$%.2f", report.TotalSavings)})

	// Environment breakdown
	w.Write([]string{}) // Empty row
	w.Write([]string{"ENVIRONMENT BREAKDOWN"})
	w.Write([]string{"Environment", "Workloads", "Recommendations", "Savings"})
	for _, envStat := range report.EnvironmentStats {
		w.Write([]string{
			envStat.Environment,
			fmt.Sprintf("%d", envStat.WorkloadCount),
			fmt.Sprintf("%d", envStat.Recommendations),
			fmt.Sprintf("$%.2f", envStat.TotalSavings),
		})
	}

	return nil
}
