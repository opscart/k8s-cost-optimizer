-- K8s Cost Optimizer Database Schema
-- Engine: PostgreSQL 14+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cluster_id VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    deployment VARCHAR(255),
    pod VARCHAR(255) NOT NULL,
    container VARCHAR(255),
    
    -- Recommendation type
    type VARCHAR(50) NOT NULL, -- RIGHT_SIZE, SCALE_DOWN, NO_ACTION
    
    -- Current state
    current_cpu_millicores BIGINT,
    current_memory_bytes BIGINT,
    
    -- Recommended state
    recommended_cpu_millicores BIGINT,
    recommended_memory_bytes BIGINT,
    
    -- Analysis
    reason TEXT,
    savings_monthly_usd DECIMAL(10,2),
    impact VARCHAR(20), -- HIGH, MEDIUM, LOW
    risk VARCHAR(20), -- NONE, LOW, MEDIUM, HIGH
    
    -- Command
    command TEXT,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    applied_by VARCHAR(255)
);

-- Indexes for recommendations
CREATE INDEX IF NOT EXISTS idx_recommendations_namespace ON recommendations(namespace, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_cluster ON recommendations(cluster_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_type ON recommendations(type);
CREATE INDEX IF NOT EXISTS idx_recommendations_created_at ON recommendations(created_at DESC);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recommendation_id UUID,
    action VARCHAR(50) NOT NULL, -- APPLIED, ROLLED_BACK, VIEWED
    status VARCHAR(50) NOT NULL, -- SUCCESS, FAILED
    error_message TEXT,
    executed_by VARCHAR(255),
    executed_at TIMESTAMPTZ DEFAULT NOW(),
    
    FOREIGN KEY (recommendation_id) REFERENCES recommendations(id) ON DELETE CASCADE
);

-- Indexes for audit log
CREATE INDEX IF NOT EXISTS idx_audit_log_recommendation ON audit_log(recommendation_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_executed_at ON audit_log(executed_at DESC);

-- Metrics cache table
CREATE TABLE IF NOT EXISTS metrics_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cluster_id VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    pod VARCHAR(255) NOT NULL,
    
    -- Metrics
    p95_cpu_millicores BIGINT,
    p99_cpu_millicores BIGINT,
    max_cpu_millicores BIGINT,
    avg_cpu_millicores BIGINT,
    
    p95_memory_bytes BIGINT,
    p99_memory_bytes BIGINT,
    max_memory_bytes BIGINT,
    avg_memory_bytes BIGINT,
    
    requested_cpu_millicores BIGINT,
    requested_memory_bytes BIGINT,
    
    -- Metadata
    collected_at TIMESTAMPTZ NOT NULL,
    duration_hours INTEGER NOT NULL,
    sample_count INTEGER,
    
    -- TTL
    expires_at TIMESTAMPTZ NOT NULL
);

-- Indexes for metrics cache
CREATE INDEX IF NOT EXISTS idx_metrics_cache_pod ON metrics_cache(cluster_id, namespace, pod);
CREATE INDEX IF NOT EXISTS idx_metrics_cache_expires ON metrics_cache(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_cache_unique ON metrics_cache(cluster_id, namespace, pod, collected_at);

-- Cluster metadata table
CREATE TABLE IF NOT EXISTS clusters (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cloud_provider VARCHAR(50), -- azure, aws, gcp, on-prem
    region VARCHAR(100),
    node_count INTEGER,
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Version tracking (for migrations)
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO schema_version (version) VALUES (1) ON CONFLICT (version) DO NOTHING;
