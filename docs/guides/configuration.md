# Configuration Guide

Customize k8s-cost-optimizer behavior through environment variables and flags.

## Environment Variables

### Database Configuration
```bash
# Database connection (required for storage)
export DATABASE_URL="postgresql://user:pass@host:5432/database?sslmode=require"

# Enable/disable storage (default: true)
export STORAGE_ENABLED=true
```

### Metrics Configuration (Future)
```bash
# Prometheus endpoint
export PROMETHEUS_URL="http://prometheus:9090"

# Metrics lookback period (default: 7 days)
export METRICS_DURATION="168h"
```

### Analysis Configuration
```bash
# Safety buffer for recommendations (default: 1.5 = 50% buffer)
export SAFETY_BUFFER="1.5"

# Minimum CPU threshold in millicores (default: 10m)
export MIN_CPU_MILLICORES="10"

# Minimum Memory threshold in bytes (default: 10Mi)
export MIN_MEMORY_BYTES="10485760"
```

## CLI Flags

### Global Flags
```bash
# Namespace to scan
-n, --namespace string

# Scan all namespaces
-A, --all-namespaces

# Output format: text, json, commands
-o, --output string (default "text")

# Save to database
--save

# Cluster identifier for multi-cluster setups
--cluster-id string (default "default")
```

### Examples
```bash
# Scan specific namespace
cost-scan -n production

# Scan all namespaces
cost-scan --all-namespaces

# JSON output for automation
cost-scan -n production -o json

# Just show kubectl commands
cost-scan -n production -o commands

# Save with custom cluster ID
cost-scan -n production --save --cluster-id prod-us-west-2
```

## Multi-Cluster Setup

### Option 1: Multiple kubeconfig contexts
```bash
# Scan cluster 1
kubectl config use-context cluster1
cost-scan -n production --save --cluster-id cluster1

# Scan cluster 2
kubectl config use-context cluster2
cost-scan -n production --save --cluster-id cluster2

# View all recommendations
cost-scan history production  # Shows from all clusters
```

### Option 2: Script for multiple clusters
```bash
#!/bin/bash
# scan-all-clusters.sh

CLUSTERS=("prod-us-east" "prod-us-west" "prod-eu-central")

for cluster in "${CLUSTERS[@]}"; do
    echo "Scanning $cluster..."
    kubectl config use-context "$cluster"
    cost-scan --all-namespaces --save --cluster-id "$cluster"
done
```

## Output Formats

### Text (Human-Readable)
```bash
cost-scan -n production
```

Output:
```
=== Optimization Recommendations ===

1. production/api-service
   Type: RIGHT_SIZE
   Current:  CPU=1000m Memory=2048Mi
   Recommended: CPU=250m Memory=512Mi
   Savings: $45.67/month
   Risk: LOW
   Command: kubectl set resources ...

Total potential savings: $127.83/month
```

### JSON (Machine-Readable)
```bash
cost-scan -n production -o json
```

Output:
```json
{
  "recommendations": [
    {
      "id": "uuid",
      "type": "RIGHT_SIZE",
      "workload": {
        "namespace": "production",
        "deployment": "api-service"
      },
      "current_cpu": 1000,
      "recommended_cpu": 250,
      "savings_monthly": 45.67,
      "risk": "LOW"
    }
  ],
  "total_savings": 127.83,
  "count": 3,
  "timestamp": "2025-11-20T19:30:00Z"
}
```

### Commands Only
```bash
cost-scan -n production -o commands
```

Output:
```bash
kubectl set resources deployment api-service -n production --requests=cpu=250m,memory=512Mi
kubectl set resources deployment worker -n production --requests=cpu=100m,memory=256Mi
kubectl scale deployment idle-service -n production --replicas=0
```

## Cost Calculation

### Default Pricing (Built-in estimates)

Current implementation uses simplified estimates:
- CPU: $0.032 per core-hour (~$23/core/month)
- Memory: $0.004 per GB-hour (~$3/GB/month)

### Custom Pricing (Coming in Week 3)
```bash
# Set custom pricing
export CPU_PRICE_PER_CORE_HOUR="0.025"
export MEMORY_PRICE_PER_GB_HOUR="0.003"
```

## Recommendation Thresholds

Control when recommendations are generated:
```bash
# Only recommend if CPU usage < 50% of request
export CPU_UNDERUTILIZATION_THRESHOLD="0.5"

# Only recommend if savings > $5/month
export MIN_SAVINGS_THRESHOLD="5.0"

# Ignore pods with < 1 day uptime
export MIN_POD_AGE_HOURS="24"
```

## Risk Levels

Recommendations are tagged with risk levels:

- **NONE**: No risk (e.g., scale down idle pods)
- **LOW**: Safe reduction (< 50% reduction)
- **MEDIUM**: Moderate reduction (50-75% reduction)
- **HIGH**: Aggressive reduction (> 75% reduction)

Control risk tolerance:
```bash
# Only show LOW and NONE risk recommendations
export MAX_RISK_LEVEL="LOW"
```

## Integration Examples

### CI/CD Pipeline
```yaml
# .github/workflows/cost-check.yaml
name: Weekly Cost Check
on:
  schedule:
    - cron: '0 9 * * 1'  # Monday 9 AM

jobs:
  cost-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup kubectl
        uses: azure/setup-kubectl@v3
      - name: Configure kubeconfig
        run: |
          echo "${{ secrets.KUBECONFIG }}" > kubeconfig
          export KUBECONFIG=kubeconfig
      - name: Run cost scan
        run: |
          cost-scan --all-namespaces -o json > cost-report.json
      - name: Upload report
        uses: actions/upload-artifact@v3
        with:
          name: cost-report
          path: cost-report.json
```

### Slack Notifications
```bash
#!/bin/bash
# notify-slack.sh

REPORT=$(cost-scan -n production -o json)
SAVINGS=$(echo "$REPORT" | jq -r '.total_savings')

curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -H 'Content-Type: application/json' \
  -d "{\"text\":\"💰 Potential savings: \$$SAVINGS/month\"}"
```

### Prometheus Metrics (Future)
```bash
# Export metrics for Prometheus scraping
cost-scan -n production --export-metrics > /tmp/cost_metrics.prom
```

## Troubleshooting

### High Memory Usage
```bash
# Reduce batch size for large clusters
export SCAN_BATCH_SIZE="50"
```

### Slow Scans
```bash
# Reduce metrics lookback
export METRICS_DURATION="24h"  # Instead of default 7 days

# Skip metrics cache
export SKIP_METRICS_CACHE="true"
```

### Debug Mode
```bash
# Enable verbose logging
export DEBUG="true"
cost-scan -n production
```

---

**Next:** [Multi-Cluster Setup](multi-cluster.md) | [Troubleshooting](troubleshooting.md)
