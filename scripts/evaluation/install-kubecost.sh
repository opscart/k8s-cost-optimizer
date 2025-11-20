#!/bin/bash
# Install and test Kubecost

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing Kubecost (v2.4.2)..."

helm repo add kubecost https://kubecost.github.io/cost-analyzer/ 2>/dev/null || true
helm repo update

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
    --set persistentVolume.enabled=false \
    --set kubecostModel.etlFileStoreEnabled=true \
    --set kubecostModel.etlStorePath=/tmp/kubecost-data \
    --timeout 10m

echo -e "${GREEN}==>${NC} Waiting for Kubecost to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cost-analyzer -n kubecost --timeout=300s 2>/dev/null || \
kubectl wait --for=condition=ready pod -l app=cost-analyzer -n kubecost --timeout=300s

# Delete Grafana deployment (uses Chainguard images that require payment)
echo -e "${GREEN}==>${NC} Removing Grafana deployment..."
sleep 5
kubectl delete deployment kubecost-grafana -n kubecost 2>/dev/null || true

echo ""
echo -e "${GREEN}✓${NC} Kubecost installed successfully!"
echo ""
echo "Pods:"
kubectl get pods -n kubecost
echo ""
echo "Access Kubecost UI:"
echo "  kubectl port-forward -n kubecost deployment/kubecost-cost-analyzer 9090:9090"
echo "  Open: http://localhost:9090"
echo ""
echo -e "${YELLOW}Note:${NC} Wait 10-15 minutes for cost data to populate"
echo ""
echo "Evaluation checklist: docs/evaluation/kubecost-checklist.md"
