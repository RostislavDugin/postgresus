package usecases

import (
	"errors"

	"databasus-backend/internal/features/backups/backups"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/restores/models"
	usecases_mariadb "databasus-backend/internal/features/restores/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/restores/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/restores/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/restores/usecases/postgresql"
	"databasus-backend/internal/features/storages"
)

type RestoreBackupUsecase struct {
	restorePostgresqlBackupUsecase *usecases_postgresql.RestorePostgresqlBackupUsecase
	restoreMysqlBackupUsecase      *usecases_mysql.RestoreMysqlBackupUsecase
	restoreMariadbBackupUsecase    *usecases_mariadb.RestoreMariadbBackupUsecase
	restoreMongodbBackupUsecase    *usecases_mongodb.RestoreMongodbBackupUsecase
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
	case databases.DatabaseTypeMariadb:
		return uc.restoreMariadbBackupUsecase.Execute(
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
		)
	case databases.DatabaseTypeMongodb:
		return uc.restoreMongodbBackupUsecase.Execute(
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
