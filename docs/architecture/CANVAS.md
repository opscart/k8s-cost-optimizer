# K8s Cost Optimizer - Architecture Canvas

**Last Updated:** November 20, 2025  
**Scope:** 8-week MVP with extensible architecture  
**Developer:** Solo (you)

---

## 🎯 Core Philosophy: Build to Extend, Not Rebuild

```
┌────────────────────────────────────────────────────────────────┐
│  LAYERED ARCHITECTURE - Add features without rewriting         │
└────────────────────────────────────────────────────────────────┘

Layer 1: Data Collection (Week 1-2)
┌──────────────────────────────────────────────────────────────┐
│  INTERFACE: DataSource                                        │
│  ├── PrometheusSource (MVP)                                  │
│  ├── MetricsServerSource (fallback)                          │
│  └── [Future] CloudWatchSource, DatadogSource                │
│                                                               │
│  WHY: Abstract data source, swap implementations later       │
└──────────────────────────────────────────────────────────────┘

Layer 2: Storage (Week 2-3)
┌──────────────────────────────────────────────────────────────┐
│  INTERFACE: Store                                             │
│  ├── SQLiteStore (MVP - single user)                         │
│  ├── [Future] PostgresStore (multi-user)                     │
│  └── [Future] S3Store (big data)                             │
│                                                               │
│  WHY: Start simple (SQLite), upgrade later without rewrite   │
└──────────────────────────────────────────────────────────────┘

Layer 3: Recommendation Engine (Week 3-4)
┌──────────────────────────────────────────────────────────────┐
│  INTERFACE: Recommender                                       │
│  ├── StatisticalRecommender (MVP - P95/P99)                 │
│  ├── [Future] BinPackingRecommender (simulation)            │
│  └── [Future] MLRecommender (patterns)                       │
│                                                               │
│  WHY: Plug in better algorithms without changing CLI         │
└──────────────────────────────────────────────────────────────┘

Layer 4: Cost Calculation (Week 5-6)
┌──────────────────────────────────────────────────────────────┐
│  INTERFACE: PricingProvider                                   │
│  ├── AzurePricingProvider (MVP)                             │
│  ├── [Future] AWSPricingProvider                            │
│  └── [Future] GCPPricingProvider                            │
│                                                               │
│  WHY: Add clouds without touching core logic                 │
└──────────────────────────────────────────────────────────────┘

Layer 5: Output (Week 7-8)
┌──────────────────────────────────────────────────────────────┐
│  INTERFACE: OutputHandler                                     │
│  ├── CLIOutput (MVP)                                         │
│  ├── [Future] APIOutput (REST)                              │
│  └── [Future] WebUIOutput (dashboard)                       │
│                                                               │
│  WHY: Add interfaces without changing recommendation engine  │
└──────────────────────────────────────────────────────────────┘
```

---

## 📦 Project Structure - Extensible Design

```
k8s-cost-optimizer/
├── pkg/
│   ├── datasource/              # Layer 1: INTERFACE + implementations
│   │   ├── datasource.go        # Interface definition
│   │   ├── prometheus.go        # MVP implementation
│   │   ├── metricsserver.go     # Fallback
│   │   └── [future] cloudwatch.go
│   │
│   ├── storage/                 # Layer 2: INTERFACE + implementations
│   │   ├── store.go             # Interface definition
│   │   ├── sqlite.go            # MVP implementation
│   │   └── [future] postgres.go
│   │
│   ├── recommender/             # Layer 3: INTERFACE + implementations
│   │   ├── recommender.go       # Interface definition
│   │   ├── statistical.go       # MVP: P95/P99 based
│   │   └── [future] binpacking.go
│   │
│   ├── pricing/                 # Layer 4: INTERFACE + implementations
│   │   ├── provider.go          # Interface definition
│   │   ├── azure.go             # MVP implementation
│   │   ├── cache.go             # In-memory cache (all providers)
│   │   └── [future] aws.go
│   │
│   ├── output/                  # Layer 5: INTERFACE + implementations
│   │   ├── handler.go           # Interface definition
│   │   ├── cli.go               # MVP implementation
│   │   └── [future] api.go
│   │
│   ├── executor/                # Command execution (keep from Day 1-2)
│   │   └── commands.go
│   │
│   └── models/                  # Shared data types
│       ├── recommendation.go
│       ├── workload.go
│       └── cost.go
│
├── cmd/
│   └── cost-scan/               # CLI entry point (keep from Day 1-2)
│
└── internal/                    # Private helpers
    ├── config/
    └── utils/
```

