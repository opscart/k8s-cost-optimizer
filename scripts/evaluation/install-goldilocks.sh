#!/bin/bash
# Install and test Goldilocks

set -e

GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing VPA (required for Goldilocks)..."
kubectl apply -f https://github.com/kubernetes/autoscaler/releases/download/vertical-pod-autoscaler-1.0.0/vpa-v1.0.0.yaml

echo -e "${GREEN}==>${NC} Installing Goldilocks..."
helm repo add fairwinds-stable https://charts.fairwinds.com/stable 2>/dev/null || true
helm repo update

helm install goldilocks fairwinds-stable/goldilocks \
    --namespace goldilocks \
    --create-namespace \
    --wait \
    --timeout 5m

# Enable goldilocks for cost-test namespace
kubectl label namespace cost-test goldilocks.fairwinds.com/enabled=true

echo ""
echo -e "${GREEN}✓${NC} Goldilocks installed successfully!"
echo ""
echo "Access Goldilocks Dashboard:"
echo "  kubectl port-forward -n goldilocks svc/goldilocks-dashboard 8080:80"
echo "  Open: http://localhost:8080"
echo ""
echo "Evaluation checklist: docs/evaluation/goldilocks-checklist.md"
