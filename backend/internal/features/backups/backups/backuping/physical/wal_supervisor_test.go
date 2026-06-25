package backuping_physical

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/storage"
)

// seedStreamerRow inserts a physical_wal_streamers row with an explicit
// last_heartbeat_at and status, for ClaimIfClaimable tests. database_id must FK
// to a real databases row.
func seedStreamerRow(
	t *testing.T,
	databaseID uuid.UUID,
	heartbeatAt time.Time,
	status physical_enums.PhysicalWalStreamerStatus,
) {
	t.Helper()

	require.NoError(t, storage.GetDb().Exec(`
		INSERT INTO physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
		VALUES (?, NOW(), ?, ?)
		ON CONFLICT (database_id) DO UPDATE
		  SET last_heartbeat_at = EXCLUDED.last_heartbeat_at, status = EXCLUDED.status
	`, databaseID, heartbeatAt, status).Error)

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(databaseID)
	})
}

func Test_ClaimIfClaimable_WhenNoRowExists_InsertsAndClaims(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(prereqs.DB.ID)
	})

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a missing row must be insert-claimed")

	row, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusRunning, row.Status)
}

func Test_ClaimIfClaimable_WhenFreshRunningRow_NotClaimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(t, prereqs.DB.ID, time.Now().UTC(), physical_enums.PhysicalWalStreamerStatusRunning)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.False(t, claimed, "a fresh RUNNING streamer owned by a live process must not be reclaimed")
}

func Test_ClaimIfClaimable_WhenHeartbeatStale_Claimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-5*time.Minute),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a streamer whose heartbeat aged past staleness must be reclaimable")
}

func Test_ClaimIfClaimable_WhenFailedRow_Claimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(t, prereqs.DB.ID, time.Now().UTC(), physical_enums.PhysicalWalStreamerStatusFailed)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a FAILED streamer must be reclaimable even with a fresh heartbeat")
}

func Test_ClaimIfClaimable_WhenConcurrentClaims_ExactlyOneWins(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-5*time.Minute),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	const racers = 8

	type claimOutcome struct {
		claimed bool
		err     error
	}

	results := make(chan claimOutcome, racers)
	start := make(chan struct{})

	for range racers {
		go func() {
			<-start

			// Don't call require.* off the test goroutine — collect and assert below.
			claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
			results <- claimOutcome{claimed: claimed, err: err}
		}()
	}

	close(start)

	wins := 0
	for range racers {
		outcome := <-results
		require.NoError(t, outcome.err)

		if outcome.claimed {
			wins++
		}
	}

	require.Equal(t, 1, wins, "exactly one concurrent claimant may win the stale streamer")
}

func Test_PhysicalWalStreamSupervisorRun_WhenCalledTwice_Panics(t *testing.T) {
	supervisor := CreateTestWalStreamSupervisor()

	ctx := t.Context()

	go supervisor.Run(ctx)

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if supervisor.IsRunning() {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	require.True(t, supervisor.IsRunning(), "supervisor must report running before the double-run check")

	require.Panics(t, func() { supervisor.Run(ctx) },
		"a second Run() on the same instance must panic")
}

func Test_StopStreamer_WhenOwnedStreamerStops_MarksRowFailed(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	supervisor := CreateTestWalStreamSupervisor()
	require.NoError(t, physical_repositories.GetWalStreamerRepository().Claim(prereqs.DB.ID))

	done := make(chan struct{})
	close(done)

	supervisor.running[prereqs.DB.ID] = &runningStreamer{
		cancel: func() {},
		done:   done,
	}

	supervisor.stopStreamer(prereqs.DB.ID, false)

	streamer, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusFailed, streamer.Status)
}

func Test_RecoverStreamersOnStartup_WhenRunningRowStale_MarksFailed(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-2*streamerHeartbeatStaleness),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	supervisor := CreateTestWalStreamSupervisor()

	require.NoError(t, supervisor.recoverStreamersOnStartup())

	streamer, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusFailed, streamer.Status)
}

func Test_RemoveWatchDirIfRequested_WhenCleanupRequested_RemovesQueue(t *testing.T) {
	supervisor := CreateTestWalStreamSupervisor()
	watchDir := filepath.Join(t.TempDir(), "wal-queue")
	require.NoError(t, os.MkdirAll(watchDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "segment"), []byte("wal"), 0o600))

	streamer := &runningStreamer{watchDir: watchDir}
	streamer.shouldRemoveWatchDir.Store(true)

	supervisor.removeWatchDirIfRequested(supervisor.logger, streamer)

	require.NoDirExists(t, watchDir)
}
