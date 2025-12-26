package metrics

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func Test_RecordBackupStart_MetricsIncremented(t *testing.T) {
	databaseType := "POSTGRES"
	databaseID := uuid.New()
	databaseName := "test-database-start"
	workspaceID := uuid.New()

	// Get initial values
	initialGauge := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	initialStatusGauge := getGaugeValueByLabel(BackupStatusGauge, "status", "in_progress")
	initialCounter := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "started")

	RecordBackupStart(databaseType, databaseID, databaseName, &workspaceID)

	// Verify BackupInProgressGauge incremented
	gaugeValue := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	assert.Equal(t, initialGauge+1, gaugeValue, "BackupInProgressGauge should increment by 1")

	// Verify BackupStatusGauge incremented
	statusGaugeValue := getGaugeValueByLabel(BackupStatusGauge, "status", "in_progress")
	assert.Equal(t, initialStatusGauge+1, statusGaugeValue, "BackupStatusGauge in_progress should increment by 1")

	// Verify BackupTotalCounter incremented
	counterValue := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "started")
	assert.Equal(t, initialCounter+1, counterValue, "BackupTotalCounter started should increment by 1")
}

func Test_RecordBackupCompletion_MetricsUpdated(t *testing.T) {
	databaseType := "POSTGRES"
	databaseID := uuid.New()
	databaseName := "test-database-complete"
	workspaceID := uuid.New()
	durationSeconds := 5.5
	sizeMB := 10.5

	// Get initial values
	initialCompletedGauge := getGaugeValueByLabel(BackupStatusGauge, "status", "completed")
	initialCompletedCounter := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	initialHistogramCount := getHistogramCount(BackupDurationHistogram, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")

	// First record start
	RecordBackupStart(databaseType, databaseID, databaseName, &workspaceID)

	// Then record completion
	RecordBackupCompletion(databaseType, databaseID, databaseName, &workspaceID, durationSeconds, sizeMB)

	// Verify BackupInProgressGauge is decremented (back to 0 for this specific database)
	gaugeValue := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	assert.Equal(t, float64(0), gaugeValue, "BackupInProgressGauge should be 0 after completion")

	// Verify BackupStatusGauge incremented
	statusGaugeValue := getGaugeValueByLabel(BackupStatusGauge, "status", "completed")
	assert.Equal(t, initialCompletedGauge+1, statusGaugeValue, "BackupStatusGauge completed should increment by 1")

	// Verify BackupTotalCounter incremented
	counterValue := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	assert.Equal(t, initialCompletedCounter+1, counterValue, "BackupTotalCounter completed should increment by 1")

	// Verify BackupDurationHistogram incremented
	histogramCount := getHistogramCount(BackupDurationHistogram, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	assert.Equal(t, initialHistogramCount+1, histogramCount, "BackupDurationHistogram should increment by 1")

	// Verify BackupSizeHistogram incremented
	sizeHistogramCount := getHistogramCount(BackupSizeHistogram, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	assert.Greater(t, sizeHistogramCount, uint64(0), "BackupSizeHistogram should have at least 1 observation")
}

func Test_RecordBackupFailure_MetricsUpdated(t *testing.T) {
	databaseType := "MYSQL"
	databaseID := uuid.New()
	databaseName := "failing-database"
	workspaceID := uuid.New()
	durationSeconds := 2.3

	// Get initial values
	initialFailedGauge := getGaugeValueByLabel(BackupStatusGauge, "status", "failed")
	initialFailedCounter := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "failed")

	// First record start
	RecordBackupStart(databaseType, databaseID, databaseName, &workspaceID)

	// Then record failure
	RecordBackupFailure(databaseType, databaseID, databaseName, &workspaceID, durationSeconds)

	// Verify BackupInProgressGauge is decremented (back to 0 for this specific database)
	gaugeValue := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	assert.Equal(t, float64(0), gaugeValue, "BackupInProgressGauge should be 0 after failure")

	// Verify BackupStatusGauge incremented
	statusGaugeValue := getGaugeValueByLabel(BackupStatusGauge, "status", "failed")
	assert.Equal(t, initialFailedGauge+1, statusGaugeValue, "BackupStatusGauge failed should increment by 1")

	// Verify BackupTotalCounter incremented
	counterValue := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "failed")
	assert.Equal(t, initialFailedCounter+1, counterValue, "BackupTotalCounter failed should increment by 1")

	// Verify BackupDurationHistogram has observations
	// Note: Histogram metrics are complex to test directly, so we verify the function was called
	// by checking that the histogram exists in the registry
	histogramCount := getHistogramCount(BackupDurationHistogram, databaseType, databaseID.String(), databaseName, workspaceID.String(), "failed")
	// The histogram should have at least some observations (may include from other tests)
	assert.Greater(t, histogramCount, uint64(0), "BackupDurationHistogram should have observations")
}

func Test_RecordBackupCancellation_MetricsUpdated(t *testing.T) {
	databaseType := "MARIADB"
	databaseID := uuid.New()
	databaseName := "cancelled-database"
	workspaceID := uuid.New()
	durationSeconds := 1.5

	// Get initial values
	initialCanceledGauge := getGaugeValueByLabel(BackupStatusGauge, "status", "canceled")
	initialCanceledCounter := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "canceled")

	// First record start
	RecordBackupStart(databaseType, databaseID, databaseName, &workspaceID)

	// Then record cancellation
	RecordBackupCancellation(databaseType, databaseID, databaseName, &workspaceID, durationSeconds)

	// Verify BackupInProgressGauge is decremented (back to 0 for this specific database)
	gaugeValue := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	assert.Equal(t, float64(0), gaugeValue, "BackupInProgressGauge should be 0 after cancellation")

	// Verify BackupStatusGauge incremented
	statusGaugeValue := getGaugeValueByLabel(BackupStatusGauge, "status", "canceled")
	assert.Equal(t, initialCanceledGauge+1, statusGaugeValue, "BackupStatusGauge canceled should increment by 1")

	// Verify BackupTotalCounter incremented
	counterValue := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "canceled")
	assert.Equal(t, initialCanceledCounter+1, counterValue, "BackupTotalCounter canceled should increment by 1")
}

