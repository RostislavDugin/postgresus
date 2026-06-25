package usecases_physical_postgresql

import (
	"context"
	"time"
)

// progressReportInterval is how often the progress poller samples the byte
// counter and reports the streamed size to the caller. It pairs with the 10 s
// byte-stall poll: 5 s gives near-real-time size growth while a multi-hour
// backup emits only a few hundred light UPDATEs. A var, not a const, so tests
// can shorten it.
var progressReportInterval = 5 * time.Second

// WithProgressReporter polls counter.BytesWritten on a fixed interval and, when
// the count has advanced since the last sample, reports the streamed size (MB)
// and the elapsed time since the stream began. It is the size-tracking sibling
// of WithByteStallWatcher and reads the SAME atomic counter from its own
// goroutine, so it never touches the io.Copy path — a slow DB write inside the
// report callback can neither back-pressure pg_basebackup nor be mistaken for a
// byte stall.
//
// The returned stop function tears the poller down (cancel + drain). It does NOT
// flush a final report: the executor persists the authoritative final size right
// after the stream settles, so a flush would only duplicate that write.
func WithProgressReporter(
	ctx context.Context,
	counter *ByteCounter,
	interval time.Duration,
	report func(completedMb float64, elapsedMs int64),
) (stop func()) {
	watcherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	start := time.Now()

	// Seed synchronously at construction (the copy has not started yet, so the
	// count is 0) rather than inside the goroutine, which would race the writer.
	lastReported := counter.BytesWritten()

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-watcherCtx.Done():
				return

			case <-ticker.C:
				current := counter.BytesWritten()
				if current == lastReported {
					continue
				}

				lastReported = current

				report(float64(current)/(1024*1024), time.Since(start).Milliseconds())
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