---

## 🔌 Interface Definitions (Write Once, Extend Forever)

### DataSource Interface
```go
type DataSource interface {
    // Get P95 CPU usage over duration
    GetP95CPU(ctx context.Context, workload Workload, duration time.Duration) (float64, error)
    
    // Get P99 Memory usage over duration
    GetP99Memory(ctx context.Context, workload Workload, duration time.Duration) (int64, error)
    
    // Get usage timeseries (for graphing trends)
    GetTimeseries(ctx context.Context, workload Workload, duration time.Duration) ([]Sample, error)
    
    // Health check
    IsAvailable(ctx context.Context) bool
}

// MVP: PrometheusSource implements this
// Future: Add CloudWatchSource, DatadogSource without changing callers
```

### Store Interface
```go
type Store interface {
    // Save recommendation
    SaveRecommendation(ctx context.Context, rec *Recommendation) error
    
    // Get recommendations by namespace
    GetRecommendations(ctx context.Context, namespace string) ([]*Recommendation, error)
    
    // Audit log
    LogAction(ctx context.Context, action *AuditEntry) error
    
    // Close connection
    Close() error
}

// MVP: SQLiteStore implements this
// Future: PostgresStore just implements same interface
```

### Recommender Interface
```go
type Recommender interface {
    // Analyze workload and generate recommendation
    Analyze(ctx context.Context, workload *Workload, metrics *Metrics) (*Recommendation, error)
    
    // Calculate risk score
    CalculateRisk(metrics *Metrics) RiskLevel
}

// MVP: StatisticalRecommender (P95 + buffer)
// Future: BinPackingRecommender, MLRecommender
```

### PricingProvider Interface
```go
type PricingProvider interface {
    // Get cost per core-hour
    GetCPUCost(ctx context.Context, region string, nodeType string) (float64, error)
    
    // Get cost per GB-hour
    GetMemoryCost(ctx context.Context, region string, nodeType string) (float64, error)
    
    // Get storage cost per GB-month
    GetStorageCost(ctx context.Context, region string, storageType string) (float64, error)
}

// MVP: AzurePricingProvider
// Future: AWSPricingProvider, GCPPricingProvider
```

---

## 🚀 8-Week Implementation Plan

### Week 1-2: Data Collection Layer
```yaml
Goal: Prometheus integration with fallback

Tasks:
  - Define DataSource interface (4 hours)
  - Implement PrometheusSource (2 days)
  - Implement MetricsServerSource fallback (1 day)
  - Write Prometheus recording rules (1 day)
  - Test both implementations (1 day)

Deliverable:
  - pkg/datasource/datasource.go (interface)
  - pkg/datasource/prometheus.go (MVP)
  - pkg/datasource/metricsserver.go (fallback)

Future extensibility:
  - Add CloudWatch: implement DataSource interface
  - Add Datadog: implement DataSource interface
  - NO core changes needed
```

### Week 3: Storage Layer
```yaml
Goal: Persistent storage with simple start

Tasks:
  - Define Store interface (2 hours)
  - Implement SQLiteStore (2 days)
  - Schema design (3 tables: metrics, recommendations, audit) (1 day)
  - Test CRUD operations (1 day)

Deliverable:
  - pkg/storage/store.go (interface)
  - pkg/storage/sqlite.go (MVP)
  - migrations/ directory

Future extensibility:
  - Add PostgreSQL: implement Store interface
  - Add S3: implement Store interface
  - CLI flag: --storage=sqlite|postgres
```

### Week 4: Recommendation Engine
```yaml
Goal: Smart P95/P99 based recommendations

Tasks:
  - Define Recommender interface (2 hours)
  - Implement StatisticalRecommender (3 days)
  - Risk scoring (LOW/MEDIUM/HIGH) (1 day)
  - Unit tests (1 day)

Deliverable:
  - pkg/recommender/recommender.go (interface)
  - pkg/recommender/statistical.go (MVP)

Future extensibility:
  - Add BinPackingRecommender: implement Recommender interface
  - Add MLRecommender: implement Recommender interface
  - CLI flag: --recommender=statistical|binpacking|ml
```

