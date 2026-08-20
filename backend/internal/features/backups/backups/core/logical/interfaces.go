package backups_core_logical

import (
	"context"

	"github.com/google/uuid"

	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
)

type NotificationSender interface {
	SendNotification(
		ctx context.Context,
		notifier *notifiers.Notifier,
		notification notifier_models.Notification,
	)
}

type CreateBackupUsecase interface {
	Execute(
		ctx context.Context,
		backup *LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		database *databases.Database,
		storage *storages.Storage,
		backupProgressListener func(completedMBs float64),
	) (*BackupMetadata, error)
}

type BackupRemoveListener interface {
	OnBeforeBackupRemove(backup *LogicalBackup) error
}

type BackupCompletionListener interface {
	OnBackupCompleted(backupID uuid.UUID)
}
