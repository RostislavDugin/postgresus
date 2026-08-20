package backups_config_logical

import (
	"context"

	"github.com/google/uuid"
)

type BackupConfigStorageChangeListener interface {
	OnBeforeBackupsStorageChange(ctx context.Context, dbID uuid.UUID) error
}
