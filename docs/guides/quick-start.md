# Quick Start Guide

Get started with k8s-cost-optimizer in 5 minutes.

## Prerequisites

- Kubernetes cluster (minikube, kind, or any cloud provider)
- kubectl configured and working
- Docker (for PostgreSQL storage)
- Go 1.21+ (to build from source)

## Step 1: Setup Test Environment

### Option A: Minikube (Recommended for testing)
```bash
# Start minikube
minikube start --cpus=2 --memory=4096

# Verify cluster is running
kubectl get nodes
```

### Option B: Use existing cluster
```bash
# Verify access
kubectl get nodes
kubectl cluster-info
```

## Step 2: Deploy Test Workloads
```bash
# Create test namespace
kubectl create namespace cost-test

# Deploy sample workloads
kubectl apply -f https://raw.githubusercontent.com/opscart/k8s-cost-optimizer/main/examples/test-workloads/idle-app.yaml
kubectl apply -f https://raw.githubusercontent.com/opscart/k8s-cost-optimizer/main/examples/test-workloads/overprovision-app.yaml

# Verify deployments
kubectl get pods -n cost-test
```

## Step 3: Setup PostgreSQL (Optional but Recommended)
```bash
# Run PostgreSQL container
docker run -d \
  --name k8s-cost-postgres \
  -e POSTGRES_USER=costuser \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=costoptimizer \
  -p 5432:5432 \
  postgres:14

# Verify it's running
docker ps | grep postgres
```

## Step 4: Install k8s-cost-optimizer

### Build from source
```bash
# Clone repository
git clone https://github.com/opscart/k8s-cost-optimizer.git
cd k8s-cost-optimizer

# Build
go build -o bin/cost-scan cmd/cost-scan/main.go

# Test it works
./bin/cost-scan --help
```

### Install globally (optional)
```bash
sudo cp bin/cost-scan /usr/local/bin/
cost-scan --help
```

## Step 5: Run Your First Scan
```bash
# Simple scan (no storage)
./bin/cost-scan -n cost-test

# Scan and save to database
./bin/cost-scan -n cost-test --save
```

**Expected output:**
```
[INFO] K8s Cost Optimizer - Starting scan
[INFO] Results will be saved to database
[INFO] Connected to cluster (version: v1.31.0)
[INFO] Scanning namespace: cost-test
[INFO] Found 2 recommendation(s)

=== Optimization Recommendations ===

1. cost-test/idle-app
   Type: SCALE_DOWN
   Current:  CPU=500m Memory=512Mi
   Recommended: CPU=0m Memory=0Mi
   Savings: $39.00/month
   Risk: MEDIUM
   Command: kubectl scale deployment idle-app -n cost-test --replicas=0

Total potential savings: $39.00/month
```

## Step 6: View History
```bash
# View past recommendations
./bin/cost-scan history cost-test

# View audit trail
./bin/cost-scan audit <recommendation-id>
```

## Step 7: Export Commands
```bash
# Generate kubectl commands only
./bin/cost-scan -n cost-test -o commands

# Save to script
./bin/cost-scan -n cost-test -o commands > optimize.sh
chmod +x optimize.sh

# Review before running!
cat optimize.sh
```

## Next Steps

- [Storage Setup Guide](storage.md) - Configure persistent storage
- [Configuration Guide](configuration.md) - Customize behavior
- [Use Cases](../examples/) - Real-world scenarios

## Troubleshooting

**Problem: "Error initializing scanner"**
```bash
# Check kubectl access
kubectl get nodes

# Check kubeconfig
export KUBECONFIG=~/.kube/config
```

**Problem: "Failed to initialize storage"**
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -U costuser -d costoptimizer
```

**Problem: "No workloads found"**
```bash
# Verify namespace exists
kubectl get namespaces

# Check pods are running
kubectl get pods -n cost-test
```

## Cleanup
```bash
# Stop PostgreSQL
docker stop k8s-cost-postgres
docker rm k8s-cost-postgres

# Delete test namespace
kubectl delete namespace cost-test

# Stop minikube (if using)
minikube stop
```

---

**Need help?** Open an issue on [GitHub](https://github.com/opscart/k8s-cost-optimizer/issues)
