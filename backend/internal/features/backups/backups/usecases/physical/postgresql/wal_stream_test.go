package usecases_physical_postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

// startStreamerForTest runs a WalStreamSupervisor against the fixture's source PG
// in a goroutine and returns a stop func that cancels and waits for full drain —
// ensuring pg_receivewal has released the slot before the DB (and its slot) are
// torn down by later cleanups.
func startStreamerForTest(t *testing.T, fixture *PhysicalDBFixture, store storages.StorageFileSaver) func() {
	t.Helper()

	spec := WalStreamSpec{
		DatabaseID:     fixture.DB.ID,
		SourceDB:       fixture.DB.PostgresqlPhysical,
		StorageID:      fixture.Storage.ID,
		Storage:        store,
		Encryption:     backups_core_enums.BackupEncryptionNone,
		FieldEncryptor: encryption.GetFieldEncryptor(),
		WalSegmentRepo: physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:    physical_repositories.GetWalHistoryRepository(),
		WatchDirRoot:   t.TempDir(),
		Logger:         logger.GetLogger(),
	}

	supervisor := NewWalStreamSupervisor(spec)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = supervisor.Run(ctx)
	}()

	return func() {
		cancel()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Log("streamer did not stop within timeout")
		}
	}
}

func Test_WalStream_FullIncrementalAndWalStream_StreamerArchivesSegments(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	store := newMockWalStorage()

	stop := startStreamerForTest(t, fixture, store)
	t.Cleanup(stop)

	adminConn := OpenAdminConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// Force three segment rotations so pg_receivewal finalizes segments the
	// uploader can archive.
	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		fixture.DB.ID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	var committed int
	for _, seg := range segments {
		if seg.FileName == nil {
			continue
		}

		committed++

		require.True(t, store.hasObject(*seg.FileName), "archived segment must exist in storage: %s", *seg.FileName)
		require.True(t, store.hasObject(*seg.FileName+metadataSuffix), "segment sidecar must exist in storage")
	}

	require.GreaterOrEqual(t, committed, 1, "at least one rotated segment must be archived")
}

// assertDbSegmentsArchivedOnlyIn asserts every committed segment of databaseID
// is present in ownStore and absent from otherStore — per-DB streamer isolation.
func assertDbSegmentsArchivedOnlyIn(
	t *testing.T,
	databaseID uuid.UUID,
	ownStore, otherStore *mockWalStorage,
) {
	t.Helper()

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	committed := 0
	for _, seg := range segments {
		if seg.FileName == nil {
			continue
		}

		committed++

		require.True(t, ownStore.hasObject(*seg.FileName), "own store must hold %s", *seg.FileName)
		require.False(t, otherStore.hasObject(*seg.FileName), "other DB's store must not hold %s", *seg.FileName)
	}

	require.GreaterOrEqual(t, committed, 1, "database %s must archive at least one segment", databaseID)
}

// committedSegmentsInOrder returns the database's committed (file_name NOT NULL)
// WAL segments ordered by start_lsn (FindByChainSpan already sorts ascending).
func committedSegmentsInOrder(t *testing.T, databaseID uuid.UUID) []*physical_models.PhysicalWalSegment {
	t.Helper()

	all, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	committed := make([]*physical_models.PhysicalWalSegment, 0, len(all))
	for _, seg := range all {
		if seg.FileName != nil {
			committed = append(committed, seg)
		}
	}

	return committed
}

func Test_WalStream_MultipleDbs_EachArchivesSegmentsIndependently(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixtureA := SetupPhysicalDBForBackup(t)
	fixtureB := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixtureA.DB.ID)
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixtureB.DB.ID)
	})

	storeA := newMockWalStorage()
	storeB := newMockWalStorage()

	t.Cleanup(startStreamerForTest(t, fixtureA, storeA))
	t.Cleanup(startStreamerForTest(t, fixtureB, storeB))

	connA := OpenAdminConn(t, fixtureA)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// One shared physical cluster: rotating WAL feeds both DBs' independent slots.
	for range 4 {
		_, err := ForceWalRotation(ctx, connA)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixtureA.DB.ID, 1, 90*time.Second)
	WaitForCommittedWalSegmentCount(t, fixtureB.DB.ID, 1, 90*time.Second)

	assertDbSegmentsArchivedOnlyIn(t, fixtureA.DB.ID, storeA, storeB)
	assertDbSegmentsArchivedOnlyIn(t, fixtureB.DB.ID, storeB, storeA)
}

