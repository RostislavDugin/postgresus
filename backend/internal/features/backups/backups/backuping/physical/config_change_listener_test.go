package backuping_physical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

func newTestCancellationListener() *PhysicalBackupCancellationListener {
	return &PhysicalBackupCancellationListener{
		NewPhysicalBackupCanceller(
			physical_repositories.GetInFlightBackupRepository(),
			tasks_cancellation.GetTaskCancelManager(),
			logger.GetLogger(),
		),
		physical_repositories.GetWalStreamerRepository(),
		tasks_cancellation.GetTaskCancelManager(),
		logger.GetLogger(),
	}
}

func seedClaimAndStreamer(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	ok, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: databaseID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   uuid.New(),
		})
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, physical_repositories.GetWalStreamerRepository().Claim(databaseID))
}

func configWithBackupType(
	databaseID uuid.UUID,
	enabled bool,
	backupType postgresql_physical.BackupType,
) *backups_config_physical.PhysicalBackupConfig {
	return &backups_config_physical.PhysicalBackupConfig{
		DatabaseID:         databaseID,
		IsBackupsEnabled:   enabled,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{BackupType: backupType},
	}
}

func Test_OnBackupConfigChanged_WhenBackupsDisabled_CancelsInFlightAndDeletesStreamer(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	oldConfig := configWithBackupType(prereqs.DB.ID, true, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	newConfig := configWithBackupType(prereqs.DB.ID, false, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	listener.OnBackupConfigChanged(oldConfig, newConfig)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim, "disabling backups must cancel + release the in-flight claim")

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer, "disabling backups must delete the streamer row")
}

func Test_OnBackupConfigChanged_WhenDemotedFromWalStream_DeletesStreamerKeepsInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	oldConfig := configWithBackupType(prereqs.DB.ID, true, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	newConfig := configWithBackupType(prereqs.DB.ID, true, postgresql_physical.BackupTypeFullAndIncremental)

	listener.OnBackupConfigChanged(oldConfig, newConfig)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.NotNil(t, claim, "demoting BackupType must leave in-flight FULL/INCR running")

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer, "demoting BackupType must delete the streamer row")
}

func Test_OnBeforeDatabaseRemove_CancelsInFlightAndDeletesStreamer(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	require.NoError(t, listener.OnBeforeDatabaseRemove(prereqs.DB.ID))

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim)

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer)
}