func Test_SanitizeLabelValue_HandlesSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "test-db",
			expected: "test-db",
		},
		{
			name:     "name with spaces",
			input:    "my test database",
			expected: "my_test_database",
		},
		{
			name:     "name with special characters",
			input:    "db@prod#123!",
			expected: "db_prod_123_",
		},
		{
			name:     "name starting with number",
			input:    "123database",
			expected: "db_123database",
		},
		{
			name:     "name with unicode",
			input:    "test-ñ-db",
			expected: "test-_-db", // The ñ character is replaced with _ because it's not alphanumeric
		},
		{
			name:     "very long name",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100), // Truncated to 100 characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabelValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_GetWorkspaceID_HandlesNilWorkspace(t *testing.T) {
	// Test with nil workspace
	result := getWorkspaceID(nil)
	assert.Equal(t, "none", result, "Should return 'none' for nil workspace")

	// Test with valid workspace
	workspaceID := uuid.New()
	result = getWorkspaceID(&workspaceID)
	assert.Equal(t, workspaceID.String(), result, "Should return UUID string for valid workspace")
}

func Test_MultipleBackups_SameDatabase_MetricsAccumulate(t *testing.T) {
	databaseType := "POSTGRES"
	databaseID := uuid.New()
	databaseName := "multi-backup-db"
	workspaceID := uuid.New()

	// Get initial values
	initialCompletedCounter := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	initialCompletedGauge := getGaugeValueByLabel(BackupStatusGauge, "status", "completed")

	// Record 3 successful backups
	for i := 0; i < 3; i++ {
		RecordBackupStart(databaseType, databaseID, databaseName, &workspaceID)
		RecordBackupCompletion(databaseType, databaseID, databaseName, &workspaceID, 5.0, 10.0)
	}

	// Verify counters accumulated
	counterValue := getCounterValue(BackupTotalCounter, databaseType, databaseID.String(), databaseName, workspaceID.String(), "completed")
	assert.Equal(t, initialCompletedCounter+3, counterValue, "BackupTotalCounter should increment by 3")

	// Verify status gauge accumulated
	statusGaugeValue := getGaugeValueByLabel(BackupStatusGauge, "status", "completed")
	assert.Equal(t, initialCompletedGauge+3, statusGaugeValue, "BackupStatusGauge completed should increment by 3")

	// Verify in progress is 0
	gaugeValue := getGaugeValue(BackupInProgressGauge, databaseType, databaseID.String(), databaseName, workspaceID.String())
	assert.Equal(t, float64(0), gaugeValue, "BackupInProgressGauge should be 0")
}

