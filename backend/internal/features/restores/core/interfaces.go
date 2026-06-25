package restores_core

import (
	"context"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
)

type RestoreBackupUsecase interface {
	Execute(
		ctx context.Context,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		restore Restore,
		originalDB *databases.Database,
		restoringToDB *databases.Database,
		backup *backups_core_logical.LogicalBackup,
		storage *storages.Storage,
		options RestoreOptions,
	) error
}
