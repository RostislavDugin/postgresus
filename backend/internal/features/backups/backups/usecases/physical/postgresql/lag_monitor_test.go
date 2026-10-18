package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
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
		runningSlotBreakSample(state, 0, firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second)),
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
		runningSlotBreakSample(state, 0, firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second)),
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
			&SlotState{WalStatus: "extended"}, 100, firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second),
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

	supervisor.reportArchivingProgress(
		t.Context(),
		supervisorLogger,
		&SlotState{LagBytes: 2048, WalStatus: "reserved"},
		supervisor.uploader.TakeArchivingSnapshot(),
	)

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

	supervisor.reportArchivingProgress(
		t.Context(),
		supervisorLogger,
		&SlotState{LagBytes: 4096, WalStatus: "reserved"},
		supervisor.uploader.TakeArchivingSnapshot(),
	)

	logLine := log.String()
	require.Contains(t, logLine, "level=DEBUG")
	require.Contains(t, logLine, "no wal archived since the last report, 4096 bytes behind")
	require.NotContains(t, logLine, "avg")
}

var lagMonitorTickStartedAt = time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)

func newLagMonitorTickSupervisor(
	t *testing.T,
	fixture *PhysicalDBFixture,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	walLagThresholdBytes int64,
) *WalStreamSupervisor {
	t.Helper()

	return NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:           fixture.DB.ID,
		SourceDB:             sourceDB,
		StorageID:            fixture.Storage.ID,
		FieldEncryptor:       encryption.GetFieldEncryptor(),
		WalSegmentRepo:       physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:          physical_repositories.GetWalHistoryRepository(),
		WalStreamSampleRepo:  physical_repositories.GetWalStreamSampleRepository(),
		WalStreamerRepo:      physical_repositories.GetWalStreamerRepository(),
		WatchDirRoot:         t.TempDir(),
		WalLagThresholdBytes: walLagThresholdBytes,
		Logger:               logger.GetLogger(),
	})
}

func createStreamerSlot(t *testing.T, fixture *PhysicalDBFixture) {
	t.Helper()

	adminConn := OpenAdminConn(t, fixture)

	require.NoError(t, fixture.DB.PostgresqlPhysical.VerifyWalSlot(
		t.Context(), logger.GetLogger(), encryption.GetFieldEncryptor(),
	))

	t.Cleanup(func() {
		DropReplicationSlotExternally(t, adminConn, fixture.DB.PostgresqlPhysical.ReplicationSlotName)
	})
}

// A committed segment below the source's position gives the tick an archive
// progress point, which the unarchived WAL is measured from.
func seedCommittedSegmentBelowCurrentPosition(t *testing.T, fixture *PhysicalDBFixture) {
	t.Helper()

	currentLSN, err := readCurrentWalLSN(t.Context(), OpenAdminConn(t, fixture))
	require.NoError(t, err)

	segmentSizeBytes := walSegmentSizeBytes(fixture.DB.PostgresqlPhysical)
	segmentEndLSN := walmath.LSN(uint64(currentLSN) / uint64(segmentSizeBytes) * uint64(segmentSizeBytes))
	segmentStartLSN := segmentEndLSN - walmath.LSN(segmentSizeBytes)

	physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		fixture.DB.ID, fixture.Storage.ID, 1, "seeded-segment", segmentStartLSN, segmentEndLSN,
	))
}

func runLagMonitorTicks(
	t *testing.T,
	supervisor *WalStreamSupervisor,
	monitorState *lagMonitorState,
	firstTickAt time.Time,
	tickCount int,
	beforeEachTick func(),
) {
	t.Helper()

	for tickIndex := range tickCount {
		if beforeEachTick != nil {
			beforeEachTick()
		}

		supervisor.runLagMonitorTick(
			t.Context(),
			logger.GetLogger(),
			monitorState,
			firstTickAt.Add(time.Duration(tickIndex)*wal_health.SampleInterval),
		)
	}
}

func getRecordedSamples(t *testing.T, fixture *PhysicalDBFixture) []*physical_models.PhysicalWalStreamSample {
	t.Helper()

	samples, err := physical_repositories.GetWalStreamSampleRepository().
		FindSinceByDatabase(fixture.DB.ID, time.Time{})
	require.NoError(t, err)

	return samples
}

func Test_RunLagMonitorTick_WhenSourceHealthy_RecordsReachableSamplesWithSlotAndQueueValues(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)
	seedCommittedSegmentBelowCurrentPosition(t, fixture)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.DB.PostgresqlPhysical, 0)

	runLagMonitorTicks(t, supervisor, supervisor.newLagMonitorState(logger.GetLogger()), lagMonitorTickStartedAt, 4,
		func() {
			_, err := GenerateWalActivity(t.Context(), adminConn, 4096)
			require.NoError(t, err)
		})

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 4)

	for index, sample := range samples {
		assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateReachable, sample.SourceState)
		assert.Equal(
			t,
			lagMonitorTickStartedAt.Add(time.Duration(index)*wal_health.SampleInterval),
			sample.SampledAt.UTC(),
		)
		assert.NotNil(t, sample.SourceCurrentLSN)
		assert.NotNil(t, sample.SlotWalStatus)
		assert.NotNil(t, sample.SlotLagBytes)
		assert.NotNil(t, sample.UnarchivedWalBytes)
		assert.Positive(t, sample.LocalQueuePauseBytes)

		if index > 0 {
			require.NotNil(t, sample.SourceWrittenBytes)
			assert.Positive(t, *sample.SourceWrittenBytes)
		}
	}

	assert.Nil(t, samples[0].SourceWrittenBytes)
}

