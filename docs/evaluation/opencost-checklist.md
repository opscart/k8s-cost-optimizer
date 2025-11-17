# OpenCost Evaluation Checklist

**Date:** [Fill in]  
**Version:** [Fill in]  
**Cluster:** minikube (3 nodes, 2CPU/2GB each)

## Installation

- [ ] Installation time: _____ minutes
- [ ] Number of commands: _____
- [ ] Pods deployed: _____
- [ ] Resource usage: _____

## Features Available

After 15 minutes, access http://localhost:9003

### Cost Visibility
- [ ] Can see cost-test namespace?
- [ ] Cost breakdown by deployment?
- [ ] Label-based filtering?
- [ ] Historical data available?

### Missing Features
- [ ] Right-sizing recommendations: YES / NO
- [ ] Waste detection: YES / NO
- [ ] Actionable suggestions: YES / NO
- [ ] Multi-cluster view: YES / NO

## API Testing
```bash
curl http://localhost:9003/allocation/compute?window=1d | jq '.'
```

- [ ] API response format:
- [ ] Ease of parsing:
- [ ] Data accuracy:

## Comparison to Kubecost

Better than Kubecost at:
1. 
2. 

Worse than Kubecost at:
1. 
2. 

## Critical Findings

### What it does well:
1. 
2. 

### What it lacks:
1. 
2. 

### Gap for our tool:
1. 
2. 