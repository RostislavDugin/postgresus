package usecases

import (
	"errors"
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/restores/models"
	usecases_postgresql "postgresus-backend/internal/features/restores/usecases/postgresql"
	"postgresus-backend/internal/features/storages"
)

type RestoreBackupUsecase struct {
	restorePostgresqlBackupUsecase *usecases_postgresql.RestorePostgresqlBackupUsecase
}

func (uc *RestoreBackupUsecase) Execute(
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	database *databases.Database,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	if database.Type == databases.DatabaseTypePostgres {
		return uc.restorePostgresqlBackupUsecase.Execute(
			database,
			backupConfig,
			restore,
			backup,
			storage,
		)
	}

	return errors.New("database type not supported")
}
