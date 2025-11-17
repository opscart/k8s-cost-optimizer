#!/bin/bash
# Clean up everything

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${RED}==>${NC} This will delete your minikube cluster and all data"
read -p "Are you sure? (y/N) " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

echo -e "${GREEN}==>${NC} Deleting minikube cluster..."
minikube delete

echo -e "${GREEN}✓${NC} Cleanup complete"