package usecases

import (
	"context"
	"errors"

	common "databasus-backend/internal/features/backups/backups/common"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	usecases_clickhouse "databasus-backend/internal/features/backups/backups/usecases/clickhouse"
	usecases_mariadb "databasus-backend/internal/features/backups/backups/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/backups/backups/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/backups/backups/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/backups/backups/usecases/postgresql"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
)

type CreateBackupUsecase struct {
	CreatePostgresqlBackupUsecase *usecases_postgresql.CreatePostgresqlBackupUsecase
	CreateMysqlBackupUsecase      *usecases_mysql.CreateMysqlBackupUsecase
	CreateMariadbBackupUsecase    *usecases_mariadb.CreateMariadbBackupUsecase
	CreateMongodbBackupUsecase    *usecases_mongodb.CreateMongodbBackupUsecase
	CreateClickhouseBackupUsecase *usecases_clickhouse.CreateClickhouseBackupUsecase
}

func (uc *CreateBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core.Backup,
	backupConfig *backups_config.BackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*common.BackupMetadata, error) {
	switch database.Type {
	case databases.DatabaseTypePostgres:
		return uc.CreatePostgresqlBackupUsecase.Execute(
			ctx,
			backup,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	case databases.DatabaseTypeMysql:
		return uc.CreateMysqlBackupUsecase.Execute(
			ctx,
			backup,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	case databases.DatabaseTypeMariadb:
		return uc.CreateMariadbBackupUsecase.Execute(
			ctx,
			backup,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	case databases.DatabaseTypeMongodb:
		return uc.CreateMongodbBackupUsecase.Execute(
			ctx,
			backup,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	case databases.DatabaseTypeClickhouse:
		return uc.CreateClickhouseBackupUsecase.Execute(
			ctx,
			backup,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	default:
		return nil, errors.New("database type not supported")
	}
}
