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

func (r *Restorer) MakeRestore(ctx context.Context, restoreID uuid.UUID) {
	logger := r.logger.With("restore_id", restoreID)

	// Get and delete cached DB credentials atomically
	dbCache := r.cacheUtil.GetAndDelete(restoreID.String())

	if dbCache == nil {
		// Cache miss - fail immediately
		restore, err := r.restoreRepository.FindByID(restoreID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get restore by ID after cache miss", "error", err)
			return
		}

		errMsg := "Database credentials expired or missing from cache (most likely due to instance restart)"
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore after cache miss", "error", err)
		}

		logger.ErrorContext(ctx, "restore failed: cache miss")
		return
	}

	restore, err := r.restoreRepository.FindByID(restoreID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get restore by ID", "error", err)
		return
	}

	backup, err := r.backupService.GetBackup(restore.BackupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup by ID", "backup_id", restore.BackupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := r.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database by ID", "database_id", databaseID, "error", err)
		return
	}

	backupConfig, err := r.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		logger.ErrorContext(ctx, "backup config storage ID is not defined")
		return
	}

	// Detached from the caller so a finished HTTP request cannot cancel a running restore.
	executionCtx, cancel := context.WithCancel(context.Background())
	r.restoreCancelManager.RegisterTask(restore.ID, cancel)
	defer r.restoreCancelManager.UnregisterTask(restore.ID)

	storage, err := r.storageService.GetStorageByID(executionCtx, *backupConfig.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	// Create restoring database from cached credentials
	restoringToDB := &databases.Database{
		Type:              database.Type,
		PostgresqlLogical: dbCache.PostgresqlLogicalDatabase,
		Mysql:             dbCache.MysqlDatabase,
		Mariadb:           dbCache.MariadbDatabase,
		Mongodb:           dbCache.MongodbDatabase,
	}

	// The restore target is the only endpoint this function connects to, so one tunnel covers
	// version detection, the TimescaleDB hooks and pg_restore itself.
	tunneledDatabase, err := databases.OpenTunnel(executionCtx, databases.OpenTunnelSpec{
		Database:  restoringToDB,
		Logger:    logger,
		Encryptor: r.fieldEncryptor,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to open the SSH tunnel to the restore target: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		logger.ErrorContext(ctx, "restore failed to open the ssh tunnel to the target", "error", err)

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
		}

		return
	}

	defer tunneledDatabase.Close()

	restoringToDBThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	if err := restoringToDBThroughTunnel.PopulateDbData(logger, r.fieldEncryptor); err != nil {
		errMsg := fmt.Sprintf("failed to auto-detect database data: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		logger.ErrorContext(ctx, "restore failed to auto-detect target database data", "error", err)

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
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

	logger.InfoContext(ctx, fmt.Sprintf("restore started: %s database %q", database.Type, database.Name),
		"backup_id", backup.ID, "database_id", databaseID, "storage_id", storage.ID)

	err = r.restoreBackupUsecase.Execute(
		executionCtx,
		backupConfig,
		*restore,
		database,
		restoringToDBThroughTunnel,
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
			logger.WarnContext(ctx, "restore was cancelled by user or system",
				"is_cancelled", isCancelled,
				"is_shutdown", isShutdown,
			)

			restore.Status = restores_core.RestoreStatusCanceled
			restore.RestoreDurationMs = time.Since(start).Milliseconds()

			if err := r.restoreRepository.Save(restore); err != nil {
				logger.ErrorContext(ctx, "failed to save cancelled restore", "error", err)
			}

			return
		}

		logger.ErrorContext(ctx, "restore execution failed",
			"backup_id", backup.ID,
			"database_id", databaseID,
			"database_type", database.Type,
			"storage_id", storage.ID,
			"storage_type", storage.Type,
			"error", err,
		)

		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
		}

		return
	}

	restore.Status = restores_core.RestoreStatusCompleted
	restore.RestoreDurationMs = time.Since(start).Milliseconds()

	if err := r.restoreRepository.Save(restore); err != nil {
		logger.ErrorContext(ctx, "failed to save restore", "error", err)
		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("restore finished in %d ms", restore.RestoreDurationMs),
		"backup_id", backup.ID)
}
