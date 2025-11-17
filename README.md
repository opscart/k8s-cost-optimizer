# Kubernetes Cost Optimizer

> A lightweight, CLI-first tool for Kubernetes cost optimization

**Status:** 🔬 Research & Evaluation Phase

## Quick Start (Local Development)
```bash
# 1. Setup local minikube environment (one command!)
./scripts/setup/setup-local-env.sh

# 2. Deploy test workloads
./scripts/setup/deploy-test-workloads.sh

# 3. Install evaluation tools
./scripts/evaluation/install-kubecost.sh
./scripts/evaluation/install-opencost.sh
./scripts/evaluation/install-goldilocks.sh
```

## Repository Structure
```
k8s-cost-optimizer/
├── cmd/                          # CLI applications
├── internal/                     # Private application code
│   ├── collector/               # Metrics & data collection
│   ├── analyzer/                # Cost analysis logic
│   ├── recommender/             # Recommendation engine
│   └── reporter/                # Output formatting
├── pkg/                         # Public libraries
├── scripts/
│   ├── setup/                   # Environment setup
│   └── evaluation/              # Tool evaluation scripts
├── examples/
│   └── test-workloads/          # Sample K8s manifests
├── docs/
│   └── evaluation/              # Tool evaluation docs
└── test/                        # Tests
```

## Current Phase: Tool Evaluation

We're evaluating existing cost optimization tools to identify gaps:

- **Kubecost** - Commercial with free tier
- **OpenCost** - CNCF sandbox project
- **Goldilocks** - VPA-based right-sizing

See [docs/evaluation/](docs/evaluation/) for detailed findings.

## Requirements

- minikube or local K8s cluster
- kubectl
- helm
- Docker (for minikube driver)

## Cleanup
```bash
./scripts/setup/cleanup.sh
```

## License

Apache 2.0 (TBD)