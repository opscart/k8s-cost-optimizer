#!/bin/bash
# Install and test OpenCost

set -e

GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Installing OpenCost..."

kubectl apply --namespace opencost \
    -f https://raw.githubusercontent.com/opencost/opencost/develop/kubernetes/opencost.yaml

echo -e "${GREEN}==>${NC} Waiting for OpenCost to be ready..."
kubectl wait --for=condition=ready pod \
    -l app=opencost \
    -n opencost \
    --timeout=300s

echo ""
echo -e "${GREEN}✓${NC} OpenCost installed successfully!"
echo ""
echo "Access OpenCost UI:"
echo "  kubectl port-forward -n opencost svc/opencost 9003:9003"
echo "  Open: http://localhost:9003"
echo ""
echo "Test OpenCost API:"
echo "  curl http://localhost:9003/allocation/compute?window=1d"
echo ""
echo "Evaluation checklist: docs/evaluation/opencost-checklist.md"