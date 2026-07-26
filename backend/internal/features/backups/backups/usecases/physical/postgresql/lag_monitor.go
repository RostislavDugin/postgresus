package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	// Balances detection latency against query load on the source (one cheap
	// indexed row).
	lagMonitorPollInterval = 30 * time.Second

	// A warning wal_status must persist this long before we alert on it (vs a
	// transient write burst).
	warningSlotStatusHoldPeriod = 5 * time.Minute

	// Beyond this many rebuild ATTEMPTS in a sliding hour (counted regardless of
	// outcome), mechanical retry won't help (creds rotated, pg_hba changed,
	// source dead); stop and surface the condition instead of dropping+recreating
	// in a loop.
	slotRebuildMaxAttemptsPerHour = 3

	// How long to wait for our own pg_receivewal to release the slot during a
	// rebuild before concluding another consumer holds it.
	rebuildReceiverStopTimeout = 30 * time.Second
	rebuildReceiverStopPoll    = 1 * time.Second
)

// NOT a catalog enum: WAL chain breaks are derived from LSN gaps between segment
// rows, never stored — the log carries the human-readable "why".
type walBreakReason string

const (
	breakReasonSlotLost      walBreakReason = "SLOT_LOST"
	breakReasonWalLag        walBreakReason = "WAL_LAG_THRESHOLD"
	breakReasonSlotStolen    walBreakReason = "SLOT_STOLEN"
	breakReasonSlotRetention walBreakReason = "SLOT_WAL_RETENTION"
)

type slotBreakAction int

const (
	slotBreakActionNone slotBreakAction = iota
	slotBreakActionAlert
	slotBreakActionRebuild
)

type slotBreakSample struct {
	SlotState            *SlotState
	WalLagThresholdBytes int64
	IsReceiverRunning    bool
	ObservedAt           time.Time
}

type slotBreakClassifier struct {
	warningStatusSince time.Time
}

// Only 'lost' and a stolen slot are unrecoverable enough to justify dropping the
// slot, which always costs a WAL gap and a fresh FULL. 'extended' merely means
// the slot retains WAL beyond max_wal_size — routine on a busy cluster, and
// guaranteed whenever back pressure pauses us — and 'unreserved' means PG will
// trim that WAL at the next checkpoint, which already protects the primary. Both
// alert instead. The operator's WalLagThresholdBytes stays the explicit
// "sacrifice the chain to protect the primary" knob, but never fires against lag
// we caused by holding the receiver down ourselves.
func (c *slotBreakClassifier) recordSampleAndClassifyBreak(
	sample slotBreakSample,
) (walBreakReason, slotBreakAction) {
	state := sample.SlotState

	if state == nil {
		return "", slotBreakActionNone
	}

	// A foreign backend holding our slot (active, but not one of our own
	// pg_receivewal processes) blocks our receiver from ever attaching. Surface
	// it as SLOT_STOLEN and let the rebuild path decide — terminateOwnedSlotBackend
	// refuses to drop a slot held by a consumer we cannot attribute, so a genuine
	// third party trips loop-protection rather than getting force-dropped.
	if state.Active && !isOwnedReceiverBackend(state) {
		return breakReasonSlotStolen, slotBreakActionRebuild
	}

	if state.WalStatus == "lost" {
		c.warningStatusSince = time.Time{}

		return breakReasonSlotLost, slotBreakActionRebuild
	}

	if state.WalStatus == "extended" || state.WalStatus == "unreserved" {
		if c.warningStatusSince.IsZero() {
			c.warningStatusSince = sample.ObservedAt
		}

		if sample.ObservedAt.Sub(c.warningStatusSince) > warningSlotStatusHoldPeriod {
			return breakReasonSlotRetention, slotBreakActionAlert
		}

		return "", slotBreakActionNone
	}

	c.warningStatusSince = time.Time{}

	if sample.WalLagThresholdBytes > 0 &&
		state.LagBytes > sample.WalLagThresholdBytes &&
		sample.IsReceiverRunning {
		return breakReasonWalLag, slotBreakActionRebuild
	}

	return "", slotBreakActionNone
}

// Source-side slot state only; consumer-side liveness is the slot-LSN watcher's
// job (slot_lsn_watcher.go).
func (s *WalStreamSupervisor) runLagMonitor(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(lagMonitorPollInterval)
	defer ticker.Stop()

	var classifier slotBreakClassifier

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			state, isSourceReachable := s.GetSlotStateIfReachable(ctx, logger)
			if !isSourceReachable {
				continue
			}

			reason, action := classifier.recordSampleAndClassifyBreak(slotBreakSample{
				SlotState:            state,
				WalLagThresholdBytes: s.spec.WalLagThresholdBytes,
				IsReceiverRunning:    s.isReceiverRunning.Load(),
				ObservedAt:           time.Now().UTC(),
			})

			switch action {
			case slotBreakActionNone:
				continue

			case slotBreakActionAlert:
				s.reportChainAtRisk(logger, reason, state)

			case slotBreakActionRebuild:
				logger.Warn("wal_stream_break_observed", "reason", string(reason), "slot", s.slotName)

				if err := s.rebuildSlot(ctx, logger, reason); err != nil {
					logger.Error("slot rebuild failed", "reason", string(reason), "error", err)
				}
			}
		}
	}
}

func (s *WalStreamSupervisor) reportChainAtRisk(logger *slog.Logger, reason walBreakReason, state *SlotState) {
	logger.Warn(
		fmt.Sprintf("wal chain at risk: slot wal_status=%s, %d bytes behind", state.WalStatus, state.LagBytes),
		"reason", string(reason),
		"slot", s.slotName,
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(ChainRiskReport{
		Reason:        string(reason),
		SlotWalStatus: state.WalStatus,
		LagBytes:      state.LagBytes,
	})
}
