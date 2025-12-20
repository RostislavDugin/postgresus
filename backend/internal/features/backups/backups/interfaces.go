package backups

import (
	"context"

	usecases_common "postgresus-backend/internal/features/backups/backups/usecases/common"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"

	"github.com/google/uuid"
)

type NotificationSender interface {
	SendNotification(
		notifier *notifiers.Notifier,
		title string,
		message string,
	)
}

type CreateBackupUsecase interface {
	Execute(
		ctx context.Context,
		backupID uuid.UUID,
		backupConfig *backups_config.BackupConfig,
		database *databases.Database,
		storage *storages.Storage,
		backupProgressListener func(completedMBs float64),
	) (*usecases_common.BackupMetadata, error)
}

type BackupRemoveListener interface {
	OnBeforeBackupRemove(backup *Backup) error
}
