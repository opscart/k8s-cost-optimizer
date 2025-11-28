#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_status() {
    echo ""
    echo "===> $1"
    echo ""
}

print_status "Deploying advanced workloads (StatefulSets, DaemonSets, HPAs)..."

# StatefulSet - PostgreSQL
kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres-test
  namespace: cost-test
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 1000m
            memory: 2Gi
        env:
        - name: POSTGRES_PASSWORD
          value: testpass
YAML

# DaemonSet - Node Exporter
kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: cost-test
spec:
  selector:
    matchLabels:
      app: node-exporter
  template:
    metadata:
      labels:
        app: node-exporter
    spec:
      containers:
      - name: node-exporter
        image: prom/node-exporter:latest
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
YAML

# HPA-enabled Deployment
kubectl apply -f - <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-autoscaled
  namespace: cost-test
  labels:
    app: api-autoscaled
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-autoscaled
  template:
    metadata:
      labels:
        app: api-autoscaled
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-autoscaled-hpa
  namespace: cost-test
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-autoscaled
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
YAML

print_status "✓ Advanced workloads deployed!"
kubectl get all -n cost-test
