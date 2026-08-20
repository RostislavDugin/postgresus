package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"databasus-backend/internal/util/walmath"
)

const (
	// pg_receivewal has no kernel-pipe back pressure (it writes files), so the
	// watch dir IS the buffer. Above HIGH we SIGTERM it; we resume only once
	// uploads drain below LOW. The 5x hysteresis prevents flapping on the
	// boundary. HIGH scales up for clusters with non-default wal_segment_size so
	// one segment does not stop the receiver.
	walLocalMinHighWatermarkBytes int64 = 512 * 1024 * 1024

	backpressurePollInterval = 1 * time.Second
)

func (s *WalStreamSupervisor) runBackpressureMonitor(ctx context.Context, _ *slog.Logger) {
	ticker := time.NewTicker(backpressurePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if s.backlogBytes() >= s.highWatermarkBytes {
				s.signalRestart()
			}
		}
	}
}

func (s *WalStreamSupervisor) waitForBacklogBelowLow(ctx context.Context, logger *slog.Logger) bool {
	if s.backlogBytes() < s.highWatermarkBytes {
		return true
	}

	logger.WarnContext(ctx, "wal backlog over high watermark; pausing pg_receivewal until it drains")

	ticker := time.NewTicker(backpressurePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false

		case <-ticker.C:
			if s.backlogBytes() < s.lowWatermarkBytes {
				logger.InfoContext(ctx, "wal backlog drained below low watermark; resuming pg_receivewal")

				return true
			}
		}
	}
}

// Uploaded segments are removed by the uploader, so summing what is left is the
// local queue depth; .partial and .history files are excluded. Segments staged out
// of pg_receivewal's resume path (wal_resume.go) are still waiting for the same
// uploader on the same disk, so they count too — otherwise a storage outage plus a
// realign grows the queue past the watermark with nothing throttling the receiver.
func (s *WalStreamSupervisor) backlogBytes() int64 {
	return sumWalSegmentBytes(s.watchDir) + sumWalSegmentBytes(filepath.Join(s.watchDir, pendingUploadDirName))
}

func sumWalSegmentBytes(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	var total int64

	for _, entry := range entries {
		if entry.IsDir() || !walmath.IsWalFilename(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		total += info.Size()
	}

	return total
}
