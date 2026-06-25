package usecases_physical_postgresql

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// progressRecorder collects the (sizeMb, elapsedMs) the reporter emits.
type progressRecorder struct {
	mu      sync.Mutex
	reports []float64
}

func (r *progressRecorder) record(completedMb float64, _ int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reports = append(r.reports, completedMb)
}

func (r *progressRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.reports)
}

func (r *progressRecorder) last() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.reports) == 0 {
		return 0
	}

	return r.reports[len(r.reports)-1]
}

func Test_WithProgressReporter_WhenBytesAdvance_ReportsIncrementalProgress(t *testing.T) {
	const oneMB = 1024 * 1024

	counter := NewByteCounter(io.Discard)
	recorder := &progressRecorder{}

	var lastElapsed int64

	stop := WithProgressReporter(t.Context(), counter, 10*time.Millisecond,
		func(completedMb float64, elapsedMs int64) {
			lastElapsed = elapsedMs
			recorder.record(completedMb, elapsedMs)
		})

	// First chunk: expect at least one report around 1 MB.
	_, err := counter.Write(make([]byte, oneMB))
	require.NoError(t, err)

	require.Eventually(t, func() bool { return recorder.last() >= 1.0 }, time.Second, 5*time.Millisecond,
		"reporter must emit progress after the first MB is written")

	// Second chunk: progress must climb, proving it is incremental — not a single
	// final value.
	_, err = counter.Write(make([]byte, 2*oneMB))
	require.NoError(t, err)

	require.Eventually(t, func() bool { return recorder.last() >= 3.0 }, time.Second, 5*time.Millisecond,
		"reporter must report the advanced byte count, not a stale value")

	assert.Positive(t, lastElapsed, "elapsed must be measured from stream start")

	stop()

	// No reports after stop, even if bytes keep moving.
	reportsAtStop := recorder.count()
	_, err = counter.Write(make([]byte, oneMB))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, reportsAtStop, recorder.count(), "no progress must be reported after stop()")
}
