# Kubecost Technical Evaluation

**Date:** November 19, 2025  
**Version:** 2.4.2  
**Environment:** Minikube 3-node cluster

## Installation

**Time:** ~10 minutes  
**Method:** Helm chart with custom configuration  
**Resources:** 2 main pods (~500MB RAM, 0.3 CPU total)

**Installation Challenges:**
- Bundled Prometheus had storage permission issues
- Grafana deployment failed (Chainguard images require payment)
- Required pointing to existing Prometheus instance

**Working Configuration:**
```bash
helm install kubecost kubecost/cost-analyzer \
    --namespace kubecost \
    --create-namespace \
    --version 2.4.2 \
    --set global.prometheus.fqdn=http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090 \
    --set persistentVolume.enabled=false \
    --set kubecostModel.etlStorePath=/tmp/kubecost-data
```

## Features Tested

### Cost Visibility ✓
- Namespace-level cost breakdown
- Deployment-level attribution
- Time-series visualization
- Cumulative and rate-based views
- Multi-dimensional filtering

### Optimization Recommendations ✓
**10+ recommendation types available:**
1. Right-size cluster nodes
2. Right-size container requests
3. Manage abandoned workloads
4. Manage unclaimed volumes
5. Manage underutilized nodes (showed $437/mo potential savings)
6. Reserved instance recommendations
7. Manage local disks
8. Manage orphaned resources
9. Spot Commander
10. Right-size persistent volumes

**Estimated savings shown:** $290.25/month

### Asset Management ✓
- Node utilization tracking
- Cluster efficiency metrics
- Resource allocation visibility

### Actions/Automation ⚠️
Available but requires cluster controller installation:
- Cluster Turndown
- Request Sizing
- Namespace Turndown

## Free Tier Limitations

**Confirmed limits:**
- **Core limit:** 250 cores maximum
- **Data retention:** 15 days
- **Multi-cluster:** Separate installations required

## Technical Observations

**Strengths:**
- Clean, modern UI
- Comprehensive recommendations with dollar estimates
- Multi-cloud support (AWS, Azure, GCP)
- Active development (IBM backing)
- Good documentation

**Challenges:**
- Installation complexity (Helm, Prometheus dependencies)
- Higher resource usage than OpenCost (~500MB vs ~200MB)
- Recommendations are generic (no specific kubectl commands)
- Efficiency metric showed 0.0% (may need real billing data)

## Integration Points

**Works with:**
- Prometheus (required for metrics)
- Cloud provider billing APIs (AWS, Azure, GCP)
- kubectl plugin available (kubectl-cost)

**API Access:**
- REST API at port 9090
- Allocation API: `/model/allocation`
- Savings API: `/model/savings`
- Documentation: https://docs.kubecost.com/apis

## Comparison to OpenCost

| Feature | OpenCost | Kubecost |
|---------|----------|----------|
| Cost visibility | ✅ | ✅ |
| Right-sizing recommendations | ❌ | ✅ |
| Waste detection | ❌ | ✅ |
| Installation complexity | Low | Medium |
| Resource usage | ~200MB | ~500MB |
| Core limit | None | 250 |
| Cost | Free | Free <250 cores |

**Key Difference:** Kubecost provides optimization recommendations with savings estimates; OpenCost provides only cost visibility.

## Screenshots

Saved to `docs/evaluation/screenshots/kubecost/`:
- Overview dashboard - kubecost-savings-page.png
- Savings recommendations page - kubecost-overview.png
- Allocations breakdown - kubecost-allocations.png
- Actions interface - kubecost-actions-page.png

## Technical Assessment

Kubecost extends beyond cost visibility into optimization territory. The recommendations engine identifies opportunities across multiple categories (right-sizing, waste, spot instances, etc.) with estimated savings impact.

However, recommendations remain high-level without specific implementation steps. Users must manually investigate which resources to modify and determine appropriate values.

The architecture requires Prometheus for metrics collection and benefits from cloud billing integration for accurate cost reconciliation.
