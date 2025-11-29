package reporter

import (
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// ReportFormat represents the output format
type ReportFormat string

const (
	FormatHTML     ReportFormat = "html"
	FormatMarkdown ReportFormat = "markdown"
	FormatCSV      ReportFormat = "csv"
)

// Report contains all data for generating reports
type Report struct {
	ClusterName       string
	Namespace         string
	GeneratedAt       time.Time
	Recommendations   []*models.Recommendation
	TotalSavings      float64
	WorkloadCount     int
	OptimizableCount  int
	EnvironmentStats  map[string]*EnvironmentStats
	WorkloadTypeStats map[string]*WorkloadTypeStats
}

// EnvironmentStats holds statistics per environment
type EnvironmentStats struct {
	Environment     string
	WorkloadCount   int
	TotalSavings    float64
	Recommendations int
	AvgSafetyBuffer float64
}

// WorkloadTypeStats holds statistics per workload type
type WorkloadTypeStats struct {
	WorkloadType     string
	Count            int
	TotalSavings     float64
	Recommendations  int
	OptimizationRate float64 // Percentage of workloads optimizable
}

// Reporter generates cost optimization reports
type Reporter struct {
	format ReportFormat
}

// New creates a new reporter
func New(format ReportFormat) *Reporter {
	return &Reporter{
		format: format,
	}
}

// Generate generates a report from recommendations
func (r *Reporter) Generate(recommendations []*models.Recommendation, clusterName, namespace string) (*Report, error) {
	report := &Report{
		ClusterName:       clusterName,
		Namespace:         namespace,
		GeneratedAt:       time.Now(),
		Recommendations:   recommendations,
		EnvironmentStats:  make(map[string]*EnvironmentStats),
		WorkloadTypeStats: make(map[string]*WorkloadTypeStats),
	}

	// Calculate statistics
	r.calculateStats(report)

	return report, nil
}

// calculateStats computes all statistics for the report
func (r *Reporter) calculateStats(report *Report) {
	for _, rec := range report.Recommendations {
		report.WorkloadCount++
		report.TotalSavings += rec.SavingsMonthly

		// Count optimizable workloads (not NO_ACTION)
		if rec.Type != models.RecommendationNoAction {
			report.OptimizableCount++
		}

		// Environment stats
		env := rec.Environment
		if env == "" {
			env = "unknown"
		}
		if _, exists := report.EnvironmentStats[env]; !exists {
			report.EnvironmentStats[env] = &EnvironmentStats{
				Environment: env,
			}
		}
		envStat := report.EnvironmentStats[env]
		envStat.WorkloadCount++
		envStat.TotalSavings += rec.SavingsMonthly
		if rec.Type != models.RecommendationNoAction {
			envStat.Recommendations++
		}

		// Workload type stats
		workloadType := rec.Workload.Deployment // This contains workload name, need type
		// For now, we'll track by deployment name
		// TODO: Add WorkloadType to models.Recommendation if not present
		if _, exists := report.WorkloadTypeStats[workloadType]; !exists {
			report.WorkloadTypeStats[workloadType] = &WorkloadTypeStats{
				WorkloadType: workloadType,
			}
		}
		wtStat := report.WorkloadTypeStats[workloadType]
		wtStat.Count++
		wtStat.TotalSavings += rec.SavingsMonthly
		if rec.Type != models.RecommendationNoAction {
			wtStat.Recommendations++
		}
	}

	// Calculate optimization rates
	for _, stat := range report.WorkloadTypeStats {
		if stat.Count > 0 {
			stat.OptimizationRate = float64(stat.Recommendations) / float64(stat.Count) * 100
		}
	}
}
