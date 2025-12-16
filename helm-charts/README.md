# K8s Cost Optimizer Helm Repository

## Usage
```bash
# Add the repository
helm repo add k8s-cost-optimizer https://opscart.github.io/k8s-cost-optimizer/helm-charts

# Update repositories
helm repo update

# Install the chart
helm install cost-optimizer k8s-cost-optimizer/k8s-cost-optimizer \
  --namespace cost-optimizer \
  --create-namespace

# Search available versions
helm search repo k8s-cost-optimizer
```

## Chart Versions

See [index.yaml](./index.yaml) for available versions.

## Chart Documentation

See [Chart README](https://github.com/opscart/k8s-cost-optimizer/tree/main/charts/k8s-cost-optimizer)

## Source

GitHub: https://github.com/opscart/k8s-cost-optimizer
