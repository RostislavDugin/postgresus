package backuping_physical

import (
	"context"

	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/features/notifiers"
)

type NotificationSender interface {
	SendNotification(notifier *notifiers.Notifier, title, message string)
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