func Test_WalStream_MissingSegmentInStreamedChain_SurfacesAsGapChainStaysExtendable(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	// Anchor a COMPLETED FULL at LSN 0 so every streamed segment falls in its span.
	MarkFullCompleted(t, fixture.BackupID, 1, walmath.LSN(0), walmath.LSN(0))

	store := newMockWalStorage()
	adminConn := OpenAdminConn(t, fixture)

	t.Cleanup(startStreamerForTest(t, fixture, store))

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	for range 5 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 3, 90*time.Second)

	// A real streamed chain is contiguous, so no gap yet.
	gapsBefore, err := chain_view.GetChainViewService().FindWalGapsInChain(fixture.BackupID)
	require.NoError(t, err)
	require.Empty(t, gapsBefore, "a contiguous streamed chain has no gaps")

	// Drop a middle committed segment to model a lost / retention-trimmed segment.
	// The gap is derived from the surviving rows' LSN math — no marker row exists.
	committed := committedSegmentsInOrder(t, fixture.DB.ID)
	require.GreaterOrEqual(t, len(committed), 3)
	removed := committed[1]
	require.NoError(t, physical_repositories.GetWalSegmentRepository().DeleteByID(removed.ID))

	gaps := WaitForWalGap(t, fixture.BackupID, 30*time.Second)
	require.Len(t, gaps, 1, "exactly the removed segment's range must surface as a gap")
	require.Equal(t, removed.StartLSN, gaps[0].Start)
	require.Equal(t, removed.EndLSN, gaps[0].End)

	// The chain remains extendable despite the internal gap (lossy chain).
	chain := WaitForExtendableChain(t, fixture.DB.ID, 10*time.Second)
	require.Equal(t, fixture.BackupID, chain.RootFull.ID)
}

func Test_WalStream_SlotLagGrowsWithoutConsumer_DrainsOnceStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// Create the persistent slot with no consumer attached, then burn WAL so the
	// slot's restart_lsn falls behind — the signal the lag monitor reads.
	require.NoError(
		t,
		fixture.DB.PostgresqlPhysical.VerifyWalSlot(ctx, logger.GetLogger(), encryption.GetFieldEncryptor()),
	)

	const lagTarget = 8 * 1024 * 1024
	ForceReplicationLag(t, adminConn, lagTarget)
	WaitUntilSlotLag(t, adminConn, slotName, lagTarget, 30*time.Second)

	// Once our streamer attaches, it consumes the backlog and the lag drains.
	t.Cleanup(startStreamerForTest(t, fixture, newMockWalStorage()))

	deadline := time.Now().UTC().Add(60 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if SlotLagBytes(t, adminConn, slotName) < lagTarget {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("slot lag did not drain below %d within 60s after streaming started", lagTarget)
}

func Test_WalStream_BackpressureWatermarks_ScaleWithWalSegmentSize(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	customSegSize := int64(512 * 1024 * 1024)
	fixture.DB.PostgresqlPhysical.WalSegmentSizeBytes = &customSegSize

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:   fixture.DB.ID,
		SourceDB:     fixture.DB.PostgresqlPhysical,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})

	require.Equal(t, 4*customSegSize, supervisor.highWatermarkBytes)
	require.Equal(t, 4*customSegSize/5, supervisor.lowWatermarkBytes)
}

func Test_WalStream_RebuildAttemptCap_StopsFourthAttemptInHour(t *testing.T) {
	supervisor := &WalStreamSupervisor{}

	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.False(t, supervisor.recordRebuildAttemptWithinCap())
}

func Test_WalStream_CustomWalSegmentSize_LsnMathCorrect(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	const customSegSize = int64(64 * 1024 * 1024) // 64 MB segments

	store := newMockWalStorage()
	uploader := NewWalUploader(WalUploadDeps{
		DatabaseID:          fixture.DB.ID,
		StorageID:           fixture.Storage.ID,
		Storage:             store,
		Encryption:          backups_core_enums.BackupEncryptionNone,
		FieldEncryptor:      encryption.GetFieldEncryptor(),
		WalSegmentRepo:      physical_repositories.GetWalSegmentRepository(),
		WalSegmentSizeBytes: customSegSize,
		Logger:              logger.GetLogger(),
	})

	// At 64 MB segments there are 64 segments per 4 GiB logid. Segment with
	// logid=2, segLow=3 starts at (2<<32) + 3*64MB.
	dir := t.TempDir()
	name := "000000010000000200000003"
	require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir, name), name))

	wantStart := walmath.LSN((uint64(2) << 32) + 3*uint64(customSegSize))

	row := findWalSegment(t, fixture.DB.ID, 1, wantStart)
	require.NotNil(t, row, "segment LSN must be derived from the DB's segsize, not the walmath global")
	require.Equal(t, wantStart, row.StartLSN)
	require.Equal(t, wantStart+walmath.LSN(customSegSize), row.EndLSN)
}

