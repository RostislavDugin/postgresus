package logicaltesting

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/storages"
	test_utils "databasus-backend/internal/util/testing"
)

// EnableBackupsViaAPI turns on backups for a database with the given storage and
// encryption via the backup-config API.
func EnableBackupsViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	encryption backups_core_enums.BackupEncryption,
	token string,
) {
	var backupConfig backups_config_logical.LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/database/%s", databaseID.String()),
		"Bearer "+token,
		http.StatusOK,
		&backupConfig,
	)

	storage := &storages.Storage{ID: storageID}
	backupConfig.IsBackupsEnabled = true
	backupConfig.Storage = storage
	backupConfig.Encryption = encryption

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+token,
		backupConfig,
		http.StatusOK,
	)
}

// CreateBackupViaAPI triggers an immediate backup for a database.
func CreateBackupViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
) {
	request := backups_dto_logical.MakeBackupRequest{DatabaseID: databaseID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+token,
		request,
		http.StatusOK,
	)
}

// WaitForBackupCompletion polls the backups API until the latest backup for the
// database reaches Completed, failing the test on Failed or timeout.
func WaitForBackupCompletion(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
	timeout time.Duration,
) *backups_core_logical.LogicalBackup {
	startTime := time.Now()
	pollInterval := 500 * time.Millisecond

	for {
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout waiting for backup completion after %v", timeout)
		}

		var response backups_dto_logical.GetBackupsResponse
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/backups?database_id=%s&limit=1", databaseID.String()),
			"Bearer "+token,
			http.StatusOK,
			&response,
		)

		if len(response.Backups) > 0 {
			backup := response.Backups[0]
			if backup.Status == backups_core_logical.BackupStatusCompleted {
				return backup
			}
			if backup.Status == backups_core_logical.BackupStatusFailed {
				failMsg := "unknown error"
				if backup.FailMessage != nil {
					failMsg = *backup.FailMessage
				}
				t.Fatalf("Backup failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

// WaitForBackupTerminalStatus polls until the latest backup reaches any terminal
// status (Completed, Failed, Canceled) and returns it, failing on timeout. Used
// by the issue-582 regression which asserts a failed backup does not hang.
func WaitForBackupTerminalStatus(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
	timeout time.Duration,
) *backups_core_logical.LogicalBackup {
	deadline := time.Now().UTC().Add(timeout)
	pollInterval := 500 * time.Millisecond

	for time.Now().UTC().Before(deadline) {
		var response backups_dto_logical.GetBackupsResponse
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/backups?database_id=%s&limit=1", databaseID.String()),
			"Bearer "+token,
			http.StatusOK,
			&response,
		)

		if len(response.Backups) > 0 {
			b := response.Backups[0]
			if b.Status == backups_core_logical.BackupStatusCompleted ||
				b.Status == backups_core_logical.BackupStatusFailed ||
				b.Status == backups_core_logical.BackupStatusCanceled {
				return b
			}
		}

		time.Sleep(pollInterval)
	}

	t.Fatalf(
		"backup for database %s did not reach a terminal status within %v "+
			"(issue #582: backup hangs forever when SaveFile fails)",
		databaseID,
		timeout,
	)

	return nil
}
