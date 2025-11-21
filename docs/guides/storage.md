# Storage Setup Guide

Configure PostgreSQL storage for persistent recommendations and audit trails.

## Why Use Storage?

Without storage:
- Recommendations are lost after each scan
- No historical tracking
- No audit trail
- Can't track what was applied

With storage:
- Track all recommendations over time
- Audit trail of applied changes
- Historical cost savings data
- Compare recommendations across scans

## Quick Setup (Docker)

### Local Development
```bash
docker run -d \
  --name k8s-cost-postgres \
  -e POSTGRES_USER=costuser \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=costoptimizer \
  -p 5432:5432 \
  postgres:14
```

### With Persistence
```bash
docker run -d \
  --name k8s-cost-postgres \
  -e POSTGRES_USER=costuser \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=costoptimizer \
  -p 5432:5432 \
  -v pgdata:/var/lib/postgresql/data \
  postgres:14
```

## Production Setup

### 1. Managed PostgreSQL (Recommended)

**Azure:**
```bash
az postgres flexible-server create \
  --name k8s-cost-optimizer \
  --resource-group mygroup \
  --location eastus \
  --admin-user costuser \
  --admin-password <secure-password> \
  --sku-name Standard_B1ms \
  --version 14
```

**AWS RDS:**
```bash
aws rds create-db-instance \
  --db-instance-identifier k8s-cost-optimizer \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 14 \
  --master-username costuser \
  --master-user-password <secure-password> \
  --allocated-storage 20
```

**Google Cloud SQL:**
```bash
gcloud sql instances create k8s-cost-optimizer \
  --database-version=POSTGRES_14 \
  --tier=db-f1-micro \
  --region=us-central1
```

### 2. Kubernetes Deployment
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:14
        env:
        - name: POSTGRES_USER
          value: costuser
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: POSTGRES_DB
          value: costoptimizer
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

## Configuration

### Environment Variables
```bash
# Default (local Docker)
export DATABASE_URL="postgresql://costuser:devpassword@localhost:5432/costoptimizer?sslmode=disable"

# Production (with SSL)
export DATABASE_URL="postgresql://costuser:password@prod-db.example.com:5432/costoptimizer?sslmode=require"

# Optional: Disable storage
export STORAGE_ENABLED=false
```

### Connection String Format
```
postgresql://[user]:[password]@[host]:[port]/[database]?[options]
```

**Options:**
- `sslmode=disable` - Local development only
- `sslmode=require` - Production (enforce SSL)
- `connect_timeout=10` - Connection timeout in seconds

## Database Schema

The tool automatically creates these tables:

1. **recommendations** - Optimization recommendations
2. **audit_log** - Action tracking
3. **metrics_cache** - Cached metrics (24h TTL)
4. **clusters** - Cluster metadata
5. **schema_version** - Migration tracking

See [Database Schema Documentation](../database/README.md) for details.

## Usage Examples

### Scan and Save
```bash
# Save to database
cost-scan -n production --save

# Specify cluster ID
cost-scan -n production --save --cluster-id=prod-us-east-1
```

### View History
```bash
# All recommendations for namespace
cost-scan history production

# Limit results
cost-scan history production --limit 20
```

### View Audit Trail
```bash
cost-scan audit <recommendation-id>
```

## Backup & Restore

### Backup
```bash
# Full backup
docker exec k8s-cost-postgres pg_dump -U costuser costoptimizer > backup.sql

# Compressed backup
docker exec k8s-cost-postgres pg_dump -U costuser costoptimizer | gzip > backup.sql.gz
```

### Restore
```bash
# From plain SQL
docker exec -i k8s-cost-postgres psql -U costuser costoptimizer < backup.sql

# From compressed
gunzip < backup.sql.gz | docker exec -i k8s-cost-postgres psql -U costuser costoptimizer
```

## Maintenance

### View Database Stats
```bash
docker exec -it k8s-cost-postgres psql -U costuser -d costoptimizer -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
```

### Cleanup Old Data
```bash
# Delete recommendations older than 90 days
docker exec -it k8s-cost-postgres psql -U costuser -d costoptimizer -c "
DELETE FROM recommendations 
WHERE created_at < NOW() - INTERVAL '90 days';"

# Clean expired cache
docker exec -it k8s-cost-postgres psql -U costuser -d costoptimizer -c "
DELETE FROM metrics_cache 
WHERE expires_at < NOW();"
```

## Troubleshooting

**Connection refused:**
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check port binding
docker port k8s-cost-postgres
```

**Authentication failed:**
```bash
# Verify credentials
docker exec -it k8s-cost-postgres psql -U costuser -d costoptimizer

# Reset password
docker exec -it k8s-cost-postgres psql -U postgres -c "ALTER USER costuser PASSWORD 'newpassword';"
```

**Table doesn't exist:**
```bash
# Schema is auto-created on first connection
# Manually run if needed:
docker exec -i k8s-cost-postgres psql -U costuser -d costoptimizer < docs/database/postgres_schema.sql
```

---

**Next:** [Configuration Guide](configuration.md)
