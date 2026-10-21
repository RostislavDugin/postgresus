package usecases_mongodb

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
)

func Test_ExecuteMongoRestore_WhenToolExitsRightAfterError_ReportsToolError(t *testing.T) {
	restoreUsecase := &RestoreMongodbBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMongoRestore(
		t.Context(),
		restores_core.WriteLateFailingToolStub(
			t, "mongorestore-stub", "Failed: restore error: orders: error restoring from archive: unexpected EOF",
		),
		nil,
		io.NopCloser(bytes.NewReader([]byte("archive"))),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.Contains(t, restoreErr.Error(), "error restoring from archive")
}
