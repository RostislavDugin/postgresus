package usecases_physical_postgresql

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"

	"databasus-backend/internal/util/walmath"
)

// A finalized segment waiting in the watch directory or in pending-upload/, including one whose
// upload is running: its file stays in place until the upload commits.
type queuedWalSegment struct {
	walFilename string
	localPath   string
	timelineID  int
	startLSN    walmath.LSN
	endLSN      walmath.LSN
	isComplete  bool
}

// The watch directory is listed before pending-upload/. realignResumePath only moves files from the
// first into the second, so a file renamed between the two reads is still found in the second.
func listQueuedWalSegments(watchDir string, segmentSizeBytes int64) []queuedWalSegment {
	queuedSegments := listQueuedWalSegmentsIn(watchDir, segmentSizeBytes)

	return append(
		queuedSegments,
		listQueuedWalSegmentsIn(filepath.Join(watchDir, pendingUploadDirName), segmentSizeBytes)...)
}

func listQueuedWalSegmentsIn(directory string, segmentSizeBytes int64) []queuedWalSegment {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}

	queuedSegments := make([]queuedWalSegment, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !walmath.IsWalFilename(entry.Name()) {
			continue
		}

		timelineID, startLSN, endLSN, err := segmentBounds(entry.Name(), segmentSizeBytes)
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		queuedSegments = append(queuedSegments, queuedWalSegment{
			walFilename: entry.Name(),
			localPath:   filepath.Join(directory, entry.Name()),
			timelineID:  timelineID,
			startLSN:    startLSN,
			endLSN:      endLSN,
			isComplete:  info.Size() == segmentSizeBytes,
		})
	}

	return queuedSegments
}

// Timeline first, then position: the same order FindLatestCommittedSegment uses, so the oldest WAL
// drains first and the archive progress point moves forward.
func compareWalOrder(firstTimelineID int, firstLSN walmath.LSN, secondTimelineID int, secondLSN walmath.LSN) int {
	return cmp.Or(cmp.Compare(firstTimelineID, secondTimelineID), cmp.Compare(firstLSN, secondLSN))
}

func sortQueuedWalSegments(queuedSegments []queuedWalSegment) {
	slices.SortStableFunc(queuedSegments, func(first, second queuedWalSegment) int {
		return compareWalOrder(first.timelineID, first.startLSN, second.timelineID, second.startLSN)
	})
}
