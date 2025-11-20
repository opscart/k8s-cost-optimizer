#!/bin/bash
# Install and test Goldilocks

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing VPA (required for Goldilocks)..."

# Clone VPA repo if not already present
if [ ! -d "/tmp/autoscaler" ]; then
    git clone https://github.com/kubernetes/autoscaler.git /tmp/autoscaler
fi

# Install VPA
cd /tmp/autoscaler/vertical-pod-autoscaler
./hack/vpa-up.sh

cd -

echo -e "${GREEN}==>${NC} Waiting for VPA to be ready..."
sleep 10
kubectl wait --for=condition=ready pod -l app=vpa-admission-controller -n kube-system --timeout=120s 2>/dev/null || true

echo -e "${GREEN}==>${NC} Installing Goldilocks..."
helm repo add fairwinds-stable https://charts.fairwinds.com/stable 2>/dev/null || true
helm repo update

helm install goldilocks fairwinds-stable/goldilocks \
    --namespace goldilocks \
    --create-namespace \
    --wait \
    --timeout 5m

# Enable goldilocks for cost-test namespace
echo -e "${GREEN}==>${NC} Enabling Goldilocks for cost-test namespace..."
kubectl label namespace cost-test goldilocks.fairwinds.com/enabled=true --overwrite

echo ""
echo -e "${GREEN}✓${NC} Goldilocks installed successfully!"
echo ""
echo "VPA Pods:"
kubectl get pods -n kube-system | grep vpa
echo ""
echo "Goldilocks Pods:"
kubectl get pods -n goldilocks
echo ""
echo "Access Goldilocks Dashboard:"
echo "  kubectl port-forward -n goldilocks svc/goldilocks-dashboard 8080:80"
echo "  Open: http://localhost:8080"
echo ""
echo -e "${YELLOW}Note:${NC} Wait 10-15 minutes for VPA to collect usage data"
echo ""
echo "Evaluation checklist: docs/evaluation/goldilocks-checklist.md"
