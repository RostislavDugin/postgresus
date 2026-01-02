package backups

import (
	"context"

	usecases_common "databasus-backend/internal/features/backups/backups/common"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"

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