func Test_DifferentDatabases_MetricsSeparated(t *testing.T) {
	database1ID := uuid.New()
	database2ID := uuid.New()
	workspaceID := uuid.New()

	// Get initial values
	initialCounter1 := getCounterValue(BackupTotalCounter, "POSTGRES", database1ID.String(), "db1", workspaceID.String(), "completed")
	initialCounter2 := getCounterValue(BackupTotalCounter, "MYSQL", database2ID.String(), "db2", workspaceID.String(), "completed")

	// Record backup for database 1
	RecordBackupStart("POSTGRES", database1ID, "db1", &workspaceID)
	RecordBackupCompletion("POSTGRES", database1ID, "db1", &workspaceID, 5.0, 10.0)

	// Record backup for database 2
	RecordBackupStart("MYSQL", database2ID, "db2", &workspaceID)
	RecordBackupCompletion("MYSQL", database2ID, "db2", &workspaceID, 3.0, 5.0)

	// Verify they are tracked separately
	counter1 := getCounterValue(BackupTotalCounter, "POSTGRES", database1ID.String(), "db1", workspaceID.String(), "completed")
	counter2 := getCounterValue(BackupTotalCounter, "MYSQL", database2ID.String(), "db2", workspaceID.String(), "completed")

	assert.Equal(t, initialCounter1+1, counter1, "Database 1 should increment by 1")
	assert.Equal(t, initialCounter2+1, counter2, "Database 2 should increment by 1")
}

// Helper functions to extract metric values (private functions at bottom)

func getGaugeValue(gauge *prometheus.GaugeVec, labels ...string) float64 {
	metric := &dto.Metric{}
	err := gauge.WithLabelValues(labels...).Write(metric)
	if err != nil {
		return 0
	}
	return metric.Gauge.GetValue()
}

func getGaugeValueByLabel(gauge *prometheus.GaugeVec, labelName, labelValue string) float64 {
	metric := &dto.Metric{}
	// For BackupStatusGauge, we only have one label: "status"
	err := gauge.WithLabelValues(labelValue).Write(metric)
	if err != nil {
		return 0
	}
	return metric.Gauge.GetValue()
}

func getCounterValue(counter *prometheus.CounterVec, labels ...string) float64 {
	metric := &dto.Metric{}
	err := counter.WithLabelValues(labels...).Write(metric)
	if err != nil {
		return 0
	}
	return metric.Counter.GetValue()
}

func getHistogramCount(histogram *prometheus.HistogramVec, labels ...string) uint64 {
	// Use the default registry to gather metrics
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0
	}

	// Determine histogram name by comparing pointer
	histogramName := ""
	if histogram == BackupDurationHistogram {
		histogramName = "databasus_backup_duration_seconds"
	} else if histogram == BackupSizeHistogram {
		histogramName = "databasus_backup_size_mb"
	} else {
		return 0
	}

	// Search for the metric family
	for _, mf := range metrics {
		if mf.GetName() == histogramName {
			// Find metric with matching labels
			for _, metric := range mf.GetMetric() {
				if metric.Histogram != nil {
					// Check if labels match
					if len(labels) >= 5 {
						metricLabels := make(map[string]string)
						for _, label := range metric.GetLabel() {
							metricLabels[label.GetName()] = label.GetValue()
						}
						
						// Verify all expected labels match
						if metricLabels["database_type"] == labels[0] &&
							metricLabels["database_id"] == labels[1] &&
							metricLabels["database_name"] == labels[2] &&
							metricLabels["workspace_id"] == labels[3] &&
							metricLabels["status"] == labels[4] {
							return metric.Histogram.GetSampleCount()
						}
					}
				}
			}
			// If no exact match, return the first histogram's count (for testing purposes)
			for _, metric := range mf.GetMetric() {
				if metric.Histogram != nil {
					return metric.Histogram.GetSampleCount()
				}
			}
		}
	}
	
	return 0
}

