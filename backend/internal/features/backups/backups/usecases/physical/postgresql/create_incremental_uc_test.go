package usecases_physical_postgresql_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/util/testing/containers"
)

// Test_CreateIncremental_WhenSummarizerOff_ChainBrokenNoArtifact is the live-path
// regression guard for the SUMMARIZER_OFF fix: against the summarize_wal=off
// cluster, the incremental pre-check must turn the INCR into CHAIN_BROKEN with
// reason SUMMARIZER_OFF *before* any pg_basebackup or upload — so the next tick
// re-anchors on a FULL instead of looping transient ERRORs (no artifact is left
// in storage, FileName stays nil).
func Test_CreateIncremental_WhenSummarizerOff_ChainBrokenNoArtifact(t *testing.T) {
	source := containers.StartPhysicalPostgres(t, "postgres:17", containers.WithoutSummarizer())
	fixture := postgresql_executor.SetupPhysicalDBForBackupNoSummary(t, source.Host, source.Port)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	incrID := postgresql_executor.BuildAndClaimIncremental(t, fixture, nil)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), incrID, false)

	summarizerOff := physical_enums.PhysicalBackupErrorSummarizerOff
	postgresql_executor.WaitForBackupStatus(t, incrID, physical_enums.PhysicalBackupTypeIncremental,
		physical_enums.PhysicalBackupStatusChainBroken, &summarizerOff, 2*time.Minute)

	incrRow, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrID)
	require.NoError(t, err)
	require.NotNil(t, incrRow)
	assert.Nil(t, incrRow.FileName,
		"pre-check must bail before claiming a file name, so no artifact is uploaded")
}

// Test_CreateIncremental_WhenWalWrittenSinceLastCheckpoint_Completes pins issue 756.
// WAL summaries close only at checkpoint records, so on a cluster that keeps writing
// the newest published summary ends far behind the WAL tip. The incremental must run
// anyway: pg_basebackup forces its own checkpoint, which closes the summary it needs.
func Test_CreateIncremental_WhenWalWrittenSinceLastCheckpoint_Completes(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	fullRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotNil(t, fullRow.StopLSN, "FULL must have stop_lsn for the INCR to anchor on")

	conn := postgresql_executor.OpenAdminConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	// Several segments' worth, and deliberately no manual CHECKPOINT, pg_switch_wal
	// or wait for a summary: the newest published summary must end well behind the
	// WAL tip when the pre-check runs. pg_basebackup forces its own checkpoint, and
	// that is what closes the summary file the incremental actually needs.
	_, err = postgresql_executor.GenerateWalActivity(ctx, conn, 64*1024*1024)
	require.NoError(t, err)

	incrID := postgresql_executor.BuildAndClaimIncremental(t, fixture, nil)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), incrID, false)
	postgresql_executor.WaitForBackupStatus(t, incrID, physical_enums.PhysicalBackupTypeIncremental,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	incrRow, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrID)
	require.NoError(t, err)
	require.NotNil(t, incrRow.FileName, "a healthy incremental must upload an artifact")
	require.NotNil(t, incrRow.StopLSN, "a completed incremental must record stop_lsn")
}

func Test_CreateIncremental_WhenSummarizerBehindOnIdleDB_StillCompletes(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	incrID := postgresql_executor.BuildAndClaimIncremental(t, fixture, nil)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), incrID, false)
	postgresql_executor.WaitForBackupStatus(t, incrID, physical_enums.PhysicalBackupTypeIncremental,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	incrRow, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrID)
	require.NoError(t, err)
	require.NotNil(t, incrRow.FileName, "an incremental on an idle DB must still upload an artifact")
}
