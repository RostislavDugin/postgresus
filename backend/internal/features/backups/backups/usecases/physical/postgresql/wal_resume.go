package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"databasus-backend/internal/util/walmath"
)

const pendingUploadDirName = "pending-upload"

// Mirrors pg_receivewal's FindStreamingStart: it starts at the segment after the
// highest complete file in --directory, and falls back to the slot's restart_lsn
// only when it finds none (PG 15+).
func GetResumeSegmentNo(
	watchDir string,
	segmentSizeBytes int64,
) (resumeSegmentNo walmath.WalSegmentNo, isResumeSegmentFound bool) {
	entries, err := os.ReadDir(watchDir)
	if err != nil {
		return 0, false
	}

	var highestSegmentNo uint64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		_, segmentNo, err := walmath.ParseWALFilenameWithSize(entry.Name(), uint64(segmentSizeBytes))
		if err != nil {
			continue
		}

		if !isCompleteSegmentFile(filepath.Join(watchDir, entry.Name()), segmentSizeBytes) {
			continue
		}

		if !isResumeSegmentFound || segmentNo > highestSegmentNo {
			highestSegmentNo = segmentNo
			isResumeSegmentFound = true
		}
	}

	if !isResumeSegmentFound {
		return 0, false
	}

	return walmath.WalSegmentNo(highestSegmentNo).Next(), true
}

// pg_receivewal reads only the top level of --directory, so a pending segment
// below the slot's restart_lsn drags the resume point into WAL the server has
// already recycled. Moving rather than deleting keeps those segments archivable:
// they are valid WAL of the older chain.
func movePendingUploadsOutOfResumePath(
	watchDir string,
	resumeFloor walmath.WalSegmentNo,
	segmentSizeBytes int64,
) (movedCount int, err error) {
	entries, err := os.ReadDir(watchDir)
	if err != nil {
		return 0, fmt.Errorf("read wal watch dir: %w", err)
	}

	pendingUploadDir := filepath.Join(watchDir, pendingUploadDirName)
	if err := os.MkdirAll(pendingUploadDir, 0o700); err != nil {
		return 0, fmt.Errorf("create pending upload dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		_, segmentNo, err := walmath.ParseWALFilenameWithSize(entry.Name(), uint64(segmentSizeBytes))
		if err != nil || walmath.WalSegmentNo(segmentNo) >= resumeFloor {
			continue
		}

		segmentPath := filepath.Join(watchDir, entry.Name())

		if !isCompleteSegmentFile(segmentPath, segmentSizeBytes) {
			continue
		}

		moveErr := os.Rename(segmentPath, filepath.Join(pendingUploadDir, entry.Name()))
		if moveErr != nil && !os.IsNotExist(moveErr) {
			return movedCount, fmt.Errorf("move %s out of the resume path: %w", entry.Name(), moveErr)
		}

		movedCount++
	}

	return movedCount, nil
}

// walmath.NewWalSegmentNo divides by the package-global segment size, which is
// wrong for a cluster with a non-default wal_segment_size.
func segmentNoAtLSN(lsn walmath.LSN, segmentSizeBytes int64) walmath.WalSegmentNo {
	return walmath.WalSegmentNo(uint64(lsn) / uint64(segmentSizeBytes))
}

func (s *WalStreamSupervisor) realignResumePath(ctx context.Context, logger *slog.Logger) {
	state, isSourceReachable := s.GetSlotStateIfReachable(ctx, logger)
	if !isSourceReachable {
		return
	}

	segmentSizeBytes := walSegmentSizeBytes(s.spec.SourceDB)

	resumeSegmentNo, isResumeSegmentFound := GetResumeSegmentNo(s.watchDir, segmentSizeBytes)
	if !isResumeSegmentFound {
		return
	}

	slotSegmentNo := segmentNoAtLSN(state.RestartLSN, segmentSizeBytes)
	if resumeSegmentNo >= slotSegmentNo {
		return
	}

	movedCount, err := movePendingUploadsOutOfResumePath(s.watchDir, slotSegmentNo, segmentSizeBytes)
	if err != nil {
		logger.ErrorContext(ctx, "failed to realign wal resume point", "error", err)

		return
	}

	logger.WarnContext(
		ctx,
		fmt.Sprintf("realigned wal resume point, moved %d segments out of the resume path", movedCount),
		"slot_restart_lsn", state.RestartLSN.String(),
		"resume_segment_no", uint64(resumeSegmentNo),
	)
}
