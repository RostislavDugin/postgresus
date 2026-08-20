package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/util/walmath"
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
	breakReasonSlotLost       walBreakReason = "SLOT_LOST"
	breakReasonWalLag         walBreakReason = "WAL_LAG_THRESHOLD"
	breakReasonSlotStolen     walBreakReason = "SLOT_STOLEN"
	breakReasonSlotRetention  walBreakReason = ChainRiskReasonSlotRetention
	breakReasonRotationDenied walBreakReason = ChainRiskReasonRotationDenied
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

	var stalenessTracker walArchiveStalenessTracker

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			state, isSourceReachable := s.GetSlotStateIfReachable(ctx, logger)
			if !isSourceReachable {
				continue
			}

			s.reportArchivingProgress(ctx, logger, state)

			s.reportArchiveStalenessIfDue(logger, &stalenessTracker, state)

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
				logger.WarnContext(ctx, "wal stream break observed", "reason", string(reason),
					"lag_bytes", state.LagBytes, "wal_status", state.WalStatus)

				if err := s.rebuildSlot(ctx, logger, reason); err != nil {
					logger.ErrorContext(ctx, "slot rebuild failed", "reason", string(reason), "error", err)
				}
			}
		}
	}
}

// Archiving throughput and lag are only visible today once something has already broken, which
// leaves no history to read after an incident. This rides the existing poll rather than adding a
// ticker, and stays at INFO only when segments actually moved.
func (s *WalStreamSupervisor) reportArchivingProgress(
	ctx context.Context,
	logger *slog.Logger,
	state *SlotState,
) {
	archivedSegments, archivedBytes := s.uploader.TakeArchivedSinceLastReport()

	if archivedSegments == 0 {
		logger.DebugContext(ctx, fmt.Sprintf("no wal archived since the last report, %d bytes behind",
			state.LagBytes), "wal_status", state.WalStatus)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("archived %d wal segments (%.1f MB), %d bytes behind",
		archivedSegments, float64(archivedBytes)/(1024*1024), state.LagBytes),
		"wal_status", state.WalStatus)
}

// The slot's restart_lsn plus its lag is where the source currently is, so the
// staleness check needs no extra query on the source.
func (s *WalStreamSupervisor) reportArchiveStalenessIfDue(
	logger *slog.Logger,
	stalenessTracker *walArchiveStalenessTracker,
	state *SlotState,
) {
	lastCommittedWalLSN, lastCommittedWalAt := s.getLastCommittedWal(logger)

	isArchiveStale := stalenessTracker.recordSampleAndDetectStaleness(walArchiveSample{
		SourceCurrentLSN:    state.RestartLSN + walmath.LSN(state.LagBytes),
		LastCommittedWalLSN: lastCommittedWalLSN,
		ObservedAt:          time.Now().UTC(),
		StalenessThreshold:  s.spec.ArchiveStalenessThreshold,
	})

	if !isArchiveStale {
		return
	}

	logger.Warn("wal archiving fell behind the source; the recovery point is no longer advancing",
		"reason", ChainRiskReasonArchiveStale,
		"lag_bytes", state.LagBytes,
		"wal_status", state.WalStatus,
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(ChainRiskReport{
		Reason:            ChainRiskReasonArchiveStale,
		SlotWalStatus:     state.WalStatus,
		LagBytes:          state.LagBytes,
		LastArchivedWalAt: lastCommittedWalAt,
	})
}

// state is nil for a risk that is not about the slot at all (a refused rotation).
func (s *WalStreamSupervisor) reportChainAtRisk(logger *slog.Logger, reason walBreakReason, state *SlotState) {
	report := ChainRiskReport{Reason: string(reason)}

	if state != nil {
		report.SlotWalStatus = state.WalStatus
		report.LagBytes = state.LagBytes
	}

	logger.Warn(
		fmt.Sprintf("wal chain at risk: slot wal_status=%s, %d bytes behind", report.SlotWalStatus, report.LagBytes),
		"reason", string(reason),
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(report)
}