func Test_RunLagMonitorTick_WhenSourceUnreachableAcrossSessions_KeepsOneUnreachableSinceTime(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	unreachableSourceDB := *fixture.DB.PostgresqlPhysical
	unreachableSourceDB.Host = "127.0.0.1"
	unreachableSourceDB.Port = 1

	firstSession := newLagMonitorTickSupervisor(t, fixture, &unreachableSourceDB, 0)
	runLagMonitorTicks(t, firstSession, firstSession.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt, 3, nil)

	secondSessionStartedAt := lagMonitorTickStartedAt.Add(5 * time.Minute)
	secondSession := newLagMonitorTickSupervisor(t, fixture, &unreachableSourceDB, 0)
	runLagMonitorTicks(t, secondSession, secondSession.newLagMonitorState(logger.GetLogger()),
		secondSessionStartedAt, 1, nil)

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 4)

	for _, sample := range samples {
		assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateUnreachable, sample.SourceState)
		require.NotNil(t, sample.SourceUnreachableSince)
		assert.Equal(t, lagMonitorTickStartedAt, sample.SourceUnreachableSince.UTC(),
			"the time a source has been unreachable survives the session restart its outage causes")
		assert.Nil(t, sample.SlotWalStatus)
		assert.Nil(t, sample.SlotLagBytes)
		assert.Nil(t, sample.SlotSafeWalSizeBytes)
		assert.Nil(t, sample.UnarchivedWalBytes)
		assert.Nil(t, sample.SourceWrittenBytes)
	}
}

func Test_RunLagMonitorTick_WhenSlotHeldExtended_RecordsTimeClassifierStartedItsHold(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)
	setMaxWalSizeForTest(t, adminConn, "32MB")

	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	require.Eventually(t, func() bool {
		ForceReplicationLag(t, adminConn, 16*1024*1024)

		slotState, err := InspectSlot(t.Context(), adminConn, slotName)

		return err == nil && slotState != nil && slotState.WalStatus == slotWalStatusExtendedForTest
	}, 90*time.Second, time.Second, "a slot lagging past max_wal_size is reported as extended")

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.DB.PostgresqlPhysical, 0)
	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		2,
		nil,
	)

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 2)

	for _, sample := range samples {
		require.NotNil(t, sample.SlotExtraWalSince)
		assert.Equal(t, lagMonitorTickStartedAt, sample.SlotExtraWalSince.UTC())
	}
}

const slotWalStatusExtendedForTest = "extended"

// max_wal_size is reloadable, so the test restores it instead of leaving the
// shared source checkpointing every two segments.
func setMaxWalSizeForTest(t *testing.T, adminConn *pgx.Conn, size string) {
	t.Helper()

	_, err := adminConn.Exec(t.Context(), "ALTER SYSTEM SET max_wal_size = '"+size+"'")
	require.NoError(t, err)

	_, err = adminConn.Exec(t.Context(), "SELECT pg_reload_conf()")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(), "ALTER SYSTEM RESET max_wal_size")
		_, _ = adminConn.Exec(context.Background(), "SELECT pg_reload_conf()")
	})
}

func Test_RunLagMonitorTick_WhenTickRebuildsSlot_RecordsSlotAsProbedBeforeRebuild(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)

	const walLagThresholdBytes = 64 * 1024

	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	ForceReplicationLag(t, adminConn, 4*walLagThresholdBytes)
	WaitUntilSlotLag(t, adminConn, slotName, 2*walLagThresholdBytes, 30*time.Second)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.DB.PostgresqlPhysical, walLagThresholdBytes)
	supervisor.isReceiverRunning.Store(true)

	sampleCountAtRebuild := -1
	supervisor.spec.OnSlotRebuilt = func(context.Context, string) error {
		sampleCountAtRebuild = len(getRecordedSamples(t, fixture))

		return nil
	}

	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		1,
		nil,
	)

	require.Equal(t, 1, sampleCountAtRebuild, "the tick records its sample before it rebuilds the slot")

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 1)
	require.NotNil(t, samples[0].SlotLagBytes)
	assert.Greater(t, *samples[0].SlotLagBytes, int64(walLagThresholdBytes),
		"the sample shows the slot that caused the rebuild")
}

func Test_RunLagMonitorTick_OnFirstSampleOfSession_ClearsStreamerFailingTime(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	streamerRepository := physical_repositories.GetWalStreamerRepository()
	t.Cleanup(func() {
		_ = streamerRepository.DeleteByDatabaseID(fixture.DB.ID)
	})

	require.NoError(t, streamerRepository.Claim(fixture.DB.ID))
	require.NoError(t, streamerRepository.MarkFailed(fixture.DB.ID))

	failedStreamer, err := streamerRepository.FindByDatabaseID(fixture.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, failedStreamer.FailingSince)

	createStreamerSlot(t, fixture)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.DB.PostgresqlPhysical, 0)
	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		1,
		nil,
	)

	recoveredStreamer, err := streamerRepository.FindByDatabaseID(fixture.DB.ID)
	require.NoError(t, err)
	assert.Nil(t, recoveredStreamer.FailingSince)
}
