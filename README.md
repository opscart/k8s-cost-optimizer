# Kubernetes Cost Optimizer

> An open-source, CLI-first tool for Kubernetes cost optimization with persistent storage and audit trails

**Status:** 🚀 Active Development (Week 2 Complete)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](go.mod)

## Features

 **Smart Recommendations**
- Right-sizing based on actual usage
- Idle workload detection
- Over-provisioning alerts

 **Persistent Storage**
- PostgreSQL backend for recommendations
- Historical tracking of all recommendations
- Audit trail for applied changes

 **Multiple Output Formats**
- Human-readable text reports
- JSON for automation
- Direct kubectl commands

 **Risk Assessment**
- Safety ratings for each recommendation
- Impact analysis
- Monthly savings calculations

## Quick Start

### Prerequisites
- Kubernetes cluster (minikube, AKS, EKS, GKE)
- kubectl configured
- Docker (for PostgreSQL)
- Go 1.21+ (for building from source)

### 1. Setup Local Test Environment
```bash
# Start minikube
minikube start

# Deploy PostgreSQL for storage
docker run -d \
  --name k8s-cost-postgres \
  -e POSTGRES_USER=costuser \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=costoptimizer \
  -p 5432:5432 \
  postgres:14

# Deploy test workloads
kubectl create namespace cost-test
kubectl apply -f examples/test-workloads/
```

### 2. Build & Install
```bash
# Clone repository
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer

# Build CLI
go build -o bin/cost-scan cmd/cost-scan/main.go

# Optional: Install globally
sudo mv bin/cost-scan /usr/local/bin/
```

### 3. Run Your First Scan
```bash
# Scan a namespace and save to database
cost-scan -n cost-test --save

# View recommendations history
cost-scan history cost-test

# View audit trail for a recommendation
cost-scan audit <recommendation-id>
```

## Usage

### Scan Cluster
```bash
# Scan specific namespace
cost-scan -n production

# Scan all namespaces
cost-scan --all-namespaces

# Save recommendations to database
cost-scan -n production --save

# Specify cluster identifier
cost-scan -n production --save --cluster-id=prod-us-east-1
```

### Output Formats
```bash
# Human-readable (default)
cost-scan -n production

# JSON for automation
cost-scan -n production -o json

# Direct kubectl commands
cost-scan -n production -o commands
```

### View History
```bash
# View past recommendations for namespace
cost-scan history production

# Limit results
cost-scan history production --limit 20
```

### Audit Trail
```bash
# View audit log for a specific recommendation
cost-scan audit <recommendation-id>
```

## Example Output
```
=== Optimization Recommendations ===

1. production/api-service
   Type: RIGHT_SIZE
   Current:  CPU=1000m Memory=2048Mi
   Recommended: CPU=250m Memory=512Mi
   Savings: $45.67/month
   Risk: LOW
   Command: kubectl set resources deployment api-service -n production --requests=cpu=250m,memory=512Mi

Total potential savings: $127.83/month
```

## Architecture
```
┌─────────────┐
│   CLI Tool  │
│ (cost-scan) │
└──────┬──────┘
       │
       ├─────────────┐
       ▼             ▼
┌─────────────┐  ┌──────────────┐
│  Scanner    │  │  PostgreSQL  │
│  Analyzer   │  │   Storage    │
│  Recommender│  └──────────────┘
└─────────────┘
       │
       ▼
┌─────────────┐
│  Kubernetes │
│   Metrics   │
└─────────────┘
```

See [docs/architecture/CANVAS.md](docs/architecture/CANVAS.md) for details.

## Configuration

Environment variables:
```bash
# Database connection
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"

# Disable storage
export STORAGE_ENABLED=false

# Prometheus endpoint (future)
export PROMETHEUS_URL="http://localhost:9090"
```

## Documentation

- [Database Schema](docs/database/README.md) - PostgreSQL table structure
- [Architecture Canvas](docs/architecture/CANVAS.md) - System design
- [Technical Debt](docs/TECHNICAL_DEBT.md) - Known issues and migration plans
- [Development Guide](CONTRIBUTING.md) - How to contribute

## Project Structure
```
k8s-cost-optimizer/
├── cmd/
│   ├── cost-scan/           # Main CLI application
│   └── test-postgres/       # Database testing tool
├── pkg/
│   ├── analyzer/            # Pod analysis logic
│   ├── config/              # Configuration management
│   ├── converter/           # Model conversion (temporary)
│   ├── executor/            # Command generation
│   ├── models/              # Data models
│   ├── recommender/         # Recommendation engine
│   ├── scanner/             # Kubernetes scanning
│   └── storage/             # PostgreSQL storage
├── docs/
│   ├── architecture/        # Architecture docs
│   ├── database/            # Database schema docs
│   └── evaluation/          # Tool comparison research
├── examples/
│   └── test-workloads/      # Sample workloads
└── scripts/
    ├── setup/               # Environment setup
    └── evaluation/          # Tool evaluation
```

## Development Roadmap

###  Week 1: Foundation (Complete)
- [x] Local environment setup
- [x] Test workload deployment
- [x] Tool evaluation (Kubecost, OpenCost, Goldilocks)
- [x] Basic scanner implementation
- [x] Recommendation engine
- [x] CLI with multiple output formats

###  Week 2: Storage Layer (Complete)
- [x] PostgreSQL integration
- [x] Database schema design
- [x] Storage implementation with audit trails
- [x] History and audit commands
- [x] Technical debt documentation

###  Week 3: Pricing & Accuracy (In Progress)
- [ ] Cloud provider pricing integration
- [ ] Prometheus metrics integration
- [ ] P95/P99 percentile analysis
- [ ] Multi-cluster support
- [ ] Enhanced recommendation logic

###  Week 4: Polish & Documentation
- [ ] Comprehensive testing
- [ ] User documentation
- [ ] CI/CD setup
- [ ] Performance optimization
- [ ] Release preparation

## Testing
```bash
# Run unit tests
go test ./...

# Test PostgreSQL storage
go run cmd/test-postgres/main.go

# Integration test with minikube
./scripts/test/integration-test.sh
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Related Projects

This tool was developed after evaluating:
- [Kubecost](https://kubecost.com) - Commercial with free tier
- [OpenCost](https://opencost.io) - CNCF sandbox project
- [Goldilocks](https://github.com/FairwindsOps/goldilocks) - VPA-based recommendations

See [docs/evaluation/](docs/evaluation/) for detailed comparison.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Support

- 📧 Email: [your-email]
- 🐛 Issues: [GitHub Issues](https://github.com/opscart/k8s-cost-optimizer/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/opscart/k8s-cost-optimizer/discussions)

## Acknowledgments

Built with:
- [Kubernetes client-go](https://github.com/kubernetes/client-go)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [PostgreSQL](https://www.postgresql.org/)

---

**Made with ❤️ for the Kubernetes community**
