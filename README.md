# k8s-cost-optimizer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-94%25-brightgreen)](https://github.com/opscart/k8s-cost-optimizer)
[![Docker](https://img.shields.io/badge/docker-shamsk22%2Fk8s--cost--optimizer-blue)](https://hub.docker.com/r/shamsk22/k8s-cost-optimizer)

A Kubernetes cost optimization tool that analyzes historical P95/P99 metrics from Prometheus to generate resource recommendations.

---

## ⚠️ Project Status: Experimental

**NOT production-ready. Use at your own risk.**

**What works:**
- ✅ Local clusters (Minikube/kind)
- ✅ Prometheus historical analysis (7-day P95)
- ✅ Pattern detection and confidence scoring
- ✅ Docker image and Kubernetes manifests
- ✅ 94% test coverage

**What doesn't work:**
- ❌ Real AKS/EKS/GKE clusters (auth issues with kubelogin/IAM)
- ❌ Large-scale testing (tested <50 pods, not 1000+)
- ❌ Production validation (no SLO/SLA checks)
- ❌ Helm chart (manual manifests only)

**This is a side project** for learning. For production use, consider [Kubecost](https://www.kubecost.com/).

---

## Key Differentiators

### Historical Trend Analysis
Unlike tools using instant snapshots, k8s-cost-optimizer analyzes **7-day P95/P99 trends** from Prometheus (2000+ samples per workload) for informed recommendations.

### Workload-Aware Optimization
Different workload types get different safety margins:
- **Deployments**: 1.5x buffer (stateless, low risk)
- **StatefulSets**: 2.0x buffer (stateful, medium risk)
- **DaemonSets**: Optimization disabled (critical services)

### Environment-Based Safety
Production workloads get extra protection:
- **Production**: 1.3x additional multiplier (2.6x total for StatefulSets)
- **Staging**: 1.0x standard
- **Development**: 0.85x aggressive (15% more savings)

---

## Features

### Core Analysis

#### Pattern-Aware Safety Buffers
Adjusts buffers based on workload behavior:
- **Steady** (CV < 0.2): -10% buffer
- **Spiky** (CV 0.5-0.8): +15% buffer  
- **Highly-variable** (CV > 0.8): +25% buffer

#### Growth Prediction
Linear regression to detect growth:
- Analyzes 7-day trends for growth rate
- **Note:** 7 days is too short for reliable long-term predictions
- Adds buffer for workloads growing >5%/month
- Plans for 3-month future usage

#### Confidence Scoring
Transparent data quality:
- **HIGH** ✓: 7 days data + steady pattern
- **MEDIUM** ~: 3+ days data + acceptable quality
- **LOW** ?: Insufficient data (<3 days)

#### Weekday vs Weekend Analysis
Separate P95 for business patterns:
- Monday-Friday vs Saturday-Sunday split
- Uses higher P95 for safety
- Shows when >20% difference detected
- Example: "Memory (Weekday: 30Mi, Weekend: 22Mi)"

#### Enhanced Reason Strings
Context-rich explanations:
- Pattern info: "Pattern: CPU steady (CV: 0.12)"
- Growth warnings: "⚠️ Growing 50%/month"
- Data quality: "⚠️ Limited historical data"
- Utilization: "CPU: 95%, Memory: 92%"

### Additional Features

- **Historical P95/P99 Analysis** - 7-day Prometheus lookback
- **Workload Type Detection** - Automatic classification
- **Environment Classification** - Label and name-pattern detection
- **HPA Detection** - Auto-skips auto-scaling workloads
- **Multi-Cloud Pricing** - Azure, AWS/GCP (estimates)
- **Graceful Fallback** - Uses instant metrics when Prometheus unavailable

### Reporting
- **HTML Executive Reports** - CNCF-themed dashboards
- **CSV Exports** - Finance-friendly spreadsheets
- **Markdown Reports** - GitHub/wiki compatible
- **Top Savings Analysis** - Ranked opportunities

---

## Quick Start

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Go 1.21+ (for building from source)
- Prometheus with 7+ days retention (recommended)
- PostgreSQL (optional - for analytics)

### Installation
```bash
# Clone repository
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer

# Build
go build -o bin/k8s-cost-optimizer ./cmd/cost-scan

# Run scan
./bin/k8s-cost-optimizer -n production --prometheus-url http://localhost:9090
```

### Sample Output
```
[INFO] K8s Cost Optimizer - Starting scan
[INFO] Using Prometheus at http://localhost:9090
[INFO] Connected to cluster (version: v1.31.0)
[INFO] Scanning namespace: production
[INFO] Using historical analysis (P95 over 7 days)

=== Optimization Recommendations ===

1. production/api-server [PRODUCTION]
   Type: RIGHT_SIZE
   Confidence: ✓ HIGH (CPU: steady)
   Reason: Over-provisioned: CPU 65% under-utilized | 
           Pattern: CPU steady (CV: 0.12) | 
           Workload: Deployment, Safety: 2.0x, Env: production
           (Based on 7-day P95: CPU 350m, Memory 512Mi)
   Current:  CPU=1000m Memory=2048Mi
   Recommended: CPU=700m Memory=1024Mi
   Savings: $42.50/month
   Risk: LOW
   Command: kubectl set resources deployment api-server -n production \
            --requests=cpu=700m,memory=1024Mi

Total potential savings: $152.45/month
```

---

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
│  │   detection │         │ - Patterns   │                    │
│  └─────────────┘         │ - Growth     │                    │
│         │                └──────────────┘                    │
│         │                        │                           │
│         v                        v                           │
│  ┌─────────────┐         ┌──────────────┐                    │
│  │ Kubernetes  │         │   Metrics    │                    │
│  │     API     │         │   Sources    │                    │
│  │             │         │              │                    │
│  │  - Nodes    │         │ ┌──────────┐ │                    │
│  │  - Pods     │         │ │Prometheus│ │ <- 7-day P95/P99  │
│  │  - Deploys  │         │ │  (2000+  │ │    historical     │
│  │  - StatefulS│         │ │ samples) │ │    analysis       │
│  │  - DaemonSet│         │ └──────────┘ │                    │
│  │  - HPAs     │         │ ┌──────────┐ │                    │
│  └─────────────┘         │ │ metrics- │ │ <- Instant        │
│         │                │ │  server  │ │    fallback       │
│         │                │ └──────────┘ │                    │
│         │                └──────────────┘                    │
│         └────────┬───────────────┘                           │
│                  v                                           │
│          ┌──────────────┐                                    │
│          │ Recommender  │                                    │
│          │              │                                    │
│          │ - Workload   │                                    │
│          │   awareness  │                                    │
│          │ - Pattern    │                                    │
│          │   buffers    │                                    │
│          │ - Growth     │                                    │
│          │   prediction │                                    │
│          │ - Confidence │                                    │
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

---

## How It Works

### 1. Historical Analysis
```
For each workload:
1. Query Prometheus for 7 days (5-minute intervals)
2. Collect ~2000 CPU and memory samples
3. Calculate P50, P90, P95, P99 percentiles
4. Analyze patterns (coefficient of variation)
5. Detect growth trends (linear regression)
6. Split weekday/weekend if significant difference
7. Calculate confidence score
8. Use P95 as baseline for recommendations
```

### 2. Safety Buffer Calculation
```
Base Buffer = Workload Type × Environment
Pattern Adjustment = Based on CV (±10-25%)
Growth Buffer = 50% of 3-month prediction
Final = Constrained between 1.2x and 3.0x

Example (StatefulSet in Production, Spiky pattern, 10% growth):
  Base: 2.0 (StatefulSet) × 1.3 (Production) = 2.6x
  Pattern: 2.6 × 1.15 (Spiky) = 2.99x
  Growth: +50% of predicted 3-month growth
  Final: Capped at 3.0x maximum
```

---

## CLI Usage

### Basic Scanning
```bash
# Single namespace
./bin/k8s-cost-optimizer -n production \
  --prometheus-url http://localhost:9090

# All namespaces
./bin/k8s-cost-optimizer --all-namespaces \
  --prometheus-url http://localhost:9090

# Verbose mode
./bin/k8s-cost-optimizer -n production -v

# Custom lookback
./bin/k8s-cost-optimizer -n production \
  --lookback-days 14
```

### Report Generation
```bash
# HTML report
./bin/k8s-cost-optimizer -n production \
  --generate-report

# CSV export
./bin/k8s-cost-optimizer -n production \
  --generate-report \
  --report-format csv

# Markdown
./bin/k8s-cost-optimizer -n production \
  --generate-report \
  --report-format markdown
```

Reports are saved to `reports/` directory with timestamps.

### CLI Flags
```
Scanning:
  -n, --namespace          Namespace to scan
  -A, --all-namespaces    Scan all namespaces
  -v, --verbose           Debug logging

Prometheus:
  --prometheus-url        Prometheus URL (default: http://localhost:9090)
  --lookback-days        Historical window (default: 7)
  --use-prometheus       Enable Prometheus (default: true)

Output:
  -o, --output           Format: text, json, commands
  --generate-report      Generate report
  --report-format        html, csv, markdown

Storage (Optional):
  --save                 Save to PostgreSQL
  --cluster-id          Cluster identifier

Provider:
  --provider            azure, aws, gcp (auto-detect)
  --region              Cloud region
```

---

## Deployment to Kubernetes

### Option 1: Manual Deployment (Minikube)
```bash
# Create namespace and RBAC
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/rbac.yaml

# Deploy as CronJob (runs daily at 2 AM)
kubectl apply -f manifests/cronjob.yaml

# Manual test
kubectl create job --from=cronjob/cost-optimizer-scan test-scan -n cost-optimizer

# View logs
kubectl logs -n cost-optimizer -l job-name=test-scan
```

### Option 2: Docker (Local Testing)
```bash
# Pull image
docker pull shamsk22/k8s-cost-optimizer:latest

# Run scan
docker run --rm \
  -v ~/.kube/config:/kubeconfig \
  -e KUBECONFIG=/kubeconfig \
  -e PROMETHEUS_URL=http://host.docker.internal:9090 \
  shamsk22/k8s-cost-optimizer:latest \
  -n production -v
```

---

## PostgreSQL Storage & Analytics (Optional)

### Setup
```bash
# Start PostgreSQL
docker run -d --name cost-optimizer-db \
  -e POSTGRES_PASSWORD=costpass \
  -e POSTGRES_DB=costdb \
  -p 5432:5432 \
  postgres:14

# Configure
export DATABASE_URL="postgres://postgres:costpass@localhost:5432/costdb?sslmode=disable"

# Run with save
./bin/k8s-cost-optimizer -n production --save
```

### Analytics Commands
```bash
# Dashboard stats
./bin/k8s-cost-optimizer analytics stats -n production --days 30

# Savings trends
./bin/k8s-cost-optimizer analytics trends -n production --days 30

# Period comparison
./bin/k8s-cost-optimizer analytics compare -n production --days 30

# Workload history
./bin/k8s-cost-optimizer analytics workload -n production --deployment api-server
```

See `docs/guides/storage.md` for detailed setup and schema information.

---

## Known Limitations

### Data & Analysis
- **7 days too short**: Can't detect monthly/seasonal patterns
- **5-min intervals**: Misses sub-minute spikes
- **No OOM detection**: Doesn't check for memory kills
- **No throttling**: Doesn't check CPU throttling
- **Arbitrary thresholds**: Safety buffers not validated

### Safety
- **No SLO validation**: Can't verify against your objectives
- **No rollback tracking**: Doesn't detect reverted changes
- **Static learning**: Doesn't adapt over time
- **No alerting integration**: Can't check error rates

### Scale & Production
- **Untested at scale**: Performance unknown with 500+ pods
- **Auth issues**: AKS/EKS/GKE with IAM not working
- **No multi-cluster**: Single cluster only
- **No Helm chart**: Manual manifest deployment

### Missing Features
- No Slack/Teams notifications
- No web dashboard
- No GitOps integration
- No approval workflow

---

## Comparison with Other Tools

| Feature | k8s-cost-optimizer | Kubecost | Goldilocks | KRR |
|---------|-------------------|----------|------------|-----|
| **Status** | Experimental | Production | Stable | Stable |
| **Support** | None | Commercial | Community | Community |
| **Historical** | 7-day P95 | ✓ Configurable | ❌ VPA | ❌ Instant |
| **Pattern Detection** | ✓ Experimental | ❌ | ❌ | ❌ |
| **Growth Prediction** | ✓ Experimental | ❌ | ❌ | ❌ |
| **Confidence Scoring** | ✓ | ❌ | ❌ | ❌ |
| **Weekday/Weekend** | ✓ | ❌ | ❌ | ❌ |
| **Large Scale** | ❌ Untested | ✓ Production | ✓ | ✓ |
| **Auth (AKS/EKS)** | ❌ Broken | ✓ Works | ✓ Works | ✓ Works |
| **Web UI** | ❌ | ✓ | ❌ | ❌ |
| **Helm Chart** | ✓ v0.1.0| ✓ | ✓ | ✓ |
| **Docker Image** | ✓ Docker Hub | ✓ | ✓ | ✓ |

**When to use this tool:**
- Learning about cost optimization
- Local/dev cluster experimentation
- Understanding pattern-based analysis

**When NOT to use:**
- Production clusters
- Need for support
- Large scale (1000+ pods)
- Enterprise requirements

---

## Development

### Build from Source
```bash
git clone https://github.com/opscart/k8s-cost-optimizer
cd k8s-cost-optimizer

# Build
go build -o bin/k8s-cost-optimizer ./cmd/cost-scan

# Run tests (94% coverage)
go test ./... -v

# Test coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Docker Build
```bash
docker build -t k8s-cost-optimizer:local .
docker run --rm k8s-cost-optimizer:local --help
```

### Project Structure
```
k8s-cost-optimizer/
├── cmd/cost-scan/          # CLI application
├── pkg/
│   ├── analyzer/           # Metrics & pattern analysis
│   ├── recommender/        # Recommendation engine
│   ├── scanner/            # Workload discovery
│   ├── storage/            # PostgreSQL persistence
│   ├── pricing/            # Cloud pricing
│   └── reporter/           # Report generation
├── manifests/              # Kubernetes YAML
├── examples/               # Test workloads
└── docs/                   # Documentation
```

---

## Configuration

### Environment Variables
```bash
# Prometheus
PROMETHEUS_URL=http://prometheus.monitoring.svc:9090

# Historical Analysis
METRICS_LOOKBACK_DAYS=7  # 1-30 days
SAFETY_BUFFER=1.5        # 1.0-3.0

# PostgreSQL (optional)
DATABASE_URL=postgres://user:pass@host:5432/dbname
STORAGE_ENABLED=true

# Cloud Provider
CLOUD_PROVIDER=azure
CLOUD_REGION=eastus

# Cluster
CLUSTER_ID=my-cluster
```

---

## Testing

### Test Coverage: 94%
```bash
# Unit tests
go test ./... -cover

# Integration tests
go test -tags=integration ./pkg/pricing -v

# E2E tests
go test -tags=e2e ./tests/e2e -v
```

---

## Contributing

Contributions welcome! This is a learning project.

**How to contribute:**
1. Open issue first (discuss before coding)
2. Fork and create feature branch
3. Add tests (maintain 90%+ coverage)
4. Update README
5. Submit PR

**What we need help with:**
- Testing on real AKS/EKS/GKE
- Fixing auth issues
- Helm chart creation
- Performance optimization

---

## License

MIT License - See [LICENSE](LICENSE)

Use at your own risk. No warranty provided.

---

## Author

**Shamsher Khan**
- Senior DevOps Engineer
- IEEE Senior Member
- Technical Writer: DZone, Medium, InfoQ

---

## Acknowledgments

- Inspired by Kubecost, Goldilocks, and KRR
- Built for learning and experimentation
- Not intended for commercial use

---

**Remember:** This is experimental software. For production, use [Kubecost](https://www.kubecost.com/).