### Week 5-6: Pricing & Cost Calculation
```yaml
Goal: Accurate Azure costs with caching

Tasks:
  - Define PricingProvider interface (2 hours)
  - Implement AzurePricingProvider (2 days)
  - In-memory cache with 24h TTL (1 day)
  - Cost calculation per pod/namespace (2 days)
  - Test with real Azure API (1 day)
  - Multi-cluster support (3-5 clusters) (2 days)

Deliverable:
  - pkg/pricing/provider.go (interface)
  - pkg/pricing/azure.go (MVP)
  - pkg/pricing/cache.go (all providers use this)

Future extensibility:
  - Add AWS: implement PricingProvider interface
  - Add GCP: implement PricingProvider interface
  - Custom pricing: implement PricingProvider interface
```

### Week 7: Safety & Audit
```yaml
Goal: Production-safe with audit trail

Tasks:
  - --apply flag with confirmations (2 days)
  - Dry-run validation (1 day)
  - Audit logging to Store (1 day)
  - Rollback command generation (1 day)

Deliverable:
  - Safe apply with multiple confirmations
  - All actions logged to audit table
  - Rollback commands generated
```

### Week 8: Polish & Release
```yaml
Goal: v1.0.0 production release

Tasks:
  - Unit tests (>70% coverage) (2 days)
  - Integration tests (real cluster) (1 day)
  - Documentation (README, guides) (1 day)
  - GitHub Actions CI (1 day)

Deliverable:
  - v1.0.0 release
  - Homebrew tap
  - Docker image
```

---

## 🎯 What We Build Now (MVP)

```
┌─────────────────────────────────────────────────────────────────┐
│  MVP STACK (Week 1-8)                                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Data: PrometheusSource + MetricsServerSource (fallback)       │
│  Storage: SQLiteStore                                           │
│  Recommendations: StatisticalRecommender (P95/P99)             │
│  Pricing: AzurePricingProvider + in-memory cache               │
│  Output: CLI (text, json, commands)                            │
│  Clusters: 3-5 cluster support                                 │
│  Safety: Audit log, confirmations, rollback                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔮 What We Can Add Later (No Rewrite)

```
┌─────────────────────────────────────────────────────────────────┐
│  V2.0 ADDITIONS (Just implement interfaces)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Data: CloudWatchSource, DatadogSource                         │
│  Storage: PostgresStore, S3Store                               │
│  Recommendations: BinPackingRecommender, MLRecommender         │
│  Pricing: AWSPricingProvider, GCPPricingProvider               │
│  Output: APIOutput (REST), WebUIOutput (dashboard)             │
│  Clusters: 100+ with federation                                │
│  Security: RBAC, OIDC                                          │
│                                                                  │
│  EFFORT: 1-2 weeks per feature (not months)                    │
│  WHY: Interfaces don't change, just add implementations        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 💡 Key Architectural Decisions

### 1. Interface-Based Design
**Decision:** Every layer has an interface  
**Rationale:** Add implementations without changing callers  
**Example:** Start with SQLite, add Postgres later (2 days, not 2 weeks)

### 2. Dependency Injection
**Decision:** Inject implementations via constructors  
**Rationale:** Easy to test, easy to swap  
**Example:**
```go
// Easy to test
recommender := statistical.New(mockDataSource, mockStore)

// Easy to swap
recommender := binpacking.New(realDataSource, realStore)
```

### 3. Single Responsibility
**Decision:** Each package does ONE thing  
**Rationale:** Change one thing without breaking others  
**Example:** Pricing changes don't affect recommendations

### 4. Configuration Over Code
**Decision:** CLI flags choose implementations  
**Rationale:** Same binary, different behavior  
**Example:**
```bash
# MVP
cost-scan --storage=sqlite --datasource=prometheus

# Future (same binary)
cost-scan --storage=postgres --datasource=cloudwatch --recommender=binpacking
```

### 5. Progressive Enhancement
**Decision:** Basic features work without advanced ones  
**Rationale:** Ship fast, iterate  
**Example:**
- Works without Prometheus (falls back to metrics-server)
- Works without Azure API (uses default pricing)
- Works without storage (ephemeral recommendations)

---

## ✅ Migration Paths (How to Upgrade Later)

### SQLite → PostgreSQL (v2.0)
```go
// No change to CLI code
// Just add postgres.go implementing Store interface

// Week 8 (MVP):
store = sqlite.New("cost-optimizer.db")

// Week 20 (v2.0):
if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
    store = postgres.New(dbURL)
} else {
    store = sqlite.New("cost-optimizer.db")
}
```