func Test_WalUpload_ConcurrentClaimSameSegment_OnlyWinnerInserts(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	repo := physical_repositories.GetWalSegmentRepository()
	startLSN := walmath.LSN(40 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	const racers = 6

	type claimOutcome struct {
		inserted bool
		err      error
	}

	results := make(chan claimOutcome, racers)
	start := make(chan struct{})

	for range racers {
		go func() {
			<-start

			// Don't call require.* off the test goroutine — collect and assert below.
			inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
				DatabaseID:  fixture.DB.ID,
				StorageID:   fixture.Storage.ID,
				TimelineID:  1,
				WalFilename: walName(1, 40),
				StartLSN:    startLSN,
				EndLSN:      endLSN,
				Encryption:  backups_core_enums.BackupEncryptionNone,
			})
			results <- claimOutcome{inserted: inserted, err: err}
		}()
	}

	close(start)

	winners := 0
	for range racers {
		outcome := <-results
		require.NoError(t, outcome.err)

		if outcome.inserted {
			winners++
		}
	}

	require.Equal(t, 1, winners, "exactly one concurrent claim may win the (db, tl, start_lsn) slot")
}

func Test_Cleaner_AbandonedNullClaim_OlderThanGrace_DeletedYoungerSurvives(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	repo := physical_repositories.GetWalSegmentRepository()

	oldClaim := &physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: walName(1, 50),
		StartLSN:    walmath.LSN(50 * uint64(testWalSegmentSize)),
		EndLSN:      walmath.LSN(51 * uint64(testWalSegmentSize)),
		Encryption:  backups_core_enums.BackupEncryptionNone,
		ClaimedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}
	inserted, err := repo.ClaimInsert(oldClaim)
	require.NoError(t, err)
	require.True(t, inserted)

	youngClaim := &physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: walName(1, 51),
		StartLSN:    walmath.LSN(51 * uint64(testWalSegmentSize)),
		EndLSN:      walmath.LSN(52 * uint64(testWalSegmentSize)),
		Encryption:  backups_core_enums.BackupEncryptionNone,
		ClaimedAt:   time.Now().UTC().Add(-30 * time.Minute),
	}
	inserted, err = repo.ClaimInsert(youngClaim)
	require.NoError(t, err)
	require.True(t, inserted)

	deleted, err := repo.DeleteAbandonedClaims(fixture.DB.ID, time.Now().UTC().Add(-1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted, "only the over-grace NULL claim must be reaped")

	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, oldClaim.StartLSN), "aged claim must be gone")
	require.NotNil(t, findWalSegment(t, fixture.DB.ID, 1, youngClaim.StartLSN), "within-grace claim must survive")
}

func Test_StallTracker_WhenFirstSample_DoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute),
		"the first sample only arms the clock; it can never be a stall")
}

func Test_StallTracker_WhenRestartLsnAdvances_ReArmsAndDoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.False(t, tracker.observe(walmath.LSN(200), base.Add(2*time.Minute), time.Minute),
		"a changed restart_lsn means progress — the advance clock must reset")
}

func Test_StallTracker_WhenFrozenWithinTimeout_DoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.False(t, tracker.observe(walmath.LSN(100), base.Add(30*time.Second), time.Minute),
		"a frozen restart_lsn within the stall timeout is not yet a stall")
}

func Test_StallTracker_WhenFrozenPastTimeout_RestartsThenReArms(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.True(t, tracker.observe(walmath.LSN(100), base.Add(90*time.Second), time.Minute),
		"a frozen restart_lsn past the stall timeout must trigger a restart")

	require.False(t, tracker.observe(walmath.LSN(100), base.Add(2*time.Minute), time.Minute),
		"after firing, the clock re-arms so we restart at most once per window")
	require.True(t, tracker.observe(walmath.LSN(100), base.Add(4*time.Minute), time.Minute),
		"a sustained stall fires again only after another full timeout window")
}
