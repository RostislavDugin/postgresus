package restores_core

import (
	"fmt"

	io_utils "databasus-backend/internal/util/io"
)

// GetBackupStreamFailure must be consulted before the restore client's own exit error:
// a truncated download makes the client fail on the half-written statement it was fed,
// and that downstream syntax error would otherwise be reported as the cause.
func GetBackupStreamFailure(backupStreamReader *io_utils.FailureTrackingReader) error {
	readError := backupStreamReader.GetFirstReadError()
	if readError == nil {
		return nil
	}

	return fmt.Errorf(
		"backup stream from storage failed mid-restore, target database is left partially restored: %w",
		readError,
	)
}
