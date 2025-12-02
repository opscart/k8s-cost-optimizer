# k8s-cost-optimizer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-94%25-brightgreen)](https://github.com/opscart/k8s-cost-optimizer)

A production-ready Kubernetes cost optimization tool that uses historical P95/P99 analysis, workload-aware intelligence, and environment-based safety buffers to provide accurate, safe recommendations.

## Key Differentiators

### Historical Trend Analysis
Unlike tools that use instant snapshots, k8s-cost-optimizer analyzes 7-day P95/P99 trends from Prometheus (1400+ data points per workload) to make informed recommendations based on actual usage patterns.

### Workload-Aware Optimization
Different workload types require different safety margins:
- **Deployments**: 1.5x safety buffer (stateless, can tolerate restarts)
- **StatefulSets**: 2.0x safety buffer (stateful data, requires extra caution)
- **DaemonSets**: Optimization disabled (node-critical services)

### Environment-Based Safety
Production workloads get additional safety buffers:
- **Production**: 1.3x additional multiplier (2.6x total for StatefulSets)
- **Staging**: 1.0x standard multiplier
- **Development**: 0.85x aggressive optimization (15% more savings)

## Features

### Core Analysis
- **Historical P95/P99 Analysis** - 7-day Prometheus lookback with 1400+ samples per workload
- **Workload Type Detection** - Automatic classification of Deployments, StatefulSets, DaemonSets
- **Environment Classification** - Label-based and name-pattern detection (production/staging/development)
- **HPA Detection** - Automatically skips auto-scaling workloads
- **Multi-Cloud Pricing** - Azure, AWS/GCP (static estimates)
- **Graceful Fallback** - Uses instant metrics when historical data unavailable

### Reporting
- **HTML Executive Reports** - CNCF-themed dashboards with environment breakdown
- **CSV Exports** - Finance-friendly format for spreadsheet analysis
- **Markdown Reports** - GitHub/wiki compatible documentation
- **Top Savings Analysis** - Ranked opportunities by potential savings

### Production Features
- **Verbose Mode** - Debug logging with `-v` flag
- **Clean Output** - Production-ready formatting
- **Timestamped Reports** - Automatic file organization in `reports/` directory
- **Sample Count Tracking** - Data quality indicators for recommendations

## Quick Start

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Go 1.21+ (for building from source)
- Prometheus (recommended for historical analysis) OR metrics-server (fallback)

### Installation
```bash
# Clone repository
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer

# Build
go build -o bin/k8s-cost-optimizer cmd/cost-scan/main.go

# Run scan
./bin/k8s-cost-optimizer -n production
```

### Sample Output
```
[INFO] K8s Cost Optimizer - Starting scan
[INFO] Using Prometheus at http://localhost:9090
[INFO] Connected to cluster (version: v1.31.0)
[INFO] Scanning namespace: production
[INFO] Using historical analysis (P95 over 7 days)
[INFO] Using 7-day historical analysis for production/api-server (1409 CPU samples, 1410 memory samples)

=== Optimization Recommendations ===

1. production/api-server [PRODUCTION]
   Type: RIGHT_SIZE
   Reason: CPU over-provisioned by 65% - Workload: Deployment, Environment: production 
           (Based on 7-day P95: CPU 350m, Memory 512Mi)
   Current: CPU=1000m Memory=2048Mi
   Recommended: CPU=682m Memory=998Mi
   Savings: $27.19/month
   Risk: LOW
   Command: kubectl set resources deployment api-server -n production \
     --requests=cpu=682m,memory=998Mi

2. production/postgres [PRODUCTION]
   Type: NO_ACTION
   Reason: Workload appears well-sized - Workload: StatefulSet, Environment: production 
           (Based on 7-day P95: CPU 180m, Memory 768Mi)
   Current: CPU=500m Memory=2048Mi
   Recommended: CPU=468m Memory=1997Mi
   Risk: MEDIUM

Total potential savings: $152.45/month
```

## Usage

### Basic Scan
```bash
# Scan single namespace
./bin/k8s-cost-optimizer -n production

# Scan all namespaces
./bin/k8s-cost-optimizer --all-namespaces

# Verbose mode (debug output)
./bin/k8s-cost-optimizer -n production -v
```

### Report Generation
```bash
# Generate HTML report
./bin/k8s-cost-optimizer -n production --generate-report

# Generate CSV report
./bin/k8s-cost-optimizer -n production --generate-report --report-format csv --report-output costs.csv

# Generate Markdown report
./bin/k8s-cost-optimizer -n production --generate-report --report-format markdown

# Custom output location
./bin/k8s-cost-optimizer -n production --generate-report --report-output /path/to/report.html
```

