# k8s-cost-optimizer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-94%25-brightgreen)](https://github.com/opscart/k8s-cost-optimizer)

A production-ready Kubernetes cost optimization tool that identifies over-provisioned and under-provisioned workloads, providing actionable recommendations with accurate cloud pricing.

## Key Features

- **Multi-Cloud Pricing** - Supports Azure, AWS, GCP with dynamic pricing
- **Smart Detection** - Identifies both over-provisioning (cost savings) and under-provisioning (performance risks)
- **Actionable Commands** - Generates ready-to-use kubectl commands
- **Configurable Analysis** - 7-day default lookback (configurable 3-30 days)
- **Production Tested** - 94% test coverage with E2E validation on real clusters
- **Flexible Metrics** - Works with metrics-server or Prometheus for historical P95/P99

## Results

Tested on a 3-node cluster with 10 pods:
```
Total Potential Savings: $105.03/month

Breakdown:
- Over-provisioned: 5 workloads, $105/month savings
- Under-provisioned: 1 workload, Performance risk detected
- Well-sized: 4 workloads, No action needed
```

## Quick Start

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Go 1.21+ (for building from source)
- metrics-server OR Prometheus (for metrics)

### Installation
```bash
# Clone repository
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer

# Build
go build -o bin/cost-scan cmd/cost-scan/main.go

# Run scan
./bin/cost-scan -n production
```

### Sample Output
```
[INFO] K8s Cost Optimizer - Starting scan
[INFO] Connected to cluster (version: v1.31.0)
[INFO] Scanning namespace: production

=== Optimization Recommendations ===

1. production/api-server
   Type: RIGHT_SIZE
   Current: CPU=1000m Memory=2048Mi
   Recommended: CPU=300m Memory=384Mi (with 1.5x safety buffer)
   Savings: $27.19/month (Azure pricing)
   Risk: LOW
   Command: kubectl set resources deployment api-server -n production \
     --requests=cpu=300m,memory=384Mi

Total potential savings: $152.45/month
```

## Usage

### Basic Scan
```bash
# Scan single namespace
./cost-scan -n production

# Scan all namespaces
./cost-scan --all-namespaces
```

### Advanced Options
```bash
# Configure analysis parameters
export METRICS_LOOKBACK_DAYS=15    # Default: 7
export SAFETY_BUFFER=2.0           # Default: 1.5
export PROMETHEUS_URL=http://prometheus:9090

./cost-scan -n production

# Integration test with real APIs
go test -tags=integration ./pkg/pricing -v

# E2E test with real cluster
go test -tags=e2e ./tests/e2e -v
```

## Architecture
```
┌───────────────────────────────────────────────────────────┐
│              k8s-cost-optimizer                           │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────┐         ┌──────────────┐                │
│  │   Scanner   │────────▶│   Analyzer   │                │
│  └─────────────┘         └──────────────┘                │
│         │                        │                       │
│         │                        │                       │
│         ▼                        ▼                       │
│  ┌─────────────┐         ┌──────────────┐                │
│  │ Kubernetes  │         │   Metrics    │                │
│  │     API     │         │   Sources    │                │
│  │             │         │              │                │
│  │  - Nodes    │         │ ┌──────────┐ │                │
│  │  - Pods     │         │ │ metrics- │ │ ← Instant      │
│  │  - Deploys  │         │ │  server  │ │   metrics      │
│  └─────────────┘         │ └──────────┘ │                │
│         │                │ ┌──────────┐ │                │
│         │                │ │Prometheus│ │ ← P95/P99      │
│         │                │ │  (opt)   │ │   historical   │
│         │                │ └──────────┘ │                │
│         │                └──────────────┘                │
│         │                        │                       │
│         └────────┬───────────────┘                       │
│                  ▼                                       │
│          ┌──────────────┐                                │
│          │ Recommender  │                                │
│          └──────────────┘                                │
│                  │                                       │
│          ┌───────┴────────┐                              │
│          ▼                ▼                              │
│   ┌─────────────┐  ┌──────────────┐                      │
│   │   Pricing   │  │   Storage    │                      │
│   │  Providers  │  │ (PostgreSQL) │                      │
│   └─────────────┘  └──────────────┘                      │
│     │  │  │  │                                           │
│     ▼  ▼  ▼  ▼                                           │
│   Azure AWS GCP Default                                  │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Metrics Sources

The tool supports two metrics sources:

1. **metrics-server** (Default)
   - Instant CPU/memory snapshots
   - Built into most clusters
   - Fast, simple setup

2. **Prometheus** (Optional)
   - Historical P95/P99 metrics
   - Configurable lookback (7-30 days)
   - More accurate for variable workloads

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

### Test Strategy

Following industry best practices (Kubecost, VPA, Goldilocks):

1. **Unit Tests** - Fast, mocked, run on every commit
2. **Integration Tests** - Real cloud APIs, optional
3. **Contract Tests** - Recorded API responses
4. **E2E Tests** - Real cluster validation

## Cloud Pricing

### Supported Providers

| Provider | Status | Implementation |
|----------|--------|----------------|
| Azure    | Production | Real-time Azure Retail Prices API |
| AWS      | Beta | Static defaults (API integration planned) |
| GCP      | Beta | Static defaults (API integration planned) |
| Default  | Production | Conservative industry estimates |

**Note:** Pricing varies by region, instance type, and market conditions. Azure pricing is fetched in real-time via API. The tool caches pricing for 24 hours to minimize API calls.

### How Pricing Works
```bash
# Auto-detect cloud provider from node labels
./cost-scan -n production

