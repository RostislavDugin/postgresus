package usecases_physical_postgresql

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/walmath"
)

func writeSegmentOfSize(t *testing.T, dir, name string, sizeBytes int64) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte{0}, int(sizeBytes)), 0o600))
}

func Test_GetResumeSegmentNo_WithCompletePartialAndTornFiles_ReturnsHighestCompletePlusOne(t *testing.T) {
	watchDir := t.TempDir()

	writeSegmentOfSize(t, watchDir, walName(1, 40), testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, walName(1, 41), testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, walName(1, 42), testWalSegmentSize/2)
	writeSegmentOfSize(t, watchDir, walName(1, 43)+".partial", testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, "00000002.history", 64)

	resumeSegmentNo, isResumeSegmentFound := GetResumeSegmentNo(watchDir, testWalSegmentSize)

	require.True(t, isResumeSegmentFound)
	require.Equal(t, walmath.WalSegmentNo(42), resumeSegmentNo,
		"a torn segment is re-streamed, so 42 is not a resume point — 41 is the highest complete one")
}

func Test_GetResumeSegmentNo_WhenWatchDirHasNoSegments_ReportsNotFound(t *testing.T) {
	watchDir := t.TempDir()

	writeSegmentOfSize(t, watchDir, "00000002.history", 64)
	require.NoError(t, os.MkdirAll(filepath.Join(watchDir, pendingUploadDirName), 0o700))
	writeSegmentOfSize(t, filepath.Join(watchDir, pendingUploadDirName), walName(1, 40), testWalSegmentSize)

	_, isResumeSegmentFound := GetResumeSegmentNo(watchDir, testWalSegmentSize)

	require.False(t, isResumeSegmentFound,
		"pg_receivewal reads only the top level, so a staged segment must not anchor the resume point")
}

func Test_MovePendingUploadsOutOfResumePath_MovesOnlySegmentsBelowFloor(t *testing.T) {
	watchDir := t.TempDir()

	for _, segmentNo := range []uint64{40, 41, 42, 43} {
		writeSegmentOfSize(t, watchDir, walName(1, segmentNo), testWalSegmentSize)
	}

	movedCount, err := movePendingUploadsOutOfResumePath(watchDir, walmath.WalSegmentNo(42), testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, 2, movedCount)

	for _, movedSegment := range []uint64{40, 41} {
		require.NoFileExists(t, filepath.Join(watchDir, walName(1, movedSegment)))
		require.FileExists(t, filepath.Join(watchDir, pendingUploadDirName, walName(1, movedSegment)))
	}

	for _, keptSegment := range []uint64{42, 43} {
		require.FileExists(t, filepath.Join(watchDir, walName(1, keptSegment)))
	}
}

func Test_MovePendingUploadsOutOfResumePath_LeavesHistoryFilesAndSubdirectories(t *testing.T) {
	watchDir := t.TempDir()

	writeSegmentOfSize(t, watchDir, "00000002.history", 64)
	require.NoError(t, os.MkdirAll(filepath.Join(watchDir, "archive_status"), 0o700))

	movedCount, err := movePendingUploadsOutOfResumePath(watchDir, walmath.WalSegmentNo(99), testWalSegmentSize)
	require.NoError(t, err)
	require.Zero(t, movedCount)

	require.FileExists(t, filepath.Join(watchDir, "00000002.history"))
	require.DirExists(t, filepath.Join(watchDir, "archive_status"))
}

func Test_SegmentNoAtLsn_WithNonDefaultSegmentSize_UsesTheClusterSize(t *testing.T) {
	const customSegmentSize = int64(64 * 1024 * 1024)

	lsn := walmath.LSN(3 * uint64(customSegmentSize))

	require.Equal(t, walmath.WalSegmentNo(3), segmentNoAtLSN(lsn, customSegmentSize),
		"the package-global segment size would give 12 here")
}
