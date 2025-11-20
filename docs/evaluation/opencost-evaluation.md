# OpenCost Technical Evaluation

**Date:** November 19, 2025  
**Version:** 1.118.0  
**Environment:** Minikube 3-node cluster

## Installation

**Time:** ~3 minutes  
**Method:** Helm chart with Prometheus endpoint configuration  
**Resources:** 1 pod (2 containers), minimal overhead

## Features Tested

### Cost Visibility ✓
- Namespace-level cost breakdown
- Multiple aggregation dimensions (Cluster, Node, Namespace, Deployment, Pod, Container)
- Time-series visualization
- Historical data (configurable window)
- Cost breakdown by resource type (CPU, RAM, Storage)

### API Access ✓
- RESTful API available on port 9003
- JSON response format
- Example query: `/allocation/compute?window=1d&aggregate=namespace`
- Well-documented endpoints

### UI Features ✓
- Clean, intuitive interface on port 9090
- Interactive charts and tables
- Flexible filtering and grouping
- Date range selection
- Export capabilities

## Technical Observations

**Strengths:**
- Lightweight deployment (single pod)
- Fast data collection (15 minutes to first results)
- Flexible aggregation options
- Clean API design
- CNCF sandbox project with active development

**Limitations Observed:**
- Cost visibility only (no optimization recommendations)
- Single cluster view in standard deployment
- Requires Prometheus for metrics
- Default pricing models for non-cloud environments

## Integration Points

**Works with:**
- Prometheus (required)
- Cloud provider billing APIs (optional)
- External cost sources (configurable)

## API Response Example
```json
{
  "code": 200,
  "data": [{
    "kube-system": {
      "name": "kube-system",
      "properties": {
        "cluster": "minikube-test",
        "namespace": "kube-system"
      },
      "cpuCores": 1.15,
      "cpuCost": 0.01,
      "ramBytes": 2147483648,
      "ramCost": 0.005,
      ...
    }
  }]
}
```

## Technical Assessment

OpenCost provides a solid foundation for Kubernetes cost attribution. It excels at:
- Accurate cost calculation and allocation
- Flexible data aggregation
- Clean API for programmatic access

The architecture is modular and could serve as a data source for higher-level optimization tools.

## Setup Notes

Requires Prometheus endpoint configuration:
```bash
kubectl set env deployment/opencost -n opencost \
  PROMETHEUS_SERVER_ENDPOINT=http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090
```
## OpenCost - Key Facts

**Status:** CNCF Incubating Project (October 2024)  
**License:** Apache 2.0 - Fully open source, free forever  
**Cost:** $0 (no paid version, no enterprise trap)  
**Multi-cluster:** Not native, requires custom setup  
**Production Adoption:** Grafana Labs, AWS, Google, Adobe, IBM

**Cluster Name:** Shows as "minikube-test" (set via Helm config, not from kubectl context)

**What It Does:** Cost visibility and attribution  
**What It Doesn't:** Optimization recommendations, waste detection