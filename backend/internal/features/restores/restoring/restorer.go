package restoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	cache_utils "databasus-backend/internal/util/cache"
	util_encryption "databasus-backend/internal/util/encryption"
)

type Restorer struct {
	databaseService      *databases.DatabaseService
	backupService        *backups_services.LogicalBackupService
	fieldEncryptor       util_encryption.FieldEncryptor
	restoreRepository    *restores_core.RestoreRepository
	backupConfigService  *backups_config_logical.BackupConfigService
	storageService       *storages.StorageService
	logger               *slog.Logger
	restoreBackupUsecase restores_core.RestoreBackupUsecase
	cacheUtil            *cache_utils.CacheUtil[RestoreDatabaseCache]
	restoreCancelManager *tasks_cancellation.TaskCancelManager
}

func (r *Restorer) MakeRestore(restoreID uuid.UUID) {
	// Get and delete cached DB credentials atomically
	dbCache := r.cacheUtil.GetAndDelete(restoreID.String())

	if dbCache == nil {
		// Cache miss - fail immediately
		restore, err := r.restoreRepository.FindByID(restoreID)
		if err != nil {
			r.logger.Error(
				"Failed to get restore by ID after cache miss",
				"restoreId",
				restoreID,
				"error",
				err,
			)
			return
		}

		errMsg := "Database credentials expired or missing from cache (most likely due to instance restart)"
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed

		if err := r.restoreRepository.Save(restore); err != nil {
			r.logger.Error("Failed to save restore after cache miss", "error", err)
		}

		r.logger.Error("Restore failed: cache miss", "restoreId", restoreID)
		return
	}

	restore, err := r.restoreRepository.FindByID(restoreID)
	if err != nil {
		r.logger.Error("Failed to get restore by ID", "restoreId", restoreID, "error", err)
		return
	}

	backup, err := r.backupService.GetBackup(restore.BackupID)
	if err != nil {
		r.logger.Error("Failed to get backup by ID", "backupId", restore.BackupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := r.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		r.logger.Error("Failed to get database by ID", "databaseId", databaseID, "error", err)
		return
	}

	backupConfig, err := r.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		r.logger.Error("Failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		r.logger.Error("Backup config storage ID is not defined")
		return
	}

	storage, err := r.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		r.logger.Error("Failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	r.restoreCancelManager.RegisterTask(restore.ID, cancel)
	defer r.restoreCancelManager.UnregisterTask(restore.ID)

	// Create restoring database from cached credentials
	restoringToDB := &databases.Database{
		Type:              database.Type,
		PostgresqlLogical: dbCache.PostgresqlLogicalDatabase,
		Mysql:             dbCache.MysqlDatabase,
		Mariadb:           dbCache.MariadbDatabase,
		Mongodb:           dbCache.MongodbDatabase,
	}

	if err := restoringToDB.PopulateDbData(r.logger, r.fieldEncryptor); err != nil {
		errMsg := fmt.Sprintf("failed to auto-detect database data: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := r.restoreRepository.Save(restore); err != nil {
			r.logger.Error("Failed to save restore", "error", err)
		}

		return
	}

	// IsExcludeExtensions is a transient choice carried on the target config from the restore
	// request; IsSkipUserMappings is a persisted property of the source database being restored.
	restoreOptions := restores_core.RestoreOptions{}
	if dbCache.PostgresqlLogicalDatabase != nil {
		restoreOptions.IsExcludeExtensions = dbCache.PostgresqlLogicalDatabase.IsExcludeExtensions
	}
	if database.PostgresqlLogical != nil {
		restoreOptions.IsSkipUserMappings = database.PostgresqlLogical.IsSkipUserMappings
	}

	err = r.restoreBackupUsecase.Execute(
		ctx,
		backupConfig,
		*restore,
		database,
		restoringToDB,
		backup,
		storage,
		restoreOptions,
	)
	if err != nil {
		errMsg := err.Error()

		// Check if restore was cancelled
		isCancelled := strings.Contains(errMsg, "restore cancelled") ||
			strings.Contains(errMsg, "context canceled") ||
			errors.Is(err, context.Canceled)
		isShutdown := strings.Contains(errMsg, "shutdown")

		if isCancelled && !isShutdown {
			r.logger.Warn("Restore was cancelled by user or system",
				"restoreId", restore.ID,
				"isCancelled", isCancelled,
				"isShutdown", isShutdown,
			)

			restore.Status = restores_core.RestoreStatusCanceled
			restore.RestoreDurationMs = time.Since(start).Milliseconds()

			if err := r.restoreRepository.Save(restore); err != nil {
				r.logger.Error("Failed to save cancelled restore", "error", err)
			}

			return
		}

		r.logger.Error("Restore execution failed",
			"restoreId", restore.ID,
			"backupId", backup.ID,
			"databaseId", databaseID,
			"databaseType", database.Type,
			"storageId", storage.ID,
			"storageType", storage.Type,
			"error", err,
			"errorMessage", errMsg,
		)

		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := r.restoreRepository.Save(restore); err != nil {
			r.logger.Error("Failed to save restore", "error", err)
		}

		return
	}

	restore.Status = restores_core.RestoreStatusCompleted
	restore.RestoreDurationMs = time.Since(start).Milliseconds()

	if err := r.restoreRepository.Save(restore); err != nil {
		r.logger.Error("Failed to save restore", "error", err)
		return
	}

	r.logger.Info(
		"Restore completed successfully",
		"restoreId", restore.ID,
		"backupId", backup.ID,
		"durationMs", restore.RestoreDurationMs,
	)
}
