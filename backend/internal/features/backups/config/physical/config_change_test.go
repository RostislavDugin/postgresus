package backups_config_physical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
)

type recordingConfigChangeListener struct {
	callCount int
}

func (r *recordingConfigChangeListener) OnBackupConfigChanged(_, _ *PhysicalBackupConfig) {
	r.callCount++
}

func configWithType(
	databaseID uuid.UUID,
	enabled bool,
	backupType postgresql_physical.BackupType,
) *PhysicalBackupConfig {
	return &PhysicalBackupConfig{
		DatabaseID:         databaseID,
		IsBackupsEnabled:   enabled,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{BackupType: backupType},
	}
}

func Test_NotifyConfigChangeIfNeeded_FiresOnDisableAndDemoteOnly(t *testing.T) {
	recorder := &recordingConfigChangeListener{}
	service := &BackupConfigService{configChangeListener: recorder}
	databaseID := uuid.New()

	walStreamEnabled := configWithType(databaseID, true, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	walStreamDisabled := configWithType(databaseID, false, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	fullIncrEnabled := configWithType(databaseID, true, postgresql_physical.BackupTypeFullAndIncremental)

	// No-op save (no transition) must not fire.
	service.notifyConfigChangeIfNeeded(walStreamEnabled, walStreamEnabled)
	assert.Equal(t, 0, recorder.callCount)

	// Disable fires.
	service.notifyConfigChangeIfNeeded(walStreamEnabled, walStreamDisabled)
	assert.Equal(t, 1, recorder.callCount)

	// Demote from WAL_STREAM (still enabled) fires.
	service.notifyConfigChangeIfNeeded(walStreamEnabled, fullIncrEnabled)
	assert.Equal(t, 2, recorder.callCount)

	// Enabling-stays-enabled with same type does not fire.
	service.notifyConfigChangeIfNeeded(fullIncrEnabled, fullIncrEnabled)
	assert.Equal(t, 2, recorder.callCount)
}
