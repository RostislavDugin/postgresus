package metrics

import (
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BackupStatusGauge tracks the current number of backups by status
	BackupStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "databasus_backups_status",
			Help: "Current number of backups by status (in_progress, completed, failed, canceled)",
		},
		[]string{"status"},
	)

	// BackupTotalCounter counts total backup attempts
	BackupTotalCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "databasus_backups_total",
			Help: "Total number of backup attempts",
		},
		[]string{"database_type", "database_id", "database_name", "workspace_id", "status"},
	)

	// BackupDurationHistogram tracks backup duration in seconds
	BackupDurationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "databasus_backup_duration_seconds",
			Help:    "Backup duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}, // 1s to 1 hour
		},
		[]string{"database_type", "database_id", "database_name", "workspace_id", "status"},
	)

	// BackupSizeHistogram tracks backup size in megabytes
	BackupSizeHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "databasus_backup_size_mb",
			Help:    "Backup size in megabytes",
			Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000}, // 1MB to 10GB
		},
		[]string{"database_type", "database_id", "database_name", "workspace_id", "status"},
	)

	// BackupInProgressGauge tracks currently running backups by database
	BackupInProgressGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "databasus_backups_in_progress",
			Help: "Number of backups currently in progress by database type",
		},
		[]string{"database_type", "database_id", "database_name", "workspace_id"},
	)
)

// getWorkspaceID returns the workspace ID as a string, or "none" if nil
func getWorkspaceID(workspaceID *uuid.UUID) string {
	if workspaceID == nil {
		return "none"
	}
	return workspaceID.String()
}

// RecordBackupStart records when a backup starts
func RecordBackupStart(databaseType string, databaseID uuid.UUID, databaseName string, workspaceID *uuid.UUID) {
	dbIDStr := databaseID.String()
	dbName := sanitizeLabelValue(databaseName)
	wsIDStr := getWorkspaceID(workspaceID)
	
	BackupInProgressGauge.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr).Inc()
	BackupStatusGauge.WithLabelValues("in_progress").Inc()
	BackupTotalCounter.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "started").Inc()
}

// RecordBackupCompletion records a successful backup completion
func RecordBackupCompletion(databaseType string, databaseID uuid.UUID, databaseName string, workspaceID *uuid.UUID, durationSeconds float64, sizeMB float64) {
	dbIDStr := databaseID.String()
	dbName := sanitizeLabelValue(databaseName)
	wsIDStr := getWorkspaceID(workspaceID)
	
	BackupInProgressGauge.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr).Dec()
	BackupStatusGauge.WithLabelValues("in_progress").Dec()
	BackupStatusGauge.WithLabelValues("completed").Inc()
	BackupTotalCounter.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "completed").Inc()
	BackupDurationHistogram.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "completed").Observe(durationSeconds)
	BackupSizeHistogram.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "completed").Observe(sizeMB)
}

// RecordBackupFailure records a failed backup
func RecordBackupFailure(databaseType string, databaseID uuid.UUID, databaseName string, workspaceID *uuid.UUID, durationSeconds float64) {
	dbIDStr := databaseID.String()
	dbName := sanitizeLabelValue(databaseName)
	wsIDStr := getWorkspaceID(workspaceID)
	
	BackupInProgressGauge.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr).Dec()
	BackupStatusGauge.WithLabelValues("in_progress").Dec()
	BackupStatusGauge.WithLabelValues("failed").Inc()
	BackupTotalCounter.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "failed").Inc()
	BackupDurationHistogram.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "failed").Observe(durationSeconds)
}

// RecordBackupCancellation records a cancelled backup
func RecordBackupCancellation(databaseType string, databaseID uuid.UUID, databaseName string, workspaceID *uuid.UUID, durationSeconds float64) {
	dbIDStr := databaseID.String()
	dbName := sanitizeLabelValue(databaseName)
	wsIDStr := getWorkspaceID(workspaceID)
	
	BackupInProgressGauge.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr).Dec()
	BackupStatusGauge.WithLabelValues("in_progress").Dec()
	BackupStatusGauge.WithLabelValues("canceled").Inc()
	BackupTotalCounter.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "canceled").Inc()
	BackupDurationHistogram.WithLabelValues(databaseType, dbIDStr, dbName, wsIDStr, "canceled").Observe(durationSeconds)
}

// sanitizeLabelValue sanitizes a string to be used as a Prometheus label value
// Prometheus labels have restrictions: must match [a-zA-Z_:][a-zA-Z0-9_:]*
// We'll replace invalid characters with underscores
func sanitizeLabelValue(value string) string {
	// Replace spaces and special characters with underscores
	// Keep only alphanumeric, underscores, and hyphens
	result := make([]rune, 0, len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	sanitized := string(result)
	// Ensure it doesn't start with a number (Prometheus requirement)
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "db_" + sanitized
	}
	// Limit length to avoid issues (Prometheus has practical limits)
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	return sanitized
}

// UpdateBackupStatusGauge updates the gauge for a specific status
// This is useful for syncing the gauge with actual database state
func UpdateBackupStatusGauge(status string, count float64) {
	BackupStatusGauge.WithLabelValues(status).Set(count)
}

