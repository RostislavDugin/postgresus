package backups_config_logical

import "github.com/google/uuid"

type BackupConfigStorageChangeListener interface {
	OnBeforeBackupsStorageChange(dbID uuid.UUID) error
}
