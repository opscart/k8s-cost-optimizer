#!/bin/bash
set -e

print_status() {
    echo ""
    echo "===> $1"
    echo ""
}

print_status "Creating multi-environment namespaces..."

# Production namespace
kubectl create namespace prod-api --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace prod-api environment=production tier=prod --overwrite

# Staging namespace
kubectl create namespace staging-api --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace staging-api environment=staging tier=staging --overwrite

# Development namespace
kubectl create namespace dev-test --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace dev-test environment=development tier=dev --overwrite

print_status "Deploying workloads to production namespace..."

kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
  namespace: prod-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: payment
  template:
    metadata:
      labels:
        app: payment
    spec:
      containers:
      - name: app
        image: nginx:alpine
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-prod
  namespace: prod-api
spec:
  serviceName: redis
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
YAML

print_status "Deploying workloads to staging namespace..."

kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-staging
  namespace: staging-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: app
        image: nginx:alpine
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 400m
            memory: 512Mi
YAML

print_status "Deploying workloads to dev namespace..."

kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: dev-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app
        image: nginx:alpine
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
YAML

print_status "✓ Multi-environment workloads deployed!"
echo ""
echo "Namespaces created:"
kubectl get namespaces --show-labels | grep -E "(prod-api|staging-api|dev-test)"
echo ""
echo "Workloads per environment:"
echo "Production:"
kubectl get all -n prod-api
echo ""
echo "Staging:"
kubectl get all -n staging-api
echo ""
echo "Development:"
kubectl get all -n dev-test
