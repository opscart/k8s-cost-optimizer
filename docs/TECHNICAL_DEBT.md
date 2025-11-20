# Technical Debt Tracker

## Active Technical Debt

### TD-001: Dual Model System (Converter)
**Status:** Active  
**Created:** Week 2, Day 2 (2025-11-20)  
**Priority:** Medium  
**Effort:** 4-6 hours  
**Target:** Week 4, Day 1

**Problem:**
- Two recommendation model systems coexist
- `pkg/recommender/Recommendation` (old)
- `pkg/models/Recommendation` (new)
- `pkg/converter/` bridges them

**Why Deferred:**
- Allows faster Week 2-3 progress
- Low risk (converter works fine)
- Better to migrate with full context

**Migration Plan:**
1. Update scanner to use `models.Workload`
2. Update analyzer to use `models.*`
3. Rewrite recommender to output `models.Recommendation`
4. Update executor to use `models.Recommendation`
5. Remove converter package
6. Update all tests

**Files to Change:**
- pkg/scanner/scanner.go
- pkg/analyzer/analyzer.go
- pkg/recommender/recommender.go
- pkg/executor/commands.go
- cmd/cost-scan/main.go
- DELETE: pkg/converter/

**Test Checklist:**
- [ ] Scanner produces correct models.Recommendation
- [ ] Storage saves/retrieves correctly
- [ ] CLI commands work unchanged
- [ ] All unit tests pass
- [ ] Integration test: scan → save → retrieve

**Decision Date:** 2025-11-20  
**Decision Maker:** Shamsher  
**Review Date:** Week 4, Day 1

---

## Retired Technical Debt

(Empty for now)

---

## Guidelines

**When to Accept Technical Debt:**
- Speeds up critical path by >50%
- Risk is low (converter pattern is safe)
- Can be paid back in <1 week effort
- Doesn't block other features

**When to Reject Technical Debt:**
- Security implications
- Data loss risk
- Blocks future features
- Effort to fix later >2x effort now

**Review Cadence:**
- Weekly review during standup
- Mandatory review before release
- Update status after each sprint
