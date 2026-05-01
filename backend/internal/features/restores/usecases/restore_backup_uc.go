package usecases

import (
	"context"
	"errors"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	usecases_clickhouse "databasus-backend/internal/features/restores/usecases/clickhouse"
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
	restoreClickhouseBackupUsecase *usecases_clickhouse.RestoreClickhouseBackupUsecase
}

func (uc *RestoreBackupUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config.BackupConfig,
	restore restores_core.Restore,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backup *backups_core.Backup,
	storage *storages.Storage,
	isExcludeExtensions bool,
) error {
	switch originalDB.Type {
	case databases.DatabaseTypePostgres:
		return uc.restorePostgresqlBackupUsecase.Execute(
			ctx,
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
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
		)
	case databases.DatabaseTypeMariadb:
		return uc.restoreMariadbBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
		)
	case databases.DatabaseTypeMongodb:
		return uc.restoreMongodbBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storage,
		)
	case databases.DatabaseTypeClickhouse:
		return uc.restoreClickhouseBackupUsecase.Execute(
			ctx,
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
