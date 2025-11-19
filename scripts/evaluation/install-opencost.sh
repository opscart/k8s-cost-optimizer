#!/bin/bash
# Install and test OpenCost via Helm

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing OpenCost via Helm..."

helm repo add opencost https://opencost.github.io/opencost-helm-chart 2>/dev/null || true
helm repo update

helm install opencost opencost/opencost \
    --namespace opencost \
    --create-namespace \
    --set opencost.exporter.defaultClusterId=minikube-test \
    --set opencost.ui.enabled=true \
    --timeout 5m

echo -e "${GREEN}==>${NC} Configuring Prometheus endpoint..."
# Wait for deployment to be created
sleep 5

# Patch deployment to use correct Prometheus URL
kubectl set env deployment/opencost -n opencost \
  PROMETHEUS_SERVER_ENDPOINT=http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090

echo -e "${GREEN}==>${NC} Waiting for OpenCost to be ready..."
kubectl wait --for=condition=ready pod \
    -l app.kubernetes.io/name=opencost \
    -n opencost \
    --timeout=300s

echo ""
echo -e "${GREEN}✓${NC} OpenCost installed successfully!"
echo ""
echo "Pods:"
kubectl get pods -n opencost
echo ""
echo "Access OpenCost UI:"
echo "  kubectl port-forward -n opencost svc/opencost 9090:9090"
echo "  Open: http://localhost:9090"
echo ""
echo "Access OpenCost API (optional):"
echo "  kubectl port-forward -n opencost svc/opencost 9003:9003"
echo "  Test: curl http://localhost:9003/allocation/compute?window=1d | jq '.'"
echo ""
echo -e "${YELLOW}Note:${NC} Wait 5-10 minutes for cost data to populate"
echo ""
echo "Evaluation checklist: docs/evaluation/opencost-checklist.md"
