#!/bin/bash
# Install and test Kubecost

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing Kubecost (v2.4.x - stable)..."

helm repo add kubecost https://kubecost.github.io/cost-analyzer/ 2>/dev/null || true
helm repo update

# Clean up any existing installation
helm uninstall kubecost -n kubecost 2>/dev/null || true
kubectl delete namespace kubecost 2>/dev/null || true
sleep 5

helm install kubecost kubecost/cost-analyzer \
    --namespace kubecost \
    --create-namespace \
    --version 2.4.2 \
    --set global.clusterId="minikube-test" \
    --set kubecostToken="" \
    --set grafana.enabled=false \
    --set prometheus.enabled=false \
    --set prometheus.server.enabled=false \
    --set global.prometheus.enabled=false \
    --set global.prometheus.fqdn=http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090 \
    --wait \
    --timeout 10m

echo -e "${GREEN}==>${NC} Waiting for Kubecost to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cost-analyzer -n kubecost --timeout=300s 2>/dev/null || \
kubectl wait --for=condition=ready pod -l app=cost-analyzer -n kubecost --timeout=300s

echo ""
echo -e "${GREEN}✓${NC} Kubecost installed successfully!"
echo ""
echo "Pods running:"
kubectl get pods -n kubecost
echo ""
echo "Access Kubecost UI:"
echo "  kubectl port-forward -n kubecost svc/kubecost-cost-analyzer 9090:9090"
echo "  Open: http://localhost:9090"
echo ""
echo -e "${YELLOW}Note:${NC} Wait 10-15 minutes for data to populate before evaluation"
echo ""
echo "Evaluation checklist: docs/evaluation/kubecost-checklist.md"