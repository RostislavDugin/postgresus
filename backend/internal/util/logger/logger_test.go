package logger

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readLogFileRecords(t *testing.T, logPath string) []map[string]any {
	t.Helper()

	writtenLog, err := os.ReadFile(logPath)
	require.NoError(t, err)

	var records []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(string(writtenLog)), "\n") {
		if line == "" {
			continue
		}

		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}

	return records
}

// This is the requirement the whole change exists for: an audit entry has to survive on disk even
// when the level is raised, so it outlives the database it describes.
func Test_BuildSinks_WithFileEnabled_AuditRecordReachesDiskAtErrorLevel(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "databasus.log")

	assembledSinks := buildSinks(settings{
		level: slog.LevelError,
		file:  logFileSettings{isEnabled: true, path: logPath},
	})
	require.Empty(t, assembledSinks.failures)
	require.Len(t, assembledSinks.handlers, 2, "stdout plus file")
	require.Len(t, assembledSinks.shutdowns, 1)

	level := new(slog.LevelVar)
	level.Set(slog.LevelError)
	fanOut := newFanOutHandler(assembledSinks.handlers, level)

	slog.New(fanOut).Info("routine work that should be filtered")
	slog.New(fanOut.withLevelBypass()).With(logTypeKey, logTypeAudit).
		Info("Database deleted: payments", "user_id", "user-7")

	for _, shutdown := range assembledSinks.shutdowns {
		require.NoError(t, shutdown(t.Context()))
	}

	records := readLogFileRecords(t, logPath)
	require.Len(t, records, 1, "the info record must be filtered, the audit record must survive")
	assert.Equal(t, logTypeAudit, records[0][logTypeKey])
	assert.Equal(t, "Database deleted: payments", records[0]["msg"])
	assert.Equal(t, "user-7", records[0]["user_id"])
}

func Test_BuildSinks_WithFileEnabled_RedactsBeforeTouchingDisk(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "databasus.log")

	assembledSinks := buildSinks(settings{
		level: slog.LevelInfo,
		file:  logFileSettings{isEnabled: true, path: logPath},
	})
	require.Empty(t, assembledSinks.failures)

	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	slog.New(newFanOutHandler(assembledSinks.handlers, level)).
		Info("connecting to postgres://admin:hunter2@db:5432/app", "user_email", "rostislav@acme.com")

	for _, shutdown := range assembledSinks.shutdowns {
		require.NoError(t, shutdown(t.Context()))
	}

	writtenLog, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.NotContains(t, string(writtenLog), "hunter2")
	assert.NotContains(t, string(writtenLog), "rostislav@acme.com")
	assert.Contains(t, string(writtenLog), "db:5432")
}

func Test_BuildSinks_WhenFileIsDisabled_OnlyStdoutSinkIsBuilt(t *testing.T) {
	assembledSinks := buildSinks(settings{level: slog.LevelInfo})

	assert.Len(t, assembledSinks.handlers, 1)
	assert.Empty(t, assembledSinks.shutdowns)
	assert.Empty(t, assembledSinks.failures)
}

// An unopenable log file must degrade to a warning, not take the process down the way a malformed
// OPEN_TELEMETRY_URL does.
func Test_BuildSinks_WhenLogFileCannotBeOpened_ReportsFailureAndKeepsStdout(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}

	blockedRoot := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.Mkdir(blockedRoot, 0o500))

	assembledSinks := buildSinks(settings{
		level: slog.LevelInfo,
		file: logFileSettings{
			isEnabled: true,
			path:      filepath.Join(blockedRoot, "nested", "databasus.log"),
		},
	})

	assert.Len(t, assembledSinks.handlers, 1, "stdout survives")
	require.Len(t, assembledSinks.failures, 1)
	assert.Equal(t, "log file sink disabled", assembledSinks.failures[0].message)
	require.Error(t, assembledSinks.failures[0].err)
}

// Shutdown must not build the pipeline just to close it: that would create the log file on a
// process that never logged, and a malformed OPEN_TELEMETRY_URL would exit the test binary.
func Test_Shutdown_WhenLoggerWasNeverInitialized_ReturnsWithoutBuildingSinks(t *testing.T) {
	require.False(t, isInitialized.Load(), "no test in this package may touch the singleton")

	require.NoError(t, Shutdown(t.Context()))

	assert.False(t, isInitialized.Load())
}
