package backups_config_physical

import "github.com/google/uuid"

type BackupConfigStorageChangeListener interface {
	OnBeforeBackupsStorageChange(dbID uuid.UUID) error
}

// BackupConfigChangeListener is notified when a config save crosses a
// transition that must stand work down: backups disabled, or BackupType demoted
// away from WAL_STREAM. The scheduler package implements it to cancel in-flight
// backups and delete the streamer row.
type BackupConfigChangeListener interface {
	OnBackupConfigChanged(oldConfig, newConfig *PhysicalBackupConfig)
}
