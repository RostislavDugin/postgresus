# Prometheus Metrics

This project is instrumented with Prometheus to monitor backup status and performance.

## Metrics Endpoint

Metrics are available at the `/metrics` endpoint (no authentication required):

```
http://localhost:4005/metrics
```

## Available Metrics

### 1. `databasus_backups_status`

Gauge that shows the current number of backups by status.

**Labels:**

- `status`: Backup status (`in_progress`, `completed`, `failed`, `canceled`)

**Example:**

```
databasus_backups_status{status="in_progress"} 2
databasus_backups_status{status="completed"} 150
databasus_backups_status{status="failed"} 5
```

### 2. `databasus_backups_total`

Counter of the total number of backup attempts.

**Labels:**

- `database_type`: Database type (`POSTGRES`, `MYSQL`, `MARIADB`, `MONGODB`)
- `database_id`: Unique database ID (UUID)
- `database_name`: Database name
- `workspace_id`: Workspace ID (UUID) or `none` if no workspace
- `status`: Backup status (`started`, `completed`, `failed`, `canceled`)

**Example:**

```
databasus_backups_total{database_type="POSTGRES",database_id="123e4567-e89b-12d3-a456-426614174000",database_name="production_db",workspace_id="456e7890-e89b-12d3-a456-426614174001",status="completed"} 100
databasus_backups_total{database_type="MYSQL",database_id="789e0123-e89b-12d3-a456-426614174002",database_name="test_db",workspace_id="none",status="failed"} 3
```

### 3. `databasus_backup_duration_seconds`

Histogram that shows backup duration in seconds.

**Labels:**

- `database_type`: Database type
- `database_id`: Unique database ID (UUID)
- `database_name`: Database name
- `workspace_id`: Workspace ID (UUID) or `none` if no workspace
- `status`: Backup status (`completed`, `failed`, `canceled`)

**Buckets:** 1s, 5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m, 1h

**Example:**

```
databasus_backup_duration_seconds_bucket{database_type="POSTGRES",status="completed",le="60"} 95
databasus_backup_duration_seconds_sum{database_type="POSTGRES",status="completed"} 4500
databasus_backup_duration_seconds_count{database_type="POSTGRES",status="completed"} 100
```

### 4. `databasus_backup_size_mb`

Histogram that shows backup size in megabytes.

**Labels:**

- `database_type`: Database type
- `database_id`: Unique database ID (UUID)
- `database_name`: Database name
- `workspace_id`: Workspace ID (UUID) or `none` if no workspace
- `status`: Backup status (`completed`)

**Buckets:** 1MB, 10MB, 50MB, 100MB, 500MB, 1GB, 5GB, 10GB

**Example:**

```
databasus_backup_size_mb_bucket{database_type="POSTGRES",status="completed",le="100"} 80
databasus_backup_size_mb_sum{database_type="POSTGRES",status="completed"} 5000
databasus_backup_size_mb_count{database_type="POSTGRES",status="completed"} 100
```

### 5. `databasus_backups_in_progress`

Gauge that shows the number of backups currently in progress per database.

**Labels:**

- `database_type`: Database type
- `database_id`: Unique database ID (UUID)
- `database_name`: Database name
- `workspace_id`: Workspace ID (UUID) or `none` if no workspace

**Example:**

```
databasus_backups_in_progress{database_type="POSTGRES",database_id="123e4567-e89b-12d3-a456-426614174000",database_name="production_db",workspace_id="456e7890-e89b-12d3-a456-426614174001"} 1
databasus_backups_in_progress{database_type="MYSQL",database_id="789e0123-e89b-12d3-a456-426614174002",database_name="test_db",workspace_id="none"} 0
```

## Useful Prometheus Queries

### Backups in progress

```promql
databasus_backups_in_progress
```

### Backup success rate (last 24 hours)

```promql
sum(rate(databasus_backups_total{status="completed"}[24h])) /
sum(rate(databasus_backups_total{status!="started"}[24h])) * 100
```

### Average backup duration by type

```promql
rate(databasus_backup_duration_seconds_sum[5m]) /
rate(databasus_backup_duration_seconds_count[5m])
```

### Average backup size

```promql
rate(databasus_backup_size_mb_sum[5m]) /
rate(databasus_backup_size_mb_count[5m])
```

### Number of failed backups in the last hour

```promql
sum(increase(databasus_backups_total{status="failed"}[1h]))
```

### Backups by database type

```promql
sum by (database_type) (databasus_backups_total{status="completed"})
```

### Backups by database name

```promql
sum by (database_name) (databasus_backups_total{status="completed"})
```

### Backups by workspace

```promql
sum by (workspace_id) (databasus_backups_total{status="completed"})
```

### Backups for a specific database

```promql
databasus_backups_total{database_name="production_db"}
```

### Success rate by database

```promql
sum by (database_name) (rate(databasus_backups_total{status="completed"}[5m])) /
sum by (database_name) (rate(databasus_backups_total{status!="started"}[5m])) * 100
```

## Prometheus Configuration

To configure Prometheus to scrape metrics, add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "databasus"
    scrape_interval: 15s
    static_configs:
      - targets: ["localhost:4005"]
```

## Grafana Dashboards

You can create Grafana dashboards using these metrics to visualize:

- Current backup status
- Success/failure rate
- Average backup duration
- Average backup size
- Backup distribution by database type
- Historical trends

## Notes

- Metrics are automatically recorded when backup events occur
- The `/metrics` endpoint is public and requires no authentication (Prometheus standard)
- Metrics are kept in memory and reset when the server restarts
