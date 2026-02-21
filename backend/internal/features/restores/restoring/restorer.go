package restoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/backups/backups"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	cache_utils "databasus-backend/internal/util/cache"
	util_encryption "databasus-backend/internal/util/encryption"
)

const (
	heartbeatTickerInterval      = 15 * time.Second
	restorerHealthcheckThreshold = 5 * time.Minute
)

type RestorerNode struct {
	nodeID uuid.UUID

	databaseService      *databases.DatabaseService
	backupService        *backups.BackupService
	fieldEncryptor       util_encryption.FieldEncryptor
	restoreRepository    *restores_core.RestoreRepository
	backupConfigService  *backups_config.BackupConfigService
	storageService       *storages.StorageService
	restoreNodesRegistry *RestoreNodesRegistry
	logger               *slog.Logger
	restoreBackupUsecase restores_core.RestoreBackupUsecase
	cacheUtil            *cache_utils.CacheUtil[RestoreDatabaseCache]
	restoreCancelManager *tasks_cancellation.TaskCancelManager

	lastHeartbeat time.Time

	runOnce sync.Once
	hasRun  atomic.Bool
}

func (n *RestorerNode) Run(ctx context.Context) {
	wasAlreadyRun := n.hasRun.Load()

	n.runOnce.Do(func() {
		n.hasRun.Store(true)

		n.lastHeartbeat = time.Now().UTC()

		throughputMBs := config.GetEnv().NodeNetworkThroughputMBs

		restoreNode := RestoreNode{
			ID:            n.nodeID,
			ThroughputMBs: throughputMBs,
		}

		if err := n.restoreNodesRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), restoreNode); err != nil {
			n.logger.Error("Failed to register node in registry", "error", err)
			panic(err)
		}

		restoreHandler := func(restoreID uuid.UUID, isCallNotifier bool) {
			n.MakeRestore(restoreID)
			if err := n.restoreNodesRegistry.PublishRestoreCompletion(n.nodeID, restoreID); err != nil {
				n.logger.Error(
					"Failed to publish restore completion",
					"error",
					err,
					"restoreID",
					restoreID,
				)
			}
		}

		err := n.restoreNodesRegistry.SubscribeNodeForRestoresAssignment(
			n.nodeID,
			restoreHandler,
		)
		if err != nil {
			n.logger.Error("Failed to subscribe to restore assignments", "error", err)
			panic(err)
		}
		defer func() {
			if err := n.restoreNodesRegistry.UnsubscribeNodeForRestoresAssignments(); err != nil {
				n.logger.Error("Failed to unsubscribe from restore assignments", "error", err)
			}
		}()

		ticker := time.NewTicker(heartbeatTickerInterval)
		defer ticker.Stop()

		n.logger.Info("Restore node started", "nodeID", n.nodeID, "throughput", throughputMBs)

		for {
			select {
			case <-ctx.Done():
				n.logger.Info("Shutdown signal received, unregistering node", "nodeID", n.nodeID)

				if err := n.restoreNodesRegistry.UnregisterNodeFromRegistry(restoreNode); err != nil {
					n.logger.Error("Failed to unregister node from registry", "error", err)
				}

				return
			case <-ticker.C:
				n.sendHeartbeat(&restoreNode)
			}
		}
	})

	if wasAlreadyRun {
		panic(fmt.Sprintf("%T.Run() called multiple times", n))
	}
}

func (n *RestorerNode) IsRestorerRunning() bool {
	return n.lastHeartbeat.After(time.Now().UTC().Add(-restorerHealthcheckThreshold))
}

