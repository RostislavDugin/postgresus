package restores_core

import (
	"fmt"

	io_utils "databasus-backend/internal/util/io"
)

type BackupStreamTrackers struct {
	StorageRead   *io_utils.FailureTrackingReader
	DecodedStream *io_utils.FailureTrackingReader
}

// GetBackupStreamFailure must be consulted before the restore client's own exit error:
// a truncated download makes the client fail on the half-written statement it was fed,
// and that downstream syntax error would otherwise be reported as the cause.
//
// The transport failure wins over the decode failure for the same reason: a severed download
// surfaces one layer up as a decompression error, and reporting that would hide the real cause.
func GetBackupStreamFailure(trackers BackupStreamTrackers) error {
	readError := trackers.StorageRead.GetFirstReadError()
	if readError == nil {
		readError = trackers.DecodedStream.GetFirstReadError()
	}

	if readError == nil {
		return nil
	}

	return fmt.Errorf(
		"backup stream from storage failed mid-restore, target database is left partially restored: %w",
		readError,
	)
}
