package usecases_physical_postgresql

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The watcher polls on byteStallPollInterval (10 s), so each test below spends a
// real tick or two. The thresholds are picked around that tick rather than
// stubbing it: the poll cadence is part of the behaviour under test.

func Test_ByteStallWatcher_WhenNoBytesBeforeFirstByteTimeout_DoesNotTrip(t *testing.T) {
	counter := NewByteCounter(&bytes.Buffer{})

	var isStalled atomic.Bool

	stopWatcher := WithByteStallWatcher(t.Context(), counter, ByteStallThresholds{
		FirstByteTimeout: time.Hour,
		StallTimeout:     time.Millisecond,
	}, func() { isStalled.Store(true) })
	defer stopWatcher()

	time.Sleep(byteStallPollInterval + 2*time.Second)

	require.False(t, isStalled.Load(),
		"zero bytes within the first-byte budget must not trip, even long past the steady-state budget")
}

func Test_ByteStallWatcher_WhenFirstByteTimeoutExceeded_Trips(t *testing.T) {
	counter := NewByteCounter(&bytes.Buffer{})

	stalled := make(chan struct{})

	stopWatcher := WithByteStallWatcher(t.Context(), counter, ByteStallThresholds{
		FirstByteTimeout: time.Millisecond,
		StallTimeout:     time.Hour,
	}, func() { close(stalled) })
	defer stopWatcher()

	select {
	case <-stalled:
	case <-time.After(byteStallPollInterval + 5*time.Second):
		t.Fatal("watcher did not trip after the first-byte budget elapsed with zero bytes")
	}
}

func Test_ByteStallWatcher_WhenStallAfterFirstByte_TripsOnStallTimeout(t *testing.T) {
	counter := NewByteCounter(&bytes.Buffer{})

	_, err := counter.Write([]byte("x"))
	require.NoError(t, err)

	stalled := make(chan struct{})

	stopWatcher := WithByteStallWatcher(t.Context(), counter, ByteStallThresholds{
		FirstByteTimeout: time.Hour,
		StallTimeout:     time.Millisecond,
	}, func() { close(stalled) })
	defer stopWatcher()

	select {
	case <-stalled:
	case <-time.After(byteStallPollInterval + 5*time.Second):
		t.Fatal("the first-byte leniency must end with the first byte, not extend the steady-state budget")
	}
}
