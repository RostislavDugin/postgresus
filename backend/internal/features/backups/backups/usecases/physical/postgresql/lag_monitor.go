package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"
)

const (
	// lagMonitorPollInterval — cadence for reading pg_replication_slots. Balances
	// detection latency against query load on the source (one cheap indexed row).
	lagMonitorPollInterval = 30 * time.Second

	// extendedSlotStatusHoldPeriod — an 'extended' slot status must persist this
	// long before we treat it as a real lag (vs a transient write burst).
	extendedSlotStatusHoldPeriod = 5 * time.Minute

	// slotRebuildMaxAttemptsPerHour — beyond this many rebuild ATTEMPTS in a
	// sliding hour (counted regardless of outcome), mechanical retry won't help
	// (creds rotated, pg_hba changed, source dead); stop and surface the
	// condition instead of dropping+recreating in a loop.
	slotRebuildMaxAttemptsPerHour = 3

	// rebuildReceiverStopTimeout — how long to wait for our own pg_receivewal to
	// release the slot during a rebuild before concluding another consumer holds it.
	rebuildReceiverStopTimeout = 30 * time.Second
	rebuildReceiverStopPoll    = 1 * time.Second
)

// walBreakReason is a structured-log value for an observed stream break. It is
// NOT a catalog enum: WAL chain breaks are derived from LSN gaps between segment
// rows, never stored — the log carries the human-readable "why".
type walBreakReason string

const (
	breakReasonSlotLost   walBreakReason = "SLOT_LOST"
	breakReasonWalLag     walBreakReason = "WAL_LAG_THRESHOLD"
	breakReasonSlotStolen walBreakReason = "SLOT_STOLEN"
)

// runLagMonitor polls the source slot and triggers a rebuild on slot loss /
// unreserved / sustained-extended / lag over threshold. Source-side slot state
// only; consumer-side liveness is the slot-LSN watcher's job (wal_stream.go).
func (s *WalStreamSupervisor) runLagMonitor(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(lagMonitorPollInterval)
	defer ticker.Stop()

	var extendedSince time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			reason, shouldRebuild := s.evaluateSlotForBreak(ctx, logger, &extendedSince)
			if !shouldRebuild {
				continue
			}

			logger.Warn("wal_stream_break_observed", "reason", string(reason), "slot", s.slotName)

			if err := s.rebuildSlot(ctx, logger, reason); err != nil {
				logger.Error("slot rebuild failed", "reason", string(reason), "error", err)
			}
		}
	}
}

// evaluateSlotForBreak inspects the slot and decides whether a rebuild is due.
// extendedSince tracks how long wal_status has been 'extended' across ticks.
func (s *WalStreamSupervisor) evaluateSlotForBreak(
	ctx context.Context,
	logger *slog.Logger,
	extendedSince *time.Time,
) (walBreakReason, bool) {
	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		logger.Debug("lag monitor: source unreachable this tick", "error", err)

		return "", false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := InspectSlot(ctx, conn, s.slotName)
	if err != nil || state == nil {
		return "", false
	}

	return classifySlotBreak(state, s.spec.WalLagThresholdBytes, extendedSince)
}

func classifySlotBreak(
	state *SlotState,
	walLagThresholdBytes int64,
	extendedSince *time.Time,
) (walBreakReason, bool) {
	if state == nil {
		return "", false
	}

	// A foreign backend holding our slot (active, but not one of our own
	// pg_receivewal processes) blocks our receiver from ever attaching. Surface
	// it as SLOT_STOLEN and let the rebuild path decide — terminateOwnedSlotBackend
	// refuses to drop a slot held by a consumer we cannot attribute, so a genuine
	// third party trips loop-protection rather than getting force-dropped.
	if state.Active && !isOwnedReceiverBackend(state) {
		return breakReasonSlotStolen, true
	}

	switch state.WalStatus {
	case "lost":
		return breakReasonSlotLost, true

	case "unreserved":
		return breakReasonWalLag, true

	case "extended":
		if extendedSince.IsZero() {
			*extendedSince = time.Now().UTC()
		}

		if time.Since(*extendedSince) > extendedSlotStatusHoldPeriod {
			return breakReasonWalLag, true
		}

		return "", false

	default:
		*extendedSince = time.Time{}
	}

	if walLagThresholdBytes > 0 && state.LagBytes > walLagThresholdBytes {
		return breakReasonWalLag, true
	}

	return "", false
}
