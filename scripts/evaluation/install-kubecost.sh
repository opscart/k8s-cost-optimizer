#!/bin/bash
# Install and test Kubecost

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing Kubecost..."

helm repo add kubecost https://kubecost.github.io/cost-analyzer/ 2>/dev/null || true
helm repo update

helm install kubecost kubecost/cost-analyzer \
    --namespace kubecost \
    --create-namespace \
    --set prometheus.server.global.external_labels.cluster_id=minikube-test \
    --set prometheus.nodeExporter.enabled=false \
    --set prometheus.serviceAccounts.nodeExporter.create=false \
    --set kubecostModel.warmCache=false \
    --set kubecostModel.warmSavingsCache=false \
    --wait \
    --timeout 5m

echo -e "${GREEN}==>${NC} Waiting for Kubecost to be ready..."
kubectl wait --for=condition=ready pod -l app=cost-analyzer -n kubecost --timeout=300s

echo ""
echo -e "${GREEN}✓${NC} Kubecost installed successfully!"
echo ""
echo "Access Kubecost UI:"
echo "  kubectl port-forward -n kubecost svc/kubecost-cost-analyzer 9090:9090"
echo "  Open: http://localhost:9090"
echo ""
echo -e "${YELLOW}Note:${NC} Wait 10-15 minutes for data to populate before evaluation"
echo ""
echo "Evaluation checklist: docs/evaluation/kubecost-checklist.md"