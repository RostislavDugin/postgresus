package usecases_physical_postgresql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"sync/atomic"
	"time"
)

// ByteStallTimeout bounds how long pg_basebackup may make zero progress before
// the watcher tears it down. Kernel-pipe back pressure means a healthy slow
// storage still ticks bytes forward; zero bytes for this long is a stuck TCP
// connection (mid-write NAT timeout, silently-dropped firewall) that
// pg_basebackup's own session-level keepalives won't notice for tens of
// minutes.
const ByteStallTimeout = 60 * time.Second

// byteStallPollInterval — the watcher reads BytesWritten on this cadence.
// Tight enough that we detect a stall within ~10 s of crossing the timeout,
// cheap enough that we don't add measurable CPU during a multi-hour backup.
const byteStallPollInterval = 10 * time.Second

// PgBasebackupWaitTimeout bounds how long we wait for cmd.Wait() to return
// after the byte-stall watcher (or a storage-save failure) has triggered
// cmd.Process.Kill(). A SIGKILL'd Postgres process normally reaps within
// hundreds of milliseconds; if Wait is still blocked after this much time,
// stderr-reading goroutines are pinned on a pipe the kernel won't drain
// and the test would otherwise hang to the package-level timeout (15m).
// 30 s is generous enough for stressed CI to settle without dragging the
// suite.
const PgBasebackupWaitTimeout = 30 * time.Second

// ByteCounter wraps an io.Writer and tracks bytes written atomically so a
// watcher goroutine can read the count without racing the writer goroutine.
// CountingWriter in util/io is the single-threaded equivalent — physical
// backups need atomic semantics because the byte-stall watcher reads from a
// goroutine separate from the pg_basebackup-to-storage copy.
type ByteCounter struct {
	writer io.Writer
	bytes  atomic.Int64
}

func NewByteCounter(w io.Writer) *ByteCounter {
	return &ByteCounter{writer: w}
}

func (b *ByteCounter) Write(p []byte) (int, error) {
	n, err := b.writer.Write(p)
	if n > 0 {
		b.bytes.Add(int64(n))
	}

	return n, err
}

func (b *ByteCounter) BytesWritten() int64 {
	return b.bytes.Load()
}

// WithByteStallWatcher polls counter.BytesWritten on a fixed interval and
// invokes onStall when no bytes have moved for stallTimeout. The returned
// stop function tears down the watcher; the caller defers it on the
// success path so the watcher does not outlive the executor.
func WithByteStallWatcher(
	ctx context.Context,
	counter *ByteCounter,
	stallTimeout time.Duration,
	onStall func(),
) (stop func()) {
	watcherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(byteStallPollInterval)
		defer ticker.Stop()

		lastBytes := counter.BytesWritten()
		lastProgressAt := time.Now()

		for {
			select {
			case <-watcherCtx.Done():
				return

			case now := <-ticker.C:
				current := counter.BytesWritten()
				if current != lastBytes {
					lastBytes = current
					lastProgressAt = now

					continue
				}

				if now.Sub(lastProgressAt) > stallTimeout {
					onStall()

					return
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// ErrPgBasebackupWaitTimeout is returned when cmd.Wait() failed to return
// within PgBasebackupWaitTimeout after the parent decided the process
// should exit.
var ErrPgBasebackupWaitTimeout = errors.New("pg_basebackup did not exit within wait timeout")

// waitCmdWithTimeout calls cmd.Wait() in a goroutine and bounds the wait to
// `timeout`. On timeout, returns ErrPgBasebackupWaitTimeout instead of
// blocking forever. The Wait goroutine is left running — its result is
// discarded but it will not panic because nothing else closes the cmd
// fields underneath it.
func waitCmdWithTimeout(
	cmd *exec.Cmd,
	timeout time.Duration,
	logger *slog.Logger,
	backupID string,
) error {
	waitCh := make(chan error, 1)

	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		return err

	case <-time.After(timeout):
		logger.Warn(
			"pg_basebackup cmd.Wait timeout; abandoning Wait to prevent suite-level hang",
			"backup_id", backupID,
			"timeout", timeout.String(),
		)

		return ErrPgBasebackupWaitTimeout
	}
}
