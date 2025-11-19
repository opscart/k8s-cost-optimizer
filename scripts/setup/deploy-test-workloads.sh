#!/bin/bash
# Deploy test workloads for evaluating cost optimization tools

set -e

GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}==>${NC} Creating namespace..."
kubectl apply -f examples/test-workloads/namespace.yaml

echo -e "${GREEN}==>${NC} Deploying test workloads..."
kubectl apply -f examples/test-workloads/

echo ""
echo -e "${GREEN}==>${NC} Test workloads deployed successfully!"
echo ""
echo "Deployed workloads:"
kubectl get deployments,pvc -n cost-test
echo ""
echo "Wait 5-10 minutes for metrics to accumulate, then run evaluation scripts."