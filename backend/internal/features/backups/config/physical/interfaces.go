package backups_config_physical

import (
	"context"

	"github.com/google/uuid"
)

type BackupConfigStorageChangeListener interface {
	OnBeforeBackupsStorageChange(ctx context.Context, dbID uuid.UUID) error
}

type BackupCancellationListener interface {
	OnBackupsDisabled(ctx context.Context, databaseID uuid.UUID)
	OnWalStreamingDisabled(ctx context.Context, databaseID uuid.UUID)
}
