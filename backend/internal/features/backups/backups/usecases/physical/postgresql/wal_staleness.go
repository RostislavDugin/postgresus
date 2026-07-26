package usecases_physical_postgresql

import (
	"log/slog"
	"time"

	"databasus-backend/internal/util/walmath"
)

// A rarely-written database can legitimately hold its newest WAL locally for a
// while, but past this the recovery point has silently stopped tracking the
// source and only the operator can decide what to do about it.
const DefaultArchiveStalenessThreshold = 30 * time.Minute

type walArchiveSample struct {
	SourceCurrentLSN    walmath.LSN
	LastCommittedWalLSN walmath.LSN
	ObservedAt          time.Time
	StalenessThreshold  time.Duration
}

type walArchiveStalenessTracker struct {
	previousSourceCurrentLSN walmath.LSN
	lastCommittedWalLSN      walmath.LSN
	hasSample                bool

	fallingBehindSince time.Time
	isStaleReported    bool
}

// Falling behind means the source keeps writing while nothing new becomes
// restorable. Both halves are needed: an idle source always leaves its open
// segment slightly ahead of the last archived one — that is the RPO forced
// rotation exists for, not an incident — and a slow archive that still delivers
// segments is not stale either. The clock survives quiet ticks, so bursty writes
// with a broken archive are still caught.
func (t *walArchiveStalenessTracker) recordSampleAndDetectStaleness(sample walArchiveSample) bool {
	if sample.StalenessThreshold <= 0 {
		*t = walArchiveStalenessTracker{}

		return false
	}

	isSourceWriting := t.hasSample && sample.SourceCurrentLSN > t.previousSourceCurrentLSN
	isArchiveProgressing := sample.LastCommittedWalLSN > t.lastCommittedWalLSN

	t.previousSourceCurrentLSN = sample.SourceCurrentLSN
	t.lastCommittedWalLSN = max(t.lastCommittedWalLSN, sample.LastCommittedWalLSN)
	t.hasSample = true

	if isArchiveProgressing {
		t.fallingBehindSince = time.Time{}
		t.isStaleReported = false

		return false
	}

	if t.fallingBehindSince.IsZero() {
		if !isSourceWriting {
			return false
		}

		t.fallingBehindSince = sample.ObservedAt

		return false
	}

	if t.isStaleReported || sample.ObservedAt.Sub(t.fallingBehindSince) < sample.StalenessThreshold {
		return false
	}

	t.isStaleReported = true

	return true
}

func (s *WalStreamSupervisor) getLastCommittedWal(logger *slog.Logger) (endLSN walmath.LSN, committedAt *time.Time) {
	segment, err := s.spec.WalSegmentRepo.FindLatestCommittedSegment(s.spec.DatabaseID)
	if err != nil {
		logger.Error("failed to read the last committed wal segment", "error", err)

		return 0, nil
	}

	if segment == nil {
		return 0, nil
	}

	receivedAt := segment.ReceivedAt

	return segment.EndLSN, &receivedAt
}
