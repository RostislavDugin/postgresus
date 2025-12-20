package usecases

import (
	"errors"

	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/restores/models"
	usecases_mysql "postgresus-backend/internal/features/restores/usecases/mysql"
	usecases_postgresql "postgresus-backend/internal/features/restores/usecases/postgresql"
	"postgresus-backend/internal/features/storages"
)

type RestoreBackupUsecase struct {
	restorePostgresqlBackupUsecase *usecases_postgresql.RestorePostgresqlBackupUsecase
	restoreMysqlBackupUsecase      *usecases_mysql.RestoreMysqlBackupUsecase
}

func (uc *RestoreBackupUsecase) Execute(
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backup *backups.Backup,
	storage *storages.Storage,
	isExcludeExtensions bool,
) error {
	switch originalDB.Type {
	case databases.DatabaseTypePostgres:
		return uc.restorePostgresqlBackupUsecase.Execute(
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
			isExcludeExtensions,
		)
	case databases.DatabaseTypeMysql:
		return uc.restoreMysqlBackupUsecase.Execute(
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
		)
	default:
		return errors.New("database type not supported")
	}
}
