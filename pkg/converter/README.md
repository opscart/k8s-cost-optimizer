# Converter Package

## Purpose
Bridge between old `pkg/recommender` models and new `pkg/models` models.

## ⚠️ TECHNICAL DEBT NOTICE

**This package is temporary and should be removed in Week 4.**

See: `docs/TECHNICAL_DEBT.md#TD-001`

**Target Migration Date:** Week 4, Day 1  
**Effort:** 4-6 hours  
**Files to Migrate:** scanner, analyzer, recommender, executor

## Why This Exists
- Allows Week 2-3 progress without rewriting existing code
- PostgreSQL storage needs `models.Recommendation`
- Existing scanner produces `recommender.Recommendation`

## Migration Checklist
When ready to remove this:
- [ ] Update scanner to produce `models.Workload`
- [ ] Update analyzer to work with `models.*`
- [ ] Rewrite recommender to output `models.Recommendation`
- [ ] Update executor to accept `models.Recommendation`
- [ ] Remove this converter package
- [ ] Update all tests
- [ ] Update CLI

## Usage
```go
// Convert old to new
oldRec := recommender.Analyze(...)
newRec := converter.OldToNew(oldRec, clusterID)
store.SaveRecommendation(ctx, newRec)
```
