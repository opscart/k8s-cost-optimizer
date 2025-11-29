package reporter

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>K8s Cost Optimizer Report - {{.ClusterName}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f5f7fa;
            color: #333;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #326ce5 0%, #1a4d8f 100%);
            color: white;
            padding: 50px 40px;
            position: relative;
            overflow: hidden;
        }
        .header::before {
            content: '';
            position: absolute;
            top: -50%;
            right: -10%;
            width: 500px;
            height: 500px;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 50%;
        }
        .header h1 {
            font-size: 2.8em;
            margin-bottom: 15px;
            position: relative;
            z-index: 1;
        }
        .header .meta {
            opacity: 0.95;
            font-size: 1.1em;
            position: relative;
            z-index: 1;
        }
        .header .meta strong {
            color: #fff;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 25px;
            padding: 40px;
            background: linear-gradient(to bottom, #f8f9fa 0%, #fff 100%);
        }
        .summary-card {
            background: white;
            padding: 30px;
            border-radius: 12px;
            border: 2px solid #e8eaed;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .summary-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 8px 20px rgba(0, 0, 0, 0.1);
        }
        .summary-card h3 {
            color: #5f6368;
            font-size: 0.85em;
            text-transform: uppercase;
            letter-spacing: 1.5px;
            margin-bottom: 15px;
            font-weight: 600;
        }
        .summary-card .value {
            font-size: 3em;
            font-weight: 700;
            color: #202124;
            line-height: 1;
        }
        .summary-card.savings {
            border-left: 6px solid #34a853;
        }
        .summary-card.savings .value {
            color: #34a853;
        }
        .summary-card.workloads {
            border-left: 6px solid #326ce5;
        }
        .summary-card.workloads .value {
            color: #326ce5;
        }
        .summary-card.opportunities {
            border-left: 6px solid #fbbc04;
        }
        .summary-card.opportunities .value {
            color: #fbbc04;
        }
        .section {
            padding: 50px 40px;
        }
        .section:nth-child(even) {
            background: #fafbfc;
        }
        .section h2 {
            font-size: 2em;
            margin-bottom: 30px;
            color: #202124;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .section h2::before {
            content: '';
            width: 5px;
            height: 40px;
            background: #326ce5;
            border-radius: 3px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 25px;
            margin-top: 25px;
        }
        .stat-card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            border: 1px solid #e8eaed;
            box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
        }
        .stat-card h4 {
            color: #202124;
            font-size: 1.3em;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .stat-row {
            display: flex;
            justify-content: space-between;
            padding: 12px 0;
            border-bottom: 1px solid #f0f2f4;
        }
        .stat-row:last-child {
            border-bottom: none;
        }
        .stat-label {
            color: #5f6368;
            font-weight: 500;
        }
        .stat-value {
            font-weight: 700;
            color: #202124;
            font-size: 1.1em;
        }
        .recommendations-table {
            width: 100%;
            border-collapse: separate;
            border-spacing: 0;
            margin-top: 25px;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
        }
        .recommendations-table th {
            background: #326ce5;
            color: white;
            padding: 18px 15px;
            text-align: left;
            font-weight: 600;
            font-size: 0.95em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .recommendations-table td {
            padding: 18px 15px;
            border-bottom: 1px solid #f0f2f4;
        }
        .recommendations-table tbody tr {
            transition: background-color 0.2s;
        }
        .recommendations-table tbody tr:hover {
            background: #f8f9fa;
        }
        .recommendations-table tbody tr:last-child td {
            border-bottom: none;
        }
        .type-badge {
            padding: 7px 14px;
            border-radius: 6px;
            font-size: 0.8em;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            display: inline-block;
        }
        .type-right_size {
            background: #e8f0fe;
            color: #1a73e8;
        }
        .type-scale_down {
            background: #fef7e0;
            color: #f9ab00;
        }
        .type-no_action {
            background: #f1f3f4;
            color: #5f6368;
        }
        .risk-badge {
            padding: 6px 12px;
            border-radius: 6px;
            font-size: 0.75em;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            display: inline-block;
        }
        .risk-low {
            background: #e6f4ea;
            color: #1e8e3e;
        }
        .risk-medium {
            background: #fef7e0;
            color: #f9ab00;
        }
        .risk-high {
            background: #fce8e6;
            color: #d93025;
        }
        .risk-none {
            background: #f1f3f4;
            color: #5f6368;
        }
        .env-badge {
            padding: 5px 12px;
            border-radius: 6px;
            font-size: 0.7em;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 1px;
            display: inline-block;
        }
        .env-production {
            background: #fce8e6;
            color: #d93025;
            border: 1px solid #d93025;
        }
        .env-staging {
            background: #fef7e0;
            color: #f9ab00;
            border: 1px solid #f9ab00;
        }
        .env-development {
            background: #e6f4ea;
            color: #1e8e3e;
            border: 1px solid #1e8e3e;
        }
        .env-unknown {
            background: #f1f3f4;
            color: #5f6368;
            border: 1px solid #9aa0a6;
        }
        .footer {
            background: #202124;
            color: #9aa0a6;
            padding: 40px;
            text-align: center;
        }
        .footer strong {
            color: #fff;
        }
        .footer a {
            color: #8ab4f8;
            text-decoration: none;
            transition: color 0.2s;
        }
        .footer a:hover {
            color: #aecbfa;
        }
        .k8s-logo {
            font-size: 2em;
            margin-right: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <div class="header">
            <h1><span class="k8s-logo">⎈</span>K8s Cost Optimizer Report</h1>
            <div class="meta">
                <p><strong>Cluster:</strong> {{.ClusterName}} | <strong>Namespace:</strong> {{if .Namespace}}{{.Namespace}}{{else}}All Namespaces{{end}}</p>
                <p><strong>Generated:</strong> {{.GeneratedAt.Format "January 2, 2006 15:04:05 MST"}}</p>
            </div>
        </div>

        <!-- Executive Summary -->
        <div class="summary">
            <div class="summary-card savings">
                <h3>Total Monthly Savings</h3>
                <div class="value">${{printf "%.2f" .TotalSavings}}</div>
            </div>
            <div class="summary-card workloads">
                <h3>Workloads Analyzed</h3>
                <div class="value">{{.WorkloadCount}}</div>
            </div>
            <div class="summary-card opportunities">
                <h3>Optimization Opportunities</h3>
                <div class="value">{{.OptimizableCount}}</div>
            </div>
        </div>

        <!-- Environment Breakdown -->
        {{if .EnvironmentStats}}
        <div class="section">
            <h2>By Environment</h2>
            <div class="stats-grid">
                {{range .EnvironmentStats}}
                <div class="stat-card">
                    <h4>
                        <span class="env-badge env-{{.Environment}}">{{.Environment}}</span>
                    </h4>
                    <div class="stat-row">
                        <span class="stat-label">Workloads</span>
                        <span class="stat-value">{{.WorkloadCount}}</span>
                    </div>
                    <div class="stat-row">
                        <span class="stat-label">Recommendations</span>
                        <span class="stat-value">{{.Recommendations}}</span>
                    </div>
                    <div class="stat-row">
                        <span class="stat-label">Monthly Savings</span>
                        <span class="stat-value">${{printf "%.2f" .TotalSavings}}</span>
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <!-- Recommendations Table -->
        <div class="section">
            <h2>Detailed Recommendations</h2>
            <table class="recommendations-table">
                <thead>
                    <tr>
                        <th>Workload</th>
                        <th>Environment</th>
                        <th>Type</th>
                        <th>Current Resources</th>
                        <th>Recommended</th>
                        <th>Savings/Month</th>
                        <th>Risk</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Recommendations}}
                    <tr>
                        <td>
                            <strong>{{.Workload.Namespace}}/{{.Workload.Deployment}}</strong>
                        </td>
                        <td>
                            <span class="env-badge env-{{.Environment}}">{{.Environment}}</span>
                        </td>
                        <td>
                            <span class="type-badge type-{{.Type | lower}}">{{.Type}}</span>
                        </td>
                        <td>
                            {{.CurrentCPU}}m CPU<br>
                            {{div .CurrentMemory 1048576}}Mi RAM
                        </td>
                        <td>
                            {{.RecommendedCPU}}m CPU<br>
                            {{div .RecommendedMemory 1048576}}Mi RAM
                        </td>
                        <td>
                            <strong style="color: #34a853;">${{printf "%.2f" .SavingsMonthly}}</strong>
                        </td>
                        <td>
                            <span class="risk-badge risk-{{.Risk | lower}}">{{.Risk}}</span>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>

        <!-- Footer -->
        <div class="footer">
            <p>Generated by <strong>k8s-cost-optimizer</strong></p>
            <p><a href="https://github.com/opscart/k8s-cost-optimizer" target="_blank">⭐ Star on GitHub</a></p>
        </div>
    </div>
</body>
</html>
`

// GenerateHTML creates an HTML report
func GenerateHTML(report *Report, writer io.Writer) error {
	// Parse template
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"lower": func(s interface{}) string {
			return strings.ToLower(fmt.Sprintf("%v", s))
		},
		"div": func(a, b int64) int64 { return a / b },
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	if err := tmpl.Execute(writer, report); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
