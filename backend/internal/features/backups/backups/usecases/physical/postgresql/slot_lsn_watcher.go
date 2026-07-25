package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"

	"databasus-backend/internal/util/walmath"
)

const (
	// Tighter than the lag monitor because it tests whether OUR pg_receivewal is
	// actually flushing.
	slotLsnWatcherPollInterval = 10 * time.Second

	// restart_lsn unchanged this long on a healthy PG means pg_receivewal is
	// stuck (alive but flushing nothing); restart it locally.
	slotLsnStallTimeout = 60 * time.Second
)

type stallTracker struct {
	lastRestartLSN walmath.LSN
	lastAdvanceAt  time.Time
	hasSample      bool
}

// A changed LSN (or the very first sample) re-arms the advance clock. On a
// positive result the clock re-arms too, so the caller restarts at most once per
// stallTimeout window.
func (t *stallTracker) observe(restartLSN walmath.LSN, now time.Time, stallTimeout time.Duration) bool {
	if !t.hasSample || restartLSN != t.lastRestartLSN {
		t.lastRestartLSN = restartLSN
		t.lastAdvanceAt = now
		t.hasSample = true

		return false
	}

	if now.Sub(t.lastAdvanceAt) > stallTimeout {
		t.lastAdvanceAt = now

		return true
	}

	return false
}

// A frozen restart_lsn on a reachable source is a stuck consumer on a healthy
// server. A stall with an unreachable server is left to the lag monitor (slot
// loss / network down).
func (s *WalStreamSupervisor) runSlotLsnWatcher(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(slotLsnWatcherPollInterval)
	defer ticker.Stop()

	var tracker stallTracker

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			restartLSN, pgReachable := s.sampleSlotRestartLSN(ctx, logger)
			if !pgReachable {
				continue
			}

			if tracker.observe(restartLSN, time.Now().UTC(), slotLsnStallTimeout) {
				logger.Warn("slot restart_lsn stalled on a reachable source; restarting pg_receivewal",
					"restart_lsn", restartLSN.String())

				s.signalRestart()
			}
		}
	}
}

// pgReachable=false means defer to the lag monitor.
func (s *WalStreamSupervisor) sampleSlotRestartLSN(ctx context.Context, logger *slog.Logger) (walmath.LSN, bool) {
	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		logger.Debug("slot-lsn watcher: source unreachable, deferring to lag monitor", "error", err)

		return 0, false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := InspectSlot(ctx, conn, s.slotName)
	if err != nil || state == nil {
		return 0, false
	}

	var alive int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&alive); err != nil {
		return 0, false
	}

	return state.RestartLSN, true
}