### Statistical → Bin-Packing (v2.0)
```go
// No change to CLI code
// Just add binpacking.go implementing Recommender interface

// Week 8 (MVP):
recommender = statistical.New(dataSource, store)

// Week 20 (v2.0):
switch config.RecommenderType {
case "binpacking":
    recommender = binpacking.New(dataSource, store)
case "ml":
    recommender = ml.New(dataSource, store, modelPath)
default:
    recommender = statistical.New(dataSource, store)
}
```

### CLI → API (v2.0)
```go
// Recommendation engine stays same
// Just add api.go implementing OutputHandler interface

// Week 8 (MVP):
output = cli.New(os.Stdout)

// Week 20 (v2.0):
if config.Mode == "server" {
    output = api.New(":8080")
} else {
    output = cli.New(os.Stdout)
}
```

---

## 🚨 Anti-Patterns to Avoid

### ❌ DON'T: Hardcode implementations
```go
// BAD - locked to Prometheus forever
metrics := prometheus.GetMetrics()
```

### ✅ DO: Use interfaces
```go
// GOOD - can swap implementations
var dataSource datasource.DataSource
dataSource = prometheus.New(config)
metrics := dataSource.GetMetrics()
```

### ❌ DON'T: Tight coupling
```go
// BAD - recommender depends on Prometheus directly
type Recommender struct {
    prometheusClient *prometheus.Client
}
```

### ✅ DO: Depend on interfaces
```go
// GOOD - recommender depends on DataSource interface
type Recommender struct {
    dataSource datasource.DataSource
}
```

### ❌ DON'T: Mix layers
```go
// BAD - CLI code doing storage
func RunScan() {
    db.Exec("INSERT INTO recommendations...")
}
```

### ✅ DO: Respect layers
```go
// GOOD - CLI uses Store interface
func RunScan() {
    store.SaveRecommendation(rec)
}
```

---

## 📊 Complexity Budget

| Feature | Now (Weeks 1-8) | Later (v2.0+) | Effort to Add |
|---------|----------------|---------------|---------------|
| **Prometheus** | ✅ YES | - | - |
| **CloudWatch** | ❌ NO | ✅ YES | 1 week (just implement DataSource) |
| **SQLite** | ✅ YES | - | - |
| **PostgreSQL** | ❌ NO | ✅ YES | 3 days (just implement Store) |
| **P95 recommender** | ✅ YES | - | - |
| **Bin-packing** | ❌ NO | ✅ YES | 2 weeks (just implement Recommender) |
| **Azure pricing** | ✅ YES | - | - |
| **AWS pricing** | ❌ NO | ✅ YES | 3 days (just implement PricingProvider) |
| **CLI** | ✅ YES | - | - |
| **REST API** | ❌ NO | ✅ YES | 1 week (just implement OutputHandler) |
| **Web UI** | ❌ NO | ✅ YES | 3 weeks (reads from API) |

**Total MVP:** 8 weeks  
**Adding one feature later:** 3 days - 2 weeks (not months)

---

## 🎯 Success Criteria

### Week 8 (MVP Complete):
- ✅ Recommends based on P95 (not snapshots)
- ✅ Works with 3-5 clusters
- ✅ Accurate Azure pricing
- ✅ Safe --apply with audit
- ✅ SQLite storage
- ✅ 70%+ test coverage

### Week 20 (v2.0 Example):
- ✅ All of above PLUS
- ✅ PostgreSQL support (added in 3 days)
- ✅ AWS pricing (added in 3 days)
- ✅ REST API (added in 1 week)
- ✅ 20+ clusters

**Key Point:** v2.0 features take DAYS, not MONTHS because interfaces already exist.

---

## 🏁 Next Action

**Tomorrow (Day 3):**
1. Create interface definitions (pkg/datasource/datasource.go, etc)
2. Deploy Prometheus to minikube
3. Start implementing PrometheusSource

**This ensures everything we build can be extended later without rewriting.**
---

## 🚨 Known Technical Debt

### TD-001: Dual Model System
**Impact:** Medium  
**Effort:** 4-6 hours  
**Plan:** Migrate in Week 4

Current workaround: Converter bridges old/new models. Works fine but creates duplication.
See: `docs/TECHNICAL_DEBT.md#TD-001`

---
