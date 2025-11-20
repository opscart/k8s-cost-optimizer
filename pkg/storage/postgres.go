package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/opscart/k8s-cost-optimizer/pkg/models"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var postgresFS embed.FS

// PostgresStore implements Store interface using PostgreSQL
type PostgresStore struct {
	db  *sql.DB
	dsn string
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PostgresStore{
		db:  db,
		dsn: dsn,
	}

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// migrate runs database migrations
func (s *PostgresStore) migrate() error {
	// Read schema file
	schema, err := postgresFS.ReadFile("migrations/001_postgres_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Execute schema
	if _, err := s.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// SaveRecommendation saves a recommendation
func (s *PostgresStore) SaveRecommendation(ctx context.Context, rec *models.Recommendation) error {
	if rec.ID == "" {
		rec.ID = uuid.New().String()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO recommendations (
			id, cluster_id, namespace, deployment, pod, container,
			type, current_cpu_millicores, current_memory_bytes,
			recommended_cpu_millicores, recommended_memory_bytes,
			reason, savings_monthly_usd, impact, risk, command,
			created_at, applied_at, applied_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	var appliedAt *time.Time
	if rec.AppliedAt != nil {
		appliedAt = rec.AppliedAt
	}

	_, err := s.db.ExecContext(ctx, query,
		rec.ID, rec.Workload.ClusterID, rec.Workload.Namespace,
		rec.Workload.Deployment, rec.Workload.Pod, rec.Workload.Container,
		rec.Type, rec.CurrentCPU, rec.CurrentMemory,
		rec.RecommendedCPU, rec.RecommendedMemory,
		rec.Reason, rec.SavingsMonthly, rec.Impact, rec.Risk, rec.Command,
		rec.CreatedAt, appliedAt, rec.AppliedBy,
	)

	return err
}

// GetRecommendation retrieves a recommendation by ID
func (s *PostgresStore) GetRecommendation(ctx context.Context, id string) (*models.Recommendation, error) {
	query := `
		SELECT id, cluster_id, namespace, deployment, pod, container,
			type, current_cpu_millicores, current_memory_bytes,
			recommended_cpu_millicores, recommended_memory_bytes,
			reason, savings_monthly_usd, impact, risk, command,
			created_at, applied_at, applied_by
		FROM recommendations
		WHERE id = $1
	`

	var rec models.Recommendation
	var workload models.Workload
	var appliedAt sql.NullTime
	var deployment, container, appliedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&rec.ID, &workload.ClusterID, &workload.Namespace,
		&deployment, &workload.Pod, &container,
		&rec.Type, &rec.CurrentCPU, &rec.CurrentMemory,
		&rec.RecommendedCPU, &rec.RecommendedMemory,
		&rec.Reason, &rec.SavingsMonthly, &rec.Impact, &rec.Risk, &rec.Command,
		&rec.CreatedAt, &appliedAt, &appliedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("recommendation not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	workload.Deployment = deployment.String
	workload.Container = container.String
	rec.Workload = &workload

	if appliedAt.Valid {
		rec.AppliedAt = &appliedAt.Time
	}
	if appliedBy.Valid {
		rec.AppliedBy = appliedBy.String
	}

	return &rec, nil
}

// ListRecommendations retrieves recommendations for a namespace
func (s *PostgresStore) ListRecommendations(ctx context.Context, namespace string, limit int) ([]*models.Recommendation, error) {
	query := `
		SELECT id, cluster_id, namespace, deployment, pod, container,
			type, current_cpu_millicores, current_memory_bytes,
			recommended_cpu_millicores, recommended_memory_bytes,
			reason, savings_monthly_usd, impact, risk, command,
			created_at, applied_at, applied_by
		FROM recommendations
		WHERE namespace = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, namespace, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommendations []*models.Recommendation
	for rows.Next() {
		var rec models.Recommendation
		var workload models.Workload
		var appliedAt sql.NullTime
		var deployment, container, appliedBy sql.NullString

		err := rows.Scan(
			&rec.ID, &workload.ClusterID, &workload.Namespace,
			&deployment, &workload.Pod, &container,
			&rec.Type, &rec.CurrentCPU, &rec.CurrentMemory,
			&rec.RecommendedCPU, &rec.RecommendedMemory,
			&rec.Reason, &rec.SavingsMonthly, &rec.Impact, &rec.Risk, &rec.Command,
			&rec.CreatedAt, &appliedAt, &appliedBy,
		)
		if err != nil {
			return nil, err
		}

		workload.Deployment = deployment.String
		workload.Container = container.String
		rec.Workload = &workload

		if appliedAt.Valid {
			rec.AppliedAt = &appliedAt.Time
		}
		if appliedBy.Valid {
			rec.AppliedBy = appliedBy.String
		}

		recommendations = append(recommendations, &rec)
	}

	return recommendations, rows.Err()
}

// UpdateRecommendation updates an existing recommendation
func (s *PostgresStore) UpdateRecommendation(ctx context.Context, rec *models.Recommendation) error {
	query := `
		UPDATE recommendations
		SET applied_at = $1, applied_by = $2
		WHERE id = $3
	`

	var appliedAt *time.Time
	if rec.AppliedAt != nil {
		appliedAt = rec.AppliedAt
	}

	result, err := s.db.ExecContext(ctx, query, appliedAt, rec.AppliedBy, rec.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("recommendation not found: %s", rec.ID)
	}

	return nil
}

// LogAction logs an action to the audit trail
func (s *PostgresStore) LogAction(ctx context.Context, entry *models.AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.ExecutedAt.IsZero() {
		entry.ExecutedAt = time.Now()
	}

	query := `
		INSERT INTO audit_log (
			id, recommendation_id, action, status,
			error_message, executed_by, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		entry.ID, entry.RecommendationID, entry.Action, entry.Status,
		entry.ErrorMessage, entry.ExecutedBy, entry.ExecutedAt,
	)

	return err
}

// GetAuditLog retrieves audit log entries for a recommendation
func (s *PostgresStore) GetAuditLog(ctx context.Context, recommendationID string) ([]*models.AuditEntry, error) {
	query := `
		SELECT id, recommendation_id, action, status,
			error_message, executed_by, executed_at
		FROM audit_log
		WHERE recommendation_id = $1
		ORDER BY executed_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, recommendationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.AuditEntry
	for rows.Next() {
		var entry models.AuditEntry
		var errorMessage, executedBy sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.RecommendationID, &entry.Action, &entry.Status,
			&errorMessage, &executedBy, &entry.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}

		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}
		if executedBy.Valid {
			entry.ExecutedBy = executedBy.String
		}

		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// CacheMetrics stores metrics in cache
func (s *PostgresStore) CacheMetrics(ctx context.Context, workload *models.Workload, metrics *models.Metrics) error {
	query := `
		INSERT INTO metrics_cache (
			cluster_id, namespace, pod,
			p95_cpu_millicores, p99_cpu_millicores, max_cpu_millicores, avg_cpu_millicores,
			p95_memory_bytes, p99_memory_bytes, max_memory_bytes, avg_memory_bytes,
			requested_cpu_millicores, requested_memory_bytes,
			collected_at, duration_hours, sample_count, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (cluster_id, namespace, pod, collected_at) DO UPDATE SET
			p95_cpu_millicores = EXCLUDED.p95_cpu_millicores,
			p99_cpu_millicores = EXCLUDED.p99_cpu_millicores,
			max_cpu_millicores = EXCLUDED.max_cpu_millicores,
			avg_cpu_millicores = EXCLUDED.avg_cpu_millicores,
			p95_memory_bytes = EXCLUDED.p95_memory_bytes,
			p99_memory_bytes = EXCLUDED.p99_memory_bytes,
			max_memory_bytes = EXCLUDED.max_memory_bytes,
			avg_memory_bytes = EXCLUDED.avg_memory_bytes,
			requested_cpu_millicores = EXCLUDED.requested_cpu_millicores,
			requested_memory_bytes = EXCLUDED.requested_memory_bytes,
			expires_at = EXCLUDED.expires_at
	`

	expiresAt := time.Now().Add(24 * time.Hour)
	durationHours := int(metrics.Duration.Hours())

	_, err := s.db.ExecContext(ctx, query,
		workload.ClusterID, workload.Namespace, workload.Pod,
		metrics.P95CPU, metrics.P99CPU, metrics.MaxCPU, metrics.AvgCPU,
		metrics.P95Memory, metrics.P99Memory, metrics.MaxMemory, metrics.AvgMemory,
		metrics.RequestedCPU, metrics.RequestedMemory,
		metrics.CollectedAt, durationHours, metrics.SampleCount, expiresAt,
	)

	return err
}

// GetCachedMetrics retrieves cached metrics
func (s *PostgresStore) GetCachedMetrics(ctx context.Context, workload *models.Workload) (*models.Metrics, error) {
	query := `
		SELECT p95_cpu_millicores, p99_cpu_millicores, max_cpu_millicores, avg_cpu_millicores,
			p95_memory_bytes, p99_memory_bytes, max_memory_bytes, avg_memory_bytes,
			requested_cpu_millicores, requested_memory_bytes,
			collected_at, duration_hours, sample_count
		FROM metrics_cache
		WHERE cluster_id = $1 AND namespace = $2 AND pod = $3
			AND expires_at > NOW()
		ORDER BY collected_at DESC
		LIMIT 1
	`

	var metrics models.Metrics
	var durationHours int

	err := s.db.QueryRowContext(ctx, query,
		workload.ClusterID, workload.Namespace, workload.Pod,
	).Scan(
		&metrics.P95CPU, &metrics.P99CPU, &metrics.MaxCPU, &metrics.AvgCPU,
		&metrics.P95Memory, &metrics.P99Memory, &metrics.MaxMemory, &metrics.AvgMemory,
		&metrics.RequestedCPU, &metrics.RequestedMemory,
		&metrics.CollectedAt, &durationHours, &metrics.SampleCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no cached metrics found")
	}
	if err != nil {
		return nil, err
	}

	metrics.Duration = time.Duration(durationHours) * time.Hour

	return &metrics, nil
}

// Ping checks database connectivity
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}