# The tool will:
# 1. Detect cloud provider (Azure/AWS/GCP/on-prem)
# 2. Fetch current pricing for the region
# 3. Cache prices for 24 hours
# 4. Calculate savings based on actual rates
```

## Configuration

### Environment Variables
```bash
# Metrics lookback period (days)
METRICS_LOOKBACK_DAYS=7    # Options: 3, 7, 14, 30

# Safety buffer multiplier
SAFETY_BUFFER=1.5          # Options: 1.0-3.0

# Prometheus URL (optional)
PROMETHEUS_URL=http://localhost:9090

# Storage (optional)
STORAGE_ENABLED=true
DATABASE_URL=postgres://user:pass@localhost/costdb
```

### Presets
```bash
# Development (fast iteration)
METRICS_LOOKBACK_DAYS=3 SAFETY_BUFFER=1.5

# Production (balanced)
METRICS_LOOKBACK_DAYS=14 SAFETY_BUFFER=2.0

# Critical workloads (very conservative)
METRICS_LOOKBACK_DAYS=30 SAFETY_BUFFER=2.5
```

## Project Structure
```
k8s-cost-optimizer/
├── cmd/
│   ├── cost-scan/          # Main CLI
│   └── record-api-responses/ # API response recorder
├── pkg/
│   ├── analyzer/           # Pod metrics analysis
│   ├── config/             # Configuration management
│   ├── datasource/         # Prometheus integration
│   ├── pricing/            # Multi-cloud pricing
│   ├── recommender/        # Optimization logic
│   └── scanner/            # Kubernetes scanning
├── tests/
│   └── e2e/                # End-to-end tests
├── examples/
│   └── test-workloads/     # Sample deployments
├── testdata/
│   └── pricing/            # Recorded API responses
└── docs/
    ├── WEEK1_SUMMARY.md    # Development logs
    ├── WEEK2_SUMMARY.md
    ├── WEEK3_SUMMARY.md
    └── TECHNICAL_DEBT.md
```

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
# Clone and build
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer
go mod download
go build ./...

# Run tests
go test ./... -v

# Deploy test workloads
kubectl apply -f examples/test-workloads/
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

**Shamsher Khan**
- Senior DevOps Engineer
- IEEE Senior Member
- Technical Writer: DZone, Medium

## Acknowledgments

- Inspired by [Kubecost](https://kubecost.com), [Goldilocks](https://goldilocks.docs.fairwinds.com/), and [KRR](https://github.com/robusta-dev/krr)
- Kubernetes community for excellent client libraries
- Azure for comprehensive pricing APIs

## Related Work

- Research Paper: LLM-Driven Kubernetes Cost Optimization
- Blog: Why 7 Days? The Science Behind Metrics Lookback
- [DZone Articles](https://dzone.com/users/4868304/opscart.html)

## Roadmap

- [x] Multi-cloud pricing (Azure, AWS, GCP)
- [x] E2E testing with real clusters
- [x] 94% test coverage
- [x] Prometheus integration (P95/P99 support)
- [ ] AWS/GCP real-time pricing APIs
- [ ] Multi-cluster support
- [ ] Web dashboard
- [ ] Helm chart
- [ ] GitHub Actions CI/CD

---

**Star this repo if you find it useful!**
