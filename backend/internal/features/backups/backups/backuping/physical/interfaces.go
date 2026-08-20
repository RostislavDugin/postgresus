package backuping_physical

import (
	"context"

	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
)

type NotificationSender interface {
	SendNotification(ctx context.Context, notifier *notifiers.Notifier, notification notifier_models.Notification)
}

type FullBackupExecutor interface {
	Execute(
		ctx context.Context,
		spec postgresql_executor.FullBackupSpec,
	) (postgresql_executor.PhysicalBackupResult, error)
}

type IncrementalBackupExecutor interface {
	Execute(
		ctx context.Context,
		spec postgresql_executor.IncrementalBackupSpec,
	) (postgresql_executor.PhysicalBackupResult, error)
}
