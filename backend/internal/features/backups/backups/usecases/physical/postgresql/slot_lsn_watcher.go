package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/util/walmath"
)

const (
	// Tighter than the lag monitor because it tests whether OUR pg_receivewal is
	// actually flushing.
	slotLsnWatcherPollInterval = 10 * time.Second

	// restart_lsn unchanged this long while WAL is waiting means pg_receivewal is
	// stuck (alive but flushing nothing); restart it locally.
	slotLsnStallTimeout = 60 * time.Second
)

type slotLivenessSample struct {
	RestartLSN        walmath.LSN
	LagBytes          int64
	IsReceiverRunning bool
	ObservedAt        time.Time
}

type stallTracker struct {
	lastRestartLSN walmath.LSN
	lastAdvanceAt  time.Time
	hasSample      bool
}

// A frozen restart_lsn is only a stall when the source has WAL we have not taken
// yet and a receiver is up to take it: an idle database legitimately parks
// restart_lsn, and so does a receiver we are deliberately holding down for a
// rebuild or back pressure. A positive result re-arms the clock, so the caller
// restarts at most once per stallTimeout window.
func (t *stallTracker) recordSampleAndDetectStall(
	sample slotLivenessSample,
	stallTimeout time.Duration,
) bool {
	if sample.LagBytes == 0 || !sample.IsReceiverRunning {
		t.reset()

		return false
	}

	if !t.hasSample || sample.RestartLSN != t.lastRestartLSN {
		t.lastRestartLSN = sample.RestartLSN
		t.lastAdvanceAt = sample.ObservedAt
		t.hasSample = true

		return false
	}

	if sample.ObservedAt.Sub(t.lastAdvanceAt) > stallTimeout {
		t.lastAdvanceAt = sample.ObservedAt

		return true
	}

	return false
}

func (t *stallTracker) reset() {
	t.hasSample = false
	t.lastRestartLSN = 0
	t.lastAdvanceAt = time.Time{}
}

// isSourceReachable=false means defer to the lag monitor.
func (s *WalStreamSupervisor) GetSlotStateIfReachable(
	ctx context.Context,
	logger *slog.Logger,
) (state *SlotState, isSourceReachable bool) {
	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		logger.DebugContext(ctx, "source unreachable, deferring to the lag monitor", "error", err)

		return nil, false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err = InspectSlot(ctx, conn, s.slotName)
	if err != nil {
		logger.DebugContext(ctx, "failed to inspect the replication slot", "error", err)

		return nil, false
	}

	if state == nil {
		// The slot is gone from the source while we still believe we are streaming from it, so WAL
		// is no longer retained for us and the chain is already at risk.
		logger.WarnContext(ctx, "the replication slot no longer exists on the source")

		return nil, false
	}

	var alive int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&alive); err != nil {
		logger.DebugContext(ctx, "source stopped responding during the slot probe", "error", err)

		return nil, false
	}

	return state, true
}

// A stall with an unreachable server is left to the lag monitor (slot loss /
// network down).
func (s *WalStreamSupervisor) runSlotLsnWatcher(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(slotLsnWatcherPollInterval)
	defer ticker.Stop()

	var tracker stallTracker

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			state, isSourceReachable := s.GetSlotStateIfReachable(ctx, logger)
			if !isSourceReachable {
				continue
			}

			sample := slotLivenessSample{
				RestartLSN:        state.RestartLSN,
				LagBytes:          state.LagBytes,
				IsReceiverRunning: s.isReceiverRunning.Load(),
				ObservedAt:        time.Now().UTC(),
			}

			if tracker.recordSampleAndDetectStall(sample, slotLsnStallTimeout) {
				logger.WarnContext(
					ctx,
					fmt.Sprintf("slot restart_lsn stalled with %d bytes pending; restarting pg_receivewal",
						state.LagBytes),
					"restart_lsn", state.RestartLSN.String(),
				)

				s.signalRestart()
			}
		}
	}
}
