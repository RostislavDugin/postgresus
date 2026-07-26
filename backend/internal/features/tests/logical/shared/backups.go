package logicaltesting

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/download/download_token"
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

// The download endpoint derives Content-Length from BackupSizeMb, so a size that
// disagrees with the delivered body makes every download announce bytes it never
// sends (issue #691).
func AssertBackupSizeMatchesDownloadedBytes(
	t *testing.T,
	router *gin.Engine,
	backup *backups_core_logical.LogicalBackup,
	token string,
) {
	t.Helper()

	var downloadToken download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+token,
		nil,
		http.StatusOK,
		&downloadToken,
	)

	downloadResponse := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), downloadToken.Token),
		"",
		http.StatusOK,
	)

	downloadedBytes := int64(len(downloadResponse.Body))

	require.Equal(
		t,
		strconv.FormatInt(downloadedBytes, 10),
		downloadResponse.Headers.Get("Content-Length"),
		"announced Content-Length must match the delivered body",
	)

	require.InDelta(
		t,
		float64(downloadedBytes)/(1024*1024),
		backup.BackupSizeMb,
		0.0001,
		"reported backup size must match the bytes the download endpoint delivers",
	)
}
