package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"sync"
)

// Four uploads per database reach the rate of a busy source against a storage in another region,
// while staying under common per-user connection limits of FTP and SFTP servers and bounding the
// encoder and write-buffer memory every streaming database holds.
const maxConcurrentSegmentUploads = 4

type segmentUpload func(ctx context.Context, localPath, walFilename string) error

// Only the goroutine that owns the dispatcher touches the in-flight set; workers report back through
// finishedUploads. The set is keyed by WAL file name rather than path, because realignResumePath can
// move a file into pending-upload/ while its upload runs, and the moved copy must not start again.
type segmentUploadDispatcher struct {
	upload                  segmentUpload
	recordConcurrentUploads func(count int)

	inFlightFilenames map[string]struct{}
	finishedUploads   chan string
	workers           sync.WaitGroup
}

func newSegmentUploadDispatcher(
	upload segmentUpload,
	recordConcurrentUploads func(count int),
) *segmentUploadDispatcher {
	return &segmentUploadDispatcher{
		upload:                  upload,
		recordConcurrentUploads: recordConcurrentUploads,
		inFlightFilenames:       make(map[string]struct{}, maxConcurrentSegmentUploads),
		// Room for every worker's report, so a worker never blocks on a dispatcher that stopped
		// reading.
		finishedUploads: make(chan string, maxConcurrentSegmentUploads),
	}
}

// Starts segments in the given order while slots are free and returns the ones it could not start
// yet. A segment already in flight is skipped rather than returned.
func (d *segmentUploadDispatcher) startUploads(
	ctx context.Context,
	logger *slog.Logger,
	sortedSegments []queuedWalSegment,
) []queuedWalSegment {
	for index, segment := range sortedSegments {
		if _, isInFlight := d.inFlightFilenames[segment.walFilename]; isInFlight {
			continue
		}

		if len(d.inFlightFilenames) >= maxConcurrentSegmentUploads || ctx.Err() != nil {
			return d.dropInFlight(sortedSegments[index:])
		}

		d.inFlightFilenames[segment.walFilename] = struct{}{}
		d.recordConcurrentUploads(len(d.inFlightFilenames))

		d.workers.Go(func() {
			if err := d.upload(ctx, segment.localPath, segment.walFilename); err != nil {
				logger.WarnContext(ctx, "wal segment upload failed; will retry on a later pass",
					"wal_filename", segment.walFilename,
					"error", err,
				)
			}

			d.finishedUploads <- segment.walFilename
		})
	}

	return nil
}

func (d *segmentUploadDispatcher) dropInFlight(segments []queuedWalSegment) []queuedWalSegment {
	notStarted := make([]queuedWalSegment, 0, len(segments))

	for _, segment := range segments {
		if _, isInFlight := d.inFlightFilenames[segment.walFilename]; !isInFlight {
			notStarted = append(notStarted, segment)
		}
	}

	return notStarted
}

func (d *segmentUploadDispatcher) markFinished(walFilename string) {
	delete(d.inFlightFilenames, walFilename)
}

func (d *segmentUploadDispatcher) getInFlightCount() int {
	return len(d.inFlightFilenames)
}

func (d *segmentUploadDispatcher) waitForUploads() {
	d.workers.Wait()
}
