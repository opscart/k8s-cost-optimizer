# Kubecost Evaluation Checklist

**Date:** [Fill in]  
**Version:** [Fill in]  
**Cluster:** minikube (3 nodes, 2CPU/2GB each)

## Installation

- [ ] Installation time: _____ minutes
- [ ] Number of commands: _____
- [ ] Pods deployed: _____ (check with `kubectl get pods -n kubecost`)
- [ ] Resource usage: _____ (check with `kubectl top pods -n kubecost`)

## Features Available (Free Tier)

After 15 minutes, access http://localhost:9090

### Cost Allocation
- [ ] Can see cost-test namespace costs?
- [ ] Can break down by deployment?
- [ ] Can filter by team label?
- [ ] Are costs realistic or estimated?
- [ ] Screenshot: Save to docs/evaluation/screenshots/

### Savings/Optimization
- [ ] Are there right-sizing recommendations?
- [ ] Does it identify overprovision-app as over-provisioned?
- [ ] Does it flag idle-app as wasteful?
- [ ] Are recommendations actionable?
- [ ] What specific actions does it suggest?

### Asset Management
- [ ] Can see unused-storage PVC?
- [ ] Shows node utilization?
- [ ] Identifies waste accurately?

### What's Paywalled?
- [ ] Multi-cluster features locked?
- [ ] Long-term retention limited?
- [ ] Advanced reports unavailable?
- [ ] List other limitations:

## API Access

Test these:
```bash
# Get allocation data
curl -s http://localhost:9090/model/allocation \
  -d window=1d \
  -d aggregate=namespace | jq '.'

# Get savings
curl -s http://localhost:9090/model/savings | jq '.'
```

- [ ] API works?
- [ ] Data format usable?
- [ ] Documentation clear?

## User Experience

Rate 1-10:
- UI clarity: _____/10
- Time to first insight: _____ minutes
- Ease of understanding: _____/10
- Documentation quality: _____/10

## Critical Findings

### Strengths
1. 
2. 
3. 

### Weaknesses
1. 
2. 
3. 

### Dealbreakers for Our Use Case
1. 
2. 

## Would This Solve Our Problem?

- [ ] Yes, completely
- [ ] Partially (explain):
- [ ] No (explain):

## Our Tool Must Do Better At:
1. 
2. 
3. 
EOF