func (n *RestorerNode) MakeRestore(restoreID uuid.UUID) {
	// Get and delete cached DB credentials atomically
	dbCache := n.cacheUtil.GetAndDelete(restoreID.String())

	if dbCache == nil {
		// Cache miss - fail immediately
		restore, err := n.restoreRepository.FindByID(restoreID)
		if err != nil {
			n.logger.Error(
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

		if err := n.restoreRepository.Save(restore); err != nil {
			n.logger.Error("Failed to save restore after cache miss", "error", err)
		}

		n.logger.Error("Restore failed: cache miss", "restoreId", restoreID)
		return
	}

	restore, err := n.restoreRepository.FindByID(restoreID)
	if err != nil {
		n.logger.Error("Failed to get restore by ID", "restoreId", restoreID, "error", err)
		return
	}

	backup, err := n.backupService.GetBackup(restore.BackupID)
	if err != nil {
		n.logger.Error("Failed to get backup by ID", "backupId", restore.BackupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := n.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		n.logger.Error("Failed to get database by ID", "databaseId", databaseID, "error", err)
		return
	}

	backupConfigs, err := n.backupConfigService.GetBackupConfigsByDatabaseID(databaseID)
	backupConfig := backupConfigs[0]
	if err != nil {
		n.logger.Error("Failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		n.logger.Error("Backup config storage ID is not defined")
		return
	}

	storage, err := n.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		n.logger.Error("Failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	n.restoreCancelManager.RegisterTask(restore.ID, cancel)
	defer n.restoreCancelManager.UnregisterTask(restore.ID)

	// Create restoring database from cached credentials
	restoringToDB := &databases.Database{
		Type:       database.Type,
		Postgresql: dbCache.PostgresqlDatabase,
		Mysql:      dbCache.MysqlDatabase,
		Mariadb:    dbCache.MariadbDatabase,
		Mongodb:    dbCache.MongodbDatabase,
	}

	if err := restoringToDB.PopulateDbData(n.logger, n.fieldEncryptor); err != nil {
		errMsg := fmt.Sprintf("failed to auto-detect database data: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := n.restoreRepository.Save(restore); err != nil {
			n.logger.Error("Failed to save restore", "error", err)
		}

		return
	}

	isExcludeExtensions := false
	if dbCache.PostgresqlDatabase != nil {
		isExcludeExtensions = dbCache.PostgresqlDatabase.IsExcludeExtensions
	}

	err = n.restoreBackupUsecase.Execute(
		ctx,
		backupConfig,
		*restore,
		database,
		restoringToDB,
		backup,
		storage,
		isExcludeExtensions,
	)

	if err != nil {
		errMsg := err.Error()

		// Check if restore was cancelled
		isCancelled := strings.Contains(errMsg, "restore cancelled") ||
			strings.Contains(errMsg, "context canceled") ||
			errors.Is(err, context.Canceled)
		isShutdown := strings.Contains(errMsg, "shutdown")

		if isCancelled && !isShutdown {
			n.logger.Warn("Restore was cancelled by user or system",
				"restoreId", restore.ID,
				"isCancelled", isCancelled,
				"isShutdown", isShutdown,
			)

			restore.Status = restores_core.RestoreStatusCanceled
			restore.RestoreDurationMs = time.Since(start).Milliseconds()

			if err := n.restoreRepository.Save(restore); err != nil {
				n.logger.Error("Failed to save cancelled restore", "error", err)
			}

			return
		}

		n.logger.Error("Restore execution failed",
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

		if err := n.restoreRepository.Save(restore); err != nil {
			n.logger.Error("Failed to save restore", "error", err)
		}

		return
	}

	restore.Status = restores_core.RestoreStatusCompleted
	restore.RestoreDurationMs = time.Since(start).Milliseconds()

	if err := n.restoreRepository.Save(restore); err != nil {
		n.logger.Error("Failed to save restore", "error", err)
		return
	}

	n.logger.Info(
		"Restore completed successfully",
		"restoreId", restore.ID,
		"backupId", backup.ID,
		"durationMs", restore.RestoreDurationMs,
	)
}

func (n *RestorerNode) sendHeartbeat(restoreNode *RestoreNode) {
	n.lastHeartbeat = time.Now().UTC()
	if err := n.restoreNodesRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), *restoreNode); err != nil {
		n.logger.Error("Failed to send heartbeat", "error", err)
	}
}
