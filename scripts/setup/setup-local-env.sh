#!/bin/bash
# Setup local minikube environment for testing k8s cost tools
# Usage: ./scripts/setup/setup-local-env.sh

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}==>${NC} $1"
}

print_error() {
    echo -e "${RED}ERROR:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}WARNING:${NC} $1"
}

# Configuration
MINIKUBE_NODES=${MINIKUBE_NODES:-3}
MINIKUBE_CPUS=${MINIKUBE_CPUS:-2}
MINIKUBE_MEMORY=${MINIKUBE_MEMORY:-2048}
MINIKUBE_DRIVER=${MINIKUBE_DRIVER:-docker}

print_status "Starting Minikube cluster setup..."
echo "Configuration:"
echo "  Nodes: $MINIKUBE_NODES"
echo "  CPUs per node: $MINIKUBE_CPUS"
echo "  Memory per node: ${MINIKUBE_MEMORY}MB"
echo "  Driver: $MINIKUBE_DRIVER"
echo ""

# Check if minikube is already running
if minikube status &> /dev/null; then
    print_warning "Minikube is already running"
    read -p "Do you want to delete and recreate? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Deleting existing minikube cluster..."
        minikube delete
    else
        print_status "Using existing cluster"
        exit 0
    fi
fi

# Start minikube
print_status "Starting Minikube with $MINIKUBE_NODES nodes..."
minikube start \
    --nodes=$MINIKUBE_NODES \
    --cpus=$MINIKUBE_CPUS \
    --memory=$MINIKUBE_MEMORY \
    --driver=$MINIKUBE_DRIVER

# Wait for cluster to be ready
print_status "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s

# Install metrics-server
print_status "Installing Metrics Server..."
kubectl apply -f - <<METRICS_EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
  name: system:aggregated-metrics-reader
rules:
- apiGroups:
  - metrics.k8s.io
  resources:
  - pods
  - nodes
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
rules:
- apiGroups:
  - ""
  resources:
  - nodes/metrics
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - pods
  - nodes
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:metrics-server
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: https
  selector:
    k8s-app: metrics-server
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: metrics-server
  strategy:
    rollingUpdate:
      maxUnavailable: 0
  template:
    metadata:
      labels:
        k8s-app: metrics-server
    spec:
      containers:
      - args:
        - --cert-dir=/tmp
        - --secure-port=10250
        - --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
        - --kubelet-use-node-status-port
        - --metric-resolution=15s
        - --kubelet-insecure-tls
        image: registry.k8s.io/metrics-server/metrics-server:v0.7.1
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /livez
            port: https
            scheme: HTTPS
          periodSeconds: 10
        name: metrics-server
        ports:
        - containerPort: 10250
          name: https
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readyz
            port: https
            scheme: HTTPS
          initialDelaySeconds: 20
          periodSeconds: 10
        resources:
          requests:
            cpu: 100m
            memory: 200Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          seccompProfile:
            type: RuntimeDefault
        volumeMounts:
        - mountPath: /tmp
          name: tmp-dir
      nodeSelector:
        kubernetes.io/os: linux
      priorityClassName: system-cluster-critical
      serviceAccountName: metrics-server
      volumes:
      - emptyDir: {}
        name: tmp-dir
---
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  labels:
    k8s-app: metrics-server
  name: v1beta1.metrics.k8s.io
spec:
  group: metrics.k8s.io
  groupPriorityMinimum: 100
  insecureSkipTLSVerify: true
  service:
    name: metrics-server
    namespace: kube-system
  version: v1beta1
  versionPriority: 100
METRICS_EOF

print_status "Waiting for Metrics Server to be ready..."
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=120s
sleep 15

# Verify metrics server
print_status "Verifying Metrics Server..."
if kubectl top nodes &> /dev/null; then
    print_status "✓ Metrics Server is working"
    kubectl top nodes
else
    print_error "Metrics Server is not working properly"
    exit 1
fi

print_status "Creating HPA-enabled workload for testing..."

kubectl apply -f - <<EOF
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
EOF

print_status "✓ HPA-enabled workload created"

# Install Prometheus stack
print_status "Installing Prometheus Stack..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
    --namespace monitoring \
    --create-namespace \
    --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
    --set prometheus.prometheusSpec.retention=7d \
    --set prometheus.prometheusSpec.resources.requests.memory=512Mi \
    --set prometheus.prometheusSpec.resources.requests.cpu=250m \
    --set grafana.enabled=true \
    --wait \
    --timeout 5m

print_status "Waiting for Prometheus to be ready..."
kubectl wait --for=condition=ready pod \
    -l app.kubernetes.io/name=prometheus \
    -n monitoring \
    --timeout=300s

print_status "Configuring Prometheus to scrape all namespaces..."

# Patch kubelet ServiceMonitor to include common namespaces
for ns in "cost-test" "default" "monitoring"; do
    kubectl patch servicemonitor prometheus-kube-prometheus-kubelet -n monitoring \
        --type='json' \
        -p="[{\"op\": \"add\", \"path\": \"/spec/namespaceSelector/matchNames/-\", \"value\": \"$ns\"}]" \
        2>/dev/null || true
done

# Configure Prometheus to auto-discover all ServiceMonitors/PodMonitors
kubectl patch prometheus prometheus-kube-prometheus-prometheus -n monitoring --type merge -p '
spec:
  podMonitorNamespaceSelector: {}
  serviceMonitorNamespaceSelector: {}
' 2>/dev/null || true

print_status "✓ Prometheus configured for multi-namespace scraping"

print_status "✓ Environment setup complete!"
echo ""
echo "Cluster Information:"
kubectl get nodes
echo ""
echo "Monitoring Stack:"
kubectl get pods -n monitoring
echo ""
echo "Next Steps:"
echo "  1. Deploy test workloads: ./scripts/setup/deploy-test-workloads.sh"
echo "  2. Install evaluation tools: ./scripts/evaluation/install-kubecost.sh"
echo ""
echo "Access Prometheus:"
echo "  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090"
echo "  Open: http://localhost:9090"
echo ""
echo "Access Grafana:"
echo "  kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
echo "  Open: http://localhost:3000 (admin/prom-operator)"
