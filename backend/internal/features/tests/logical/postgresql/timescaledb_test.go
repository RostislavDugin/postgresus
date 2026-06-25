package postgresql_logical

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

const timescaleImage = "timescale/timescaledb:2.17.0-pg17"

// Test_PostgresqlBackupRestore_TimescaleDB boots one timescaledb container and runs the full
// backup→restore cycle of a hypertable through it, on both the CPU=1 streaming path and the CPU=4
// parallel path (the path that previously failed with _timescaledb_catalog FK violations). See
// ADR-0013 for the one-container-per-version pattern.
func Test_PostgresqlBackupRestore_TimescaleDB(t *testing.T) {
	endpoint := containers.StartTimescaleDB(t, timescaleImage)

	for _, cpuCount := range []int{1, 4} {
		t.Run(fmt.Sprintf("CPU=%d", cpuCount), func(t *testing.T) {
			testTimescaleBackupRestoreForCpuCount(t, endpoint, cpuCount)
		})
	}
}

func seedHypertableQuery(tableName string) string {
	return fmt.Sprintf(`
CREATE EXTENSION IF NOT EXISTS timescaledb;

DROP TABLE IF EXISTS %s;

CREATE TABLE %s (
    time        TIMESTAMPTZ NOT NULL,
    sensor_id   INTEGER NOT NULL,
    temperature DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('%s', 'time');

INSERT INTO %s (time, sensor_id, temperature)
SELECT ts, (random() * 10)::int, random() * 100
FROM generate_series('2024-01-01'::timestamptz, '2024-03-01'::timestamptz, interval '1 hour') AS ts;
`, tableName, tableName, tableName, tableName)
}

func testTimescaleBackupRestoreForCpuCount(t *testing.T, endpoint containers.Endpoint, cpuCount int) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	require.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	tableName := fmt.Sprintf("sensor_data_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(seedHypertableQuery(tableName))
	require.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	}()

	var sourceRowCount int
	require.NoError(t, container.DB.Get(&sourceRowCount, fmt.Sprintf("SELECT count(*) FROM %s", tableName)))
	require.Positive(t, sourceRowCount)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithCpuCountViaAPI(
		t, router, "Timescale Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		cpuCount,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)
	assert.NotEmpty(t, backup.TimescaledbVersion, "backup should record the source timescaledb version")

	newDBName := fmt.Sprintf("restored_ts_cpu%d_%s", cpuCount, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	require.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	require.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	restoredDB, err := sqlx.Connect("postgres", newDSN)
	require.NoError(t, err)
	defer restoredDB.Close()

	createRestoreWithCpuCountViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		cpuCount,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	assertHypertableRestored(t, restoredDB, tableName, sourceRowCount)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t, router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(storage.ID)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func assertHypertableRestored(t *testing.T, restoredDB *sqlx.DB, tableName string, expectedRows int) {
	t.Helper()

	var restoredRowCount int
	require.NoError(t, restoredDB.Get(&restoredRowCount, fmt.Sprintf("SELECT count(*) FROM %s", tableName)))
	assert.Equal(t, expectedRows, restoredRowCount, "restored hypertable should keep every row")

	var hypertableCount int
	require.NoError(t, restoredDB.Get(&hypertableCount,
		"SELECT count(*) FROM timescaledb_information.hypertables WHERE hypertable_name = $1", tableName))
	assert.Equal(t, 1, hypertableCount, "restored table should be a hypertable")

	var chunkCount int
	require.NoError(t, restoredDB.Get(&chunkCount,
		"SELECT count(*) FROM timescaledb_information.chunks WHERE hypertable_name = $1", tableName))
	assert.Greater(t, chunkCount, 1, "restored hypertable should keep its chunks")
}
