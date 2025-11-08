package usecases

import (
	"context"
	"errors"
	usecases_postgresql "postgresus-backend/internal/features/backups/backups/usecases/postgresql"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/storages"

	"github.com/google/uuid"
)

type CreateBackupUsecase struct {
	CreatePostgresqlBackupUsecase *usecases_postgresql.CreatePostgresqlBackupUsecase
}

// Execute creates a backup of the database and returns the backup size in MB
func (uc *CreateBackupUsecase) Execute(
	ctx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(
		completedMBs float64,
	),
) error {
	if database.Type == databases.DatabaseTypePostgres {
		return uc.CreatePostgresqlBackupUsecase.Execute(
			ctx,
			backupID,
			backupConfig,
			database,
			storage,
			backupProgressListener,
		)
	}

	return errors.New("database type not supported")
}