Reports are automatically organized in `reports/` directory with timestamps:
```
reports/
├── cost-report-production-20251201-143022.html
├── cost-report-production-20251201-143022.csv
└── cost-report-production-20251201-143022.md
```

### Advanced Options
```bash
# Configure analysis parameters
export METRICS_LOOKBACK_DAYS=7        # Historical lookback period
export SAFETY_BUFFER=1.5              # Base safety multiplier
export PROMETHEUS_URL=http://prometheus:9090

./bin/k8s-cost-optimizer -n production

# Specify Prometheus URL directly
./bin/k8s-cost-optimizer -n production --prometheus-url http://prometheus:9090

# Set custom lookback period
export METRICS_LOOKBACK_DAYS=14
./bin/k8s-cost-optimizer -n production
```

## Architecture
```
┌───────────────────────────────────────────────────────────────┐
│              k8s-cost-optimizer                               │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────┐         ┌──────────────┐                    │
│  │   Scanner   │────────>│   Analyzer   │                    │
│  │             │         │              │                    │
│  │ - Multi-    │         │ - Historical │                    │
│  │   workload  │         │   Analysis   │                    │
│  │ - HPA       │         │ - P95/P99    │                    │
│  │   detection │         │ - Percentiles│                    │
│  └─────────────┘         └──────────────┘                    │
│         │                        │                           │
│         │                        │                           │
│         v                        v                           │
│  ┌─────────────┐         ┌──────────────┐                    │
│  │ Kubernetes  │         │   Metrics    │                    │
│  │     API     │         │   Sources    │                    │
│  │             │         │              │                    │
│  │  - Nodes    │         │ ┌──────────┐ │                    │
│  │  - Pods     │         │ │Prometheus│ │ <- 7-day P95/P99  │
│  │  - Deploys  │         │ │  (1400+  │ │    historical     │
│  │  - StatefulS│         │ │ samples) │ │    analysis       │
│  │  - DaemonSet│         │ └──────────┘ │                    │
│  │  - HPAs     │         │ ┌──────────┐ │                    │
│  └─────────────┘         │ │ metrics- │ │ <- Instant        │
│         │                │ │  server  │ │    fallback       │
│         │                │ └──────────┘ │                    │
│         │                └──────────────┘                    │
│         │                        │                           │
│         └────────┬───────────────┘                           │
│                  v                                           │
│          ┌──────────────┐                                    │
│          │ Recommender  │                                    │
│          │              │                                    │
│          │ - Workload   │                                    │
│          │   awareness  │                                    │
│          │ - Environment│                                    │
│          │   safety     │                                    │
│          └──────────────┘                                    │
│                  │                                           │
│          ┌───────┴────────┐                                  │
│          v                v                                  │
│   ┌─────────────┐  ┌──────────────┐                          │
│   │   Pricing   │  │   Reporter   │                          │
│   │  Providers  │  │              │                          │
│   │             │  │ - HTML       │                          │
│   │ Azure/AWS/  │  │ - CSV        │                          │
│   │ GCP/Default │  │ - Markdown   │                          │
│   └─────────────┘  └──────────────┘                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## How It Works

### 1. Historical Analysis
```
For each workload:
1. Query Prometheus for 7-day metrics (5-minute intervals)
2. Collect 1400+ CPU and memory samples
3. Calculate P50, P90, P95, P99 percentiles
4. Convert counter to rate for accurate CPU usage
5. Use P95 as baseline for recommendations
```

### 2. Workload-Aware Safety Buffers
```
Base safety buffer by workload type:
- Deployment:  1.5x (stateless, low risk)
- StatefulSet: 2.0x (stateful data, medium risk)
- DaemonSet:   Optimization disabled (critical services)
- Job/CronJob: 1.2x (batch workloads, low risk)
```

### 3. Environment-Based Multipliers
```
Additional safety based on environment:
- Production:  1.3x extra (conservative)
- Staging:     1.0x standard
- Development: 0.85x aggressive (15% more savings)

Example: StatefulSet in production = 2.0 × 1.3 = 2.6x total buffer
```

### 4. Combined Calculation
```
Recommended CPU = P95 × workload_buffer × environment_multiplier
Recommended Memory = P95 × workload_buffer × environment_multiplier

Example for StatefulSet in production:
P95 CPU: 200m
Workload buffer: 2.0x
Environment multiplier: 1.3x
Recommended: 200m × 2.0 × 1.3 = 520m
```

## Testing

### Test Coverage: 94%
```bash
# Unit tests (fast, mocked)
go test ./... -cover

# Integration tests (real APIs)
go test -tags=integration ./pkg/pricing -v

# E2E tests (real cluster)
go test -tags=e2e ./tests/e2e -v
```

## Configuration

### Environment Variables
```bash
# Historical analysis lookback period
METRICS_LOOKBACK_DAYS=7    # Options: 1-30 days

