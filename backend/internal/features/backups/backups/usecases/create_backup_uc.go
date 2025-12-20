package usecases

import (
	"context"
	"errors"

	usecases_common "postgresus-backend/internal/features/backups/backups/usecases/common"
	usecases_mysql "postgresus-backend/internal/features/backups/backups/usecases/mysql"
	usecases_postgresql "postgresus-backend/internal/features/backups/backups/usecases/postgresql"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/storages"

	"github.com/google/uuid"
)

type CreateBackupUsecase struct {
	CreatePostgresqlBackupUsecase *usecases_postgresql.CreatePostgresqlBackupUsecase
	CreateMysqlBackupUsecase      *usecases_mysql.CreateMysqlBackupUsecase
}

func (uc *CreateBackupUsecase) Execute(
	ctx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*usecases_common.BackupMetadata, error) {
	switch database.Type {
	case databases.DatabaseTypePostgres:
		return uc.CreatePostgresqlBackupUsecase.Execute(
			ctx,
			backupID,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	case databases.DatabaseTypeMysql:
		return uc.CreateMysqlBackupUsecase.Execute(
			ctx,
			backupID,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)

	default:
		return nil, errors.New("database type not supported")
	}
}
