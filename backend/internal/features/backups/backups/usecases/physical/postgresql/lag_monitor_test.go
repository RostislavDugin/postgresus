package usecases_physical_postgresql

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func runningSlotBreakSample(state *SlotState, walLagThresholdBytes int64, observedAt time.Time) slotBreakSample {
	return slotBreakSample{
		SlotState:            state,
		WalLagThresholdBytes: walLagThresholdBytes,
		IsReceiverRunning:    true,
		ObservedAt:           observedAt,
	}
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusLost_Rebuilds(t *testing.T) {
	var classifier slotBreakClassifier

	reason, action := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(&SlotState{WalStatus: "lost"}, 0, time.Now().UTC()),
	)

	require.Equal(t, slotBreakActionRebuild, action)
	require.Equal(t, breakReasonSlotLost, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusUnreserved_AlertsWithoutRebuilding(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()
	state := &SlotState{WalStatus: "unreserved"}

	_, immediateAction := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, firstSeenAt))
	require.Equal(t, slotBreakActionNone, immediateAction,
		"unreserved is a warning; it must hold before it is worth telling anyone about")

	reason, heldAction := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(state, 0, firstSeenAt.Add(warningSlotStatusHoldPeriod+time.Second)),
	)

	require.Equal(t, slotBreakActionAlert, heldAction,
		"PG trims that WAL at the next checkpoint, so dropping the slot would only add a gap")
	require.Equal(t, breakReasonSlotRetention, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusExtendedPastHold_AlertsWithoutRebuilding(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()
	state := &SlotState{WalStatus: "extended"}

	classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, firstSeenAt))

	reason, action := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(state, 0, firstSeenAt.Add(warningSlotStatusHoldPeriod+time.Second)),
	)

	require.Equal(t, slotBreakActionAlert, action,
		"retaining WAL beyond max_wal_size is routine on a busy cluster, not a reason to drop the slot")
	require.Equal(t, breakReasonSlotRetention, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenLagOverThresholdAndReceiverRunning_Rebuilds(t *testing.T) {
	var classifier slotBreakClassifier

	reason, action := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(&SlotState{WalStatus: "reserved", LagBytes: 101}, 100, time.Now().UTC()),
	)

	require.Equal(t, slotBreakActionRebuild, action)
	require.Equal(t, breakReasonWalLag, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenLagOverThresholdAndReceiverStopped_DoesNotRebuild(t *testing.T) {
	var classifier slotBreakClassifier

	_, action := classifier.recordSampleAndClassifyBreak(slotBreakSample{
		SlotState:            &SlotState{WalStatus: "reserved", LagBytes: 101},
		WalLagThresholdBytes: 100,
		IsReceiverRunning:    false,
		ObservedAt:           time.Now().UTC(),
	})

	require.Equal(t, slotBreakActionNone, action,
		"lag we caused by holding the receiver down must never cost the chain")
}

func Test_RecordSampleAndClassifyBreak_WhenForeignConsumerHoldsSlot_Rebuilds(t *testing.T) {
	var classifier slotBreakClassifier

	state := &SlotState{
		WalStatus:       "reserved",
		Active:          true,
		ActivePID:       new(4242),
		ApplicationName: "some_other_consumer",
	}

	reason, action := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, time.Now().UTC()))

	require.Equal(t, slotBreakActionRebuild, action)
	require.Equal(t, breakReasonSlotStolen, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenOurReceiverActive_DoesNotRebuild(t *testing.T) {
	var classifier slotBreakClassifier

	state := &SlotState{
		WalStatus:       "reserved",
		Active:          true,
		ActivePID:       new(17),
		ApplicationName: receivewalApplicationNamePrefix + "db-1",
		LagBytes:        10,
	}

	_, action := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 100, time.Now().UTC()))

	require.Equal(t, slotBreakActionNone, action)
}

func Test_RecordSampleAndClassifyBreak_WhenSlotHealthy_DoesNotRebuildAndClearsWarningClock(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()

	classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(&SlotState{WalStatus: "extended"}, 100, firstSeenAt),
	)

	_, healthyAction := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(&SlotState{WalStatus: "reserved", LagBytes: 10}, 100, firstSeenAt.Add(time.Minute)),
	)
	require.Equal(t, slotBreakActionNone, healthyAction)

	_, actionAfterRecovery := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(
			&SlotState{WalStatus: "extended"}, 100, firstSeenAt.Add(warningSlotStatusHoldPeriod+time.Second),
		),
	)

	require.Equal(t, slotBreakActionNone, actionAfterRecovery,
		"a recovered slot restarts the hold window instead of inheriting the previous one")
}

func newSupervisorReportingTo(log *capturedLog) (*WalStreamSupervisor, *slog.Logger) {
	return &WalStreamSupervisor{uploader: NewWalUploader(WalUploadDeps{})},
		slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func Test_ReportArchivingProgress_AfterBusyInterval_LogsCountsStepTimingsAndConcurrency(t *testing.T) {
	log := &capturedLog{}
	supervisor, supervisorLogger := newSupervisorReportingTo(log)

	supervisor.uploader.stats.recordArchived(3*1024*1024, segmentUploadTimings{
		Claim:        4 * time.Millisecond,
		SegmentWrite: 300 * time.Millisecond,
		SidecarWrite: 60 * time.Millisecond,
		Commit:       8 * time.Millisecond,
	})
	supervisor.uploader.stats.recordArchived(1024*1024, segmentUploadTimings{
		Claim:        2 * time.Millisecond,
		SegmentWrite: 100 * time.Millisecond,
		SidecarWrite: 40 * time.Millisecond,
		Commit:       4 * time.Millisecond,
	})
	supervisor.uploader.RecordConcurrentUploads(4)

	supervisor.reportArchivingProgress(t.Context(), supervisorLogger, &SlotState{LagBytes: 2048, WalStatus: "reserved"})

	logLine := log.String()
	require.Contains(t, logLine, "level=INFO")
	require.Contains(t, logLine, "archived 2 wal segments (4.0 MB)")
	require.Contains(t, logLine, "up to 4 uploads at once")
	require.Contains(t, logLine, "claim avg 3ms max 4ms")
	require.Contains(t, logLine, "segment write avg 200ms max 300ms")
	require.Contains(t, logLine, "sidecar write avg 50ms max 60ms")
	require.Contains(t, logLine, "commit avg 6ms max 8ms")
	require.Contains(t, logLine, "2048 bytes behind")
}

func Test_ReportArchivingProgress_AfterQuietInterval_LogsOnlyBytesBehindAtDebug(t *testing.T) {
	log := &capturedLog{}
	supervisor, supervisorLogger := newSupervisorReportingTo(log)

	supervisor.reportArchivingProgress(t.Context(), supervisorLogger, &SlotState{LagBytes: 4096, WalStatus: "reserved"})

	logLine := log.String()
	require.Contains(t, logLine, "level=DEBUG")
	require.Contains(t, logLine, "no wal archived since the last report, 4096 bytes behind")
	require.NotContains(t, logLine, "avg")
}
