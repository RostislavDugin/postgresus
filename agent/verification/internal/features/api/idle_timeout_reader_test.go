package api

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadThroughIdleTimeoutReader_WhenBytesFlowContinuously_DoesNotCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	pr, pw := io.Pipe()

	// The writer pauses 50 ms between writes, so a timeout of seconds leaves a margin no
	// scheduler pause on a loaded runner reaches.
	idleReader := NewIdleTimeoutReader(pr, 5*time.Second, cancel)
	defer idleReader.Stop()

	go func() {
		for range 5 {
			_, _ = pw.Write([]byte("data"))
			time.Sleep(50 * time.Millisecond)
		}

		_ = pw.Close()
	}()

	data, err := io.ReadAll(idleReader)

	require.NoError(t, err)
	assert.Equal(t, "datadatadatadatadata", string(data))
	assert.NoError(t, ctx.Err(), "context should not be cancelled when bytes flow continuously")
}

func Test_ReadThroughIdleTimeoutReader_WhenNoBytesTransmitted_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	pr, _ := io.Pipe()

	idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
	defer idleReader.Stop()

	require.Eventually(t, func() bool { return ctx.Err() != nil }, 5*time.Second, 10*time.Millisecond,
		"context should be cancelled when no bytes are transmitted")
	assert.Contains(t, context.Cause(ctx).Error(), "upload idle timeout")
}

func Test_ReadThroughIdleTimeoutReader_WhenBytesStopMidStream_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	pr, pw := io.Pipe()

	idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
	defer idleReader.Stop()

	go func() {
		_, _ = pw.Write([]byte("initial"))
		// Stop writing — simulate stalled source
	}()

	buf := make([]byte, 1024)
	n, _ := idleReader.Read(buf)
	assert.Equal(t, "initial", string(buf[:n]))

	require.Eventually(t, func() bool { return ctx.Err() != nil }, 5*time.Second, 10*time.Millisecond,
		"context should be cancelled when bytes stop mid-stream")
	assert.Contains(t, context.Cause(ctx).Error(), "upload idle timeout")
}

func Test_StopIdleTimeoutReader_WhenCalledBeforeTimeout_DoesNotCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	pr, _ := io.Pipe()

	idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
	idleReader.Stop()

	time.Sleep(200 * time.Millisecond)

	assert.NoError(t, ctx.Err(), "context should not be cancelled when reader is stopped before timeout")
}

func Test_ReadThroughIdleTimeoutReader_WhenReaderReturnsError_PropagatesError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	pr, pw := io.Pipe()

	idleReader := NewIdleTimeoutReader(pr, 5*time.Second, cancel)
	defer idleReader.Stop()

	expectedErr := fmt.Errorf("test read error")
	_ = pw.CloseWithError(expectedErr)

	buf := make([]byte, 1024)
	_, err := idleReader.Read(buf)

	assert.ErrorIs(t, err, expectedErr)

	// Timer should be stopped after error — context should not be cancelled
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, ctx.Err(), "context should not be cancelled after reader error stops the timer")
}
