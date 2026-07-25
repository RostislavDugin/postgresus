package usecases_physical_postgresql

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

func Test_WalStream_FullIncrementalAndWalStream_StreamerArchivesSegments(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	store := newMockWalStorage()

	stop := StartWalStreamerForTest(t, fixture, store, t.TempDir()).Stop
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

	t.Cleanup(StartWalStreamerForTest(t, fixtureA, storeA, t.TempDir()).Stop)
	t.Cleanup(StartWalStreamerForTest(t, fixtureB, storeB, t.TempDir()).Stop)

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

	t.Cleanup(StartWalStreamerForTest(t, fixture, store, t.TempDir()).Stop)

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
	t.Cleanup(StartWalStreamerForTest(t, fixture, newMockWalStorage(), t.TempDir()).Stop)

	deadline := time.Now().UTC().Add(60 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if SlotLagBytes(t, adminConn, slotName) < lagTarget {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("slot lag did not drain below %d within 60s after streaming started", lagTarget)
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

	expectedStartLSN := walmath.LSN((uint64(2) << 32) + 3*uint64(customSegSize))

	row := findWalSegment(t, fixture.DB.ID, 1, expectedStartLSN)
	require.NotNil(t, row, "segment LSN must be derived from the DB's segsize, not the walmath global")
	require.Equal(t, expectedStartLSN, row.StartLSN)
	require.Equal(t, expectedStartLSN+walmath.LSN(customSegSize), row.EndLSN)
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

func Test_WalStream_ResumePointBelowSlotRestartLsn_RealignsAndKeepsStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("streamer integration test runs pg_receivewal; skipped in -short")
	}

	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	watchDirRoot := t.TempDir()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	// A storage that refuses every save leaves finalized segments sitting in the
	// watch dir — the local queue this issue is about. It has to keep failing
	// across the restart, or startup recovery would drain the queue and there
	// would be nothing left to drag the resume point down.
	store := newMockWalStorage()
	store.startFailingSaves()

	firstRun := StartWalStreamerForTest(t, fixture, store, watchDirRoot)

	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	waitForQueuedSegments(t, firstRun.WatchDir, 1, 60*time.Second)
	firstRun.Stop()

	queuedBeforeRebuild := queuedSegmentNames(t, firstRun.WatchDir)
	require.NotEmpty(t, queuedBeforeRebuild)

	// Model the incident: the slot is rebuilt while the queue still holds
	// pre-rebuild segments, so the recreated slot reserves from a position far
	// above them and pg_receivewal would otherwise resume below it.
	DropReplicationSlotExternally(t, adminConn, slotName)

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	resumeSegmentNo, isResumeSegmentFound := GetResumeSegmentNo(firstRun.WatchDir, segmentSizeBytes)
	require.True(t, isResumeSegmentFound, "the local queue must hold at least one complete segment")

	burnWalPastSegment(t, ctx, adminConn, resumeSegmentNo+1)

	// pg_create_physical_replication_slot(..., true) reserves from the last
	// checkpoint's redo point, not from the current insert position, so without
	// this the recreated slot can still land below the queue.
	_, err := adminConn.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	require.NoError(t, fixture.DB.PostgresqlPhysical.VerifyWalSlot(
		ctx, logger.GetLogger(), encryption.GetFieldEncryptor(),
	))

	requireQueueBelowSlot(t, ctx, adminConn, slotName, resumeSegmentNo)

	secondRun := StartWalStreamerForTest(t, fixture, store, watchDirRoot)
	t.Cleanup(secondRun.Stop)

	pendingUploadDir := filepath.Join(secondRun.WatchDir, pendingUploadDirName)

	for _, staleSegment := range queuedBeforeRebuild {
		require.Eventually(t, func() bool {
			_, err := os.Stat(filepath.Join(pendingUploadDir, staleSegment))

			return err == nil
		}, 60*time.Second, 250*time.Millisecond,
			"segment below the new restart_lsn must leave pg_receivewal's resume path: %s", staleSegment)

		require.NoFileExists(t, filepath.Join(secondRun.WatchDir, staleSegment))
	}

	// Storage recovers: the staged segments are valid WAL of the older chain, so
	// they must still reach storage, and the receiver must keep streaming the new
	// chain rather than crash-looping on the recycled WAL it used to ask for.
	store.stopFailingSaves()

	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	for _, staleSegment := range queuedBeforeRebuild {
		require.Eventually(t, func() bool {
			return store.hasObject(walSegmentObjectName(fixture.DB.ID, 1, staleSegment))
		}, 60*time.Second, 250*time.Millisecond,
			"a segment moved out of the resume path must still reach storage: %s", staleSegment)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)
}

func queuedSegmentNames(t *testing.T, watchDir string) []string {
	t.Helper()

	entries, err := os.ReadDir(watchDir)
	require.NoError(t, err)

	var queuedSegments []string

	for _, entry := range entries {
		if !entry.IsDir() && walmath.IsWalFilename(entry.Name()) {
			queuedSegments = append(queuedSegments, entry.Name())
		}
	}

	return queuedSegments
}

func waitForQueuedSegments(t *testing.T, watchDir string, minCount int, timeout time.Duration) {
	t.Helper()

	require.Eventually(t, func() bool {
		return len(queuedSegmentNames(t, watchDir)) >= minCount
	}, timeout, 250*time.Millisecond, "watch dir never accumulated %d finalized segments", minCount)
}

// pg_switch_wal is a no-op on an already-empty segment, so the cluster only
// moves past the queue if real WAL is written between switches.
func burnWalPastSegment(
	t *testing.T,
	ctx context.Context,
	adminConn *pgx.Conn,
	targetSegmentNo walmath.WalSegmentNo,
) {
	t.Helper()

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	require.Eventually(t, func() bool {
		if _, err := GenerateWalActivity(ctx, adminConn, segmentSizeBytes); err != nil {
			return false
		}

		if _, err := ForceWalRotation(ctx, adminConn); err != nil {
			return false
		}

		var currentLSN walmath.LSN
		if err := adminConn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&currentLSN); err != nil {
			return false
		}

		return segmentNoAtLSN(currentLSN, segmentSizeBytes) > targetSegmentNo
	}, 2*time.Minute, 100*time.Millisecond, "cluster never wrote past segment %d", uint64(targetSegmentNo))
}

// The realign only has work to do when the recreated slot reserves above the
// queue pg_receivewal would otherwise resume from, so assert that rather than
// let the test pass without exercising anything.
func requireQueueBelowSlot(
	t *testing.T,
	ctx context.Context,
	adminConn *pgx.Conn,
	slotName string,
	resumeSegmentNo walmath.WalSegmentNo,
) {
	t.Helper()

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	slotState, err := InspectSlot(ctx, adminConn, slotName)
	require.NoError(t, err)
	require.NotNil(t, slotState)

	require.Less(t, uint64(resumeSegmentNo), uint64(segmentNoAtLSN(slotState.RestartLSN, segmentSizeBytes)),
		"the queue must sit below the recreated slot for this test to mean anything")
}