# Base safety buffer multiplier
SAFETY_BUFFER=1.5          # Options: 1.0-3.0

# Prometheus URL
PROMETHEUS_URL=http://localhost:9090

# Cluster identification
CLUSTER_ID=production-cluster

# Storage (optional, for future premium features)
STORAGE_ENABLED=false
DATABASE_URL=postgres://user:pass@localhost/costdb
```

### Setup Local Environment

The project includes comprehensive setup scripts:
```bash
# 1. Create local Minikube cluster with Prometheus
./scripts/setup/setup-local-env.sh

# 2. Deploy basic test workloads
./scripts/setup/deploy-test-workloads.sh

# 3. Deploy advanced workloads (StatefulSets, DaemonSets, HPAs)
./scripts/setup/deploy-advanced-workloads.sh

# 4. Deploy multi-environment workloads (prod/staging/dev namespaces)
./scripts/setup/deploy-multi-env-workloads.sh

# 5. Port-forward Prometheus for local testing
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
```

## Project Structure
```
k8s-cost-optimizer/
├── cmd/
│   └── cost-scan/              # Main CLI application
├── pkg/
│   ├── analyzer/
│   │   ├── analyzer.go         # Pod analysis orchestration
│   │   ├── historical.go       # Prometheus historical queries
│   │   ├── percentile.go       # P95/P99 calculations
│   │   ├── workload_config.go  # Workload-specific settings
│   │   └── namespace_classifier.go  # Environment detection
│   ├── scanner/
│   │   └── scanner.go          # Multi-workload type scanning
│   ├── recommender/
│   │   └── recommender.go      # Optimization logic
│   ├── reporter/
│   │   ├── reporter.go         # Report orchestration
│   │   ├── html.go             # HTML report generation
│   │   ├── csv.go              # CSV export
│   │   └── markdown.go         # Markdown reports
│   ├── pricing/                # Multi-cloud pricing
│   ├── datasource/             # Prometheus integration
│   ├── storage/                # PostgreSQL (future premium)
│   └── models/                 # Data structures
├── scripts/
│   └── setup/                  # Environment setup scripts
├── reports/                    # Generated reports (gitignored)
├── tests/
│   └── e2e/                    # End-to-end tests
└── docs/
    ├── WEEK1_SUMMARY.md        # Development logs
    ├── WEEK2_SUMMARY.md
    └── WEEK3_SUMMARY.md
```

## Cloud Pricing

### Supported Providers

| Provider | Status | Implementation |
|----------|--------|----------------|
| Azure    | Production | Real-time Azure Retail Prices API |
| AWS      | Beta | Static defaults (API integration planned) |
| GCP      | Beta | Static defaults (API integration planned) |
| Default  | Production | Conservative industry estimates |

## Comparison with Other Tools

| Feature | k8s-cost-optimizer | Kubecost | Goldilocks | KRR |
|---------|-------------------|----------|------------|-----|
| Historical P95/P99 | Yes (7-day, 1400+ samples) | Basic | No | No |
| Workload-Aware Safety | Yes (per-type buffers) | No | No | No |
| Environment Classification | Yes (prod/staging/dev) | No | No | No |
| HPA Detection | Yes (auto-skip) | Yes | Yes | No |
| Professional Reports | Yes (HTML/CSV/MD) | Yes | No | No |
| Multi-Cloud Pricing | Yes (Azure/AWS/GCP) | Yes | No | No |
| Open Source | Yes | Partial | Yes | Yes |

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

**Shamsher Khan**
- Senior DevOps Engineer
- IEEE Senior Member
- Technical Writer: DZone, Medium, InfoQ

## Acknowledgments

- Inspired by Kubecost, Goldilocks, and KRR
- Kubernetes community for excellent client libraries
- Prometheus for robust metrics infrastructure

## Related Work

- DZone: AI-Assisted Kubernetes Diagnostics series
- Medium: Kubernetes cost optimization techniques
- IEEE: LLM comparative analysis for DevOps

## Roadmap

- [x] Historical P95/P99 analysis (7-day Prometheus)
- [x] Workload-aware optimization (Deployment/StatefulSet/DaemonSet)
- [x] Environment-based safety buffers (prod/staging/dev)
- [x] Professional reporting (HTML/CSV/Markdown)
- [x] HPA detection and auto-skip
- [x] Multi-cloud pricing (Azure real-time)
- [x] Verbose debug mode
- [x] 94% test coverage
- [ ] Historical trend tracking (premium feature)
- [ ] Dashboard statistics and analytics
- [ ] Month-over-month comparison
- [ ] AWS/GCP real-time pricing APIs
- [ ] Web dashboard UI
- [ ] Helm chart deployment
- [ ] GitHub Actions CI/CD

---

For detailed development logs, see docs/WEEK*.md files.