# Goldilocks Technical Evaluation

**Date:** November 19, 2025  
**Version:** Latest (via Helm chart fairwinds-stable/goldilocks)  
**Environment:** Minikube 3-node cluster

## Installation

**Time:** ~15 minutes  
**Dependencies:** VPA (Vertical Pod Autoscaler) - Required  
**Method:** Git clone + shell script for VPA, then Helm for Goldilocks

**Setup Steps:**
1. Clone kubernetes/autoscaler repository
2. Run VPA install script: `./hack/vpa-up.sh`
3. Install Goldilocks via Helm
4. Label namespaces to enable: `kubectl label namespace cost-test goldilocks.fairwinds.com/enabled=true`

**Complexity:** Medium - VPA adds significant complexity

## Features Tested

### Right-Sizing Recommendations ✓

**Provides specific CPU/memory values:**
- Based on VPA (Vertical Pod Autoscaler) analysis
- Two QoS profiles: Guaranteed and Burstable
- YAML snippets ready to copy/paste

**Example recommendations observed:**
```yaml
overprovision-app:
  Current:      CPU 1000m, Memory 2Gi
  Recommended:  CPU 25m,   Memory 250Mi
  Reduction:    40x CPU,   8x Memory

idle-app:
  Current:      CPU 500m,  Memory 512Mi
  Recommended:  CPU 25m,   Memory 250Mi
  Reduction:    20x CPU,   2x Memory

normal-app:
  Current:      CPU 100m,  Memory 128Mi
  Recommended:  CPU 25m,   Memory 250Mi
  (Modest adjustment)
```

**Provides YAML:**
```yaml
resources:
  requests:
    cpu: 25m
    memory: 250Mi
  limits:
    cpu: 25m
    memory: 250Mi
```

### What It Doesn't Provide

- No cost attribution (no dollar amounts)
- No waste detection (unused PVCs, orphaned resources)
- No kubectl commands (only YAML snippets)
- No savings estimates
- No priority scoring

## Scope & Limitations

**Single-cluster only:** Per-cluster deployment required  
**VPA-based only:** Pure visualization of VPA recommendations  
**Namespace-scoped:** Must enable per namespace with labels  
**Right-sizing only:** No other optimization types

## API Access

**Limited/Indirect API:**

Goldilocks doesn't provide a REST API. Access recommendations via Kubernetes API:
```bash
# Get all VPA recommendations
kubectl get vpa -n cost-test -o json

# Get specific workload
kubectl get vpa <vpa-name> -n cost-test -o json

# Example response structure:
{
  "status": {
    "recommendation": {
      "containerRecommendations": [{
        "containerName": "nginx",
        "lowerBound": {"cpu": "25m", "memory": "250Mi"},
        "target": {"cpu": "25m", "memory": "250Mi"},
        "upperBound": {"cpu": "3178m", "memory": "3324Mi"}
      }]
    }
  }
}
```

## Technical Observations

**Strengths:**
- Very specific CPU/memory values
- YAML snippets provided
- VPA-based (uses actual Kubernetes autoscaler)
- Found significant over-provisioning (40x reduction detected)

**Limitations:**
- Requires VPA installation (adds complexity)
- No cost context or $ savings
- Basic UI (simple lists)
- No executable commands
- Manual YAML editing required

## Comparison to Other Tools

| Feature | OpenCost | Kubecost | Goldilocks |
|---------|----------|----------|------------|
| Cost visibility | ✅ | ✅ | ❌ |
| Generic recommendations | ❌ | ✅ | ❌ |
| Specific CPU/memory values | ❌ | ❌ | ✅ |
| YAML snippets | ❌ | ❌ | ✅ |
| kubectl commands | ❌ | ❌ | ❌ |

**Key Difference:** Goldilocks provides specific values; others provide insights or generic recommendations.

## Screenshots

Saved to `docs/evaluation/screenshots/goldilocks/`:
- Dashboard namespace view
- Workload-specific recommendations
- YAML snippet examples

## Technical Assessment

Goldilocks serves as a VPA visualization tool. It bridges the gap between generic recommendations (like Kubecost) and specific values, but still requires manual YAML editing and doesn't provide cost context or executable commands.

The VPA dependency adds installation complexity, making it less suitable for quick scans or ad-hoc analysis.
