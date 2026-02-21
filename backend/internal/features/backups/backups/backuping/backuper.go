package backuping

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"
)

const (
	heartbeatTickerInterval     = 15 * time.Second
	backuperHeathcheckThreshold = 5 * time.Minute
)

type BackuperNode struct {
	databaseService     *databases.DatabaseService
	fieldEncryptor      util_encryption.FieldEncryptor
	workspaceService    *workspaces_services.WorkspaceService
	backupRepository    *backups_core.BackupRepository
	backupConfigService *backups_config.BackupConfigService
	storageService      *storages.StorageService
	notificationSender  backups_core.NotificationSender
	backupCancelManager *tasks_cancellation.TaskCancelManager
	backupNodesRegistry *BackupNodesRegistry
	logger              *slog.Logger
	createBackupUseCase backups_core.CreateBackupUsecase
	nodeID              uuid.UUID

	lastHeartbeat time.Time

	runOnce sync.Once
	hasRun  atomic.Bool
}

func (n *BackuperNode) Run(ctx context.Context) {
	wasAlreadyRun := n.hasRun.Load()

	n.runOnce.Do(func() {
		n.hasRun.Store(true)

		n.lastHeartbeat = time.Now().UTC()

		throughputMBs := config.GetEnv().NodeNetworkThroughputMBs

		backupNode := BackupNode{
			ID:            n.nodeID,
			ThroughputMBs: throughputMBs,
			LastHeartbeat: time.Now().UTC(),
		}

		if err := n.backupNodesRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), backupNode); err != nil {
			n.logger.Error("Failed to register node in registry", "error", err)
			panic(err)
		}

		backupHandler := func(backupID uuid.UUID, isCallNotifier bool) {
			go func() {
				n.MakeBackup(backupID, isCallNotifier)
				if err := n.backupNodesRegistry.PublishBackupCompletion(n.nodeID, backupID); err != nil {
					n.logger.Error(
						"Failed to publish backup completion",
						"error",
						err,
						"backupID",
						backupID,
					)
				}
			}()
		}

		err := n.backupNodesRegistry.SubscribeNodeForBackupsAssignment(n.nodeID, backupHandler)
		if err != nil {
			n.logger.Error("Failed to subscribe to backup assignments", "error", err)
			panic(err)
		}
		defer func() {
			if err := n.backupNodesRegistry.UnsubscribeNodeForBackupsAssignments(); err != nil {
				n.logger.Error("Failed to unsubscribe from backup assignments", "error", err)
			}
		}()

		ticker := time.NewTicker(heartbeatTickerInterval)
		defer ticker.Stop()

		n.logger.Info("Backup node started", "nodeID", n.nodeID, "throughput", throughputMBs)

		for {
			select {
			case <-ctx.Done():
				n.logger.Info("Shutdown signal received, unregistering node", "nodeID", n.nodeID)

				if err := n.backupNodesRegistry.UnregisterNodeFromRegistry(backupNode); err != nil {
					n.logger.Error("Failed to unregister node from registry", "error", err)
				}

				return
			case <-ticker.C:
				n.sendHeartbeat(&backupNode)
			}
		}
	})

	if wasAlreadyRun {
		panic(fmt.Sprintf("%T.Run() called multiple times", n))
	}
}

func (n *BackuperNode) IsBackuperRunning() bool {
	return n.lastHeartbeat.After(time.Now().UTC().Add(-backuperHeathcheckThreshold))
}

func (n *BackuperNode) MakeBackup(backupID uuid.UUID, isCallNotifier bool) {
	backup, err := n.backupRepository.FindByID(backupID)
	if err != nil {
		n.logger.Error("Failed to get backup by ID", "backupId", backupID, "error", err)
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

	ctx, cancel := context.WithCancel(context.Background())
	n.backupCancelManager.RegisterTask(backup.ID, cancel)
	defer n.backupCancelManager.UnregisterTask(backup.ID)

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		// Check size limit (0 = unlimited)
		if backupConfig.MaxBackupSizeMB > 0 &&
			completedMBs > float64(backupConfig.MaxBackupSizeMB) {
			errMsg := fmt.Sprintf(
				"backup size (%.2f MB) exceeded maximum allowed size (%d MB)",
				completedMBs,
				backupConfig.MaxBackupSizeMB,
			)

			backup.Status = backups_core.BackupStatusFailed
			backup.IsSkipRetry = true
			backup.FailMessage = &errMsg
			if err := n.backupRepository.Save(backup); err != nil {
				n.logger.Error("Failed to save backup with size exceeded error", "error", err)
			}
			cancel() // Cancel the backup context

			return
		}

		if err := n.backupRepository.Save(backup); err != nil {
			n.logger.Error("Failed to update backup progress", "error", err)
		}
	}

	backupMetadata, err := n.createBackupUseCase.Execute(
		ctx,
		backup,
		backupConfig,
		database,
		storage,
		backupProgressListener,
	)
	if err != nil {
		// Check if backup was already marked as failed by progress listener (e.g., size limit exceeded)
		// If so, skip error handling to avoid overwriting the status
		currentBackup, fetchErr := n.backupRepository.FindByID(backup.ID)
		if fetchErr == nil && currentBackup.Status == backups_core.BackupStatusFailed {
			n.logger.Warn(
				"Backup already marked as failed by progress listener, skipping error handling",
				"backupId",
				backup.ID,
				"failMessage",
				*currentBackup.FailMessage,
			)

			// Still call notification for size limit failures
			n.SendBackupNotification(
				backupConfig,
				currentBackup,
				backups_config.NotificationBackupFailed,
				currentBackup.FailMessage,
			)

			return
		}

		errMsg := err.Error()

		// Log detailed error information for debugging
		n.logger.Error("Backup execution failed",
			"backupId", backup.ID,
			"databaseId", databaseID,
			"databaseType", database.Type,
			"storageId", storage.ID,
			"storageType", storage.Type,
			"error", err,
			"errorMessage", errMsg,
		)

		// Check if backup was cancelled (not due to shutdown)
		isCancelled := strings.Contains(errMsg, "backup cancelled") ||
			strings.Contains(errMsg, "context canceled") ||
			errors.Is(err, context.Canceled)
		isShutdown := strings.Contains(errMsg, "shutdown")

		if isCancelled && !isShutdown {
			n.logger.Warn("Backup was cancelled by user or system",
				"backupId", backup.ID,
				"isCancelled", isCancelled,
				"isShutdown", isShutdown,
			)

			backup.Status = backups_core.BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := n.backupRepository.Save(backup); err != nil {
				n.logger.Error("Failed to save cancelled backup", "error", err)
			}

			// Delete partial backup from storage
			storage, storageErr := n.storageService.GetStorageByID(backup.StorageID)
			if storageErr == nil {
				if deleteErr := storage.DeleteFile(n.fieldEncryptor, backup.FileName); deleteErr != nil {
					n.logger.Error(
						"Failed to delete partial backup file",
						"backupId",
						backup.ID,
						"error",
						deleteErr,
					)
				}
			}

			return
		}

		backup.FailMessage = &errMsg
		backup.Status = backups_core.BackupStatusFailed
		backup.BackupDurationMs = time.Since(start).Milliseconds()
		backup.BackupSizeMb = 0

		if updateErr := n.databaseService.SetBackupError(databaseID, errMsg); updateErr != nil {
			n.logger.Error(
				"Failed to update database last backup time",
				"databaseId",
				databaseID,
				"error",
				updateErr,
			)
		}

		if err := n.backupRepository.Save(backup); err != nil {
			n.logger.Error("Failed to save backup", "error", err)
		}

		n.SendBackupNotification(
			backupConfig,
			backup,
			backups_config.NotificationBackupFailed,
			&errMsg,
		)

		return
	}

	backup.Status = backups_core.BackupStatusCompleted
	backup.BackupDurationMs = time.Since(start).Milliseconds()

	// Update backup with encryption metadata if provided
	if backupMetadata != nil {
		backupMetadata.BackupID = backup.ID

		if err := backupMetadata.Validate(); err != nil {
			n.logger.Error("Failed to validate backup metadata", "error", err)
			return
		}

		backup.EncryptionSalt = backupMetadata.EncryptionSalt
		backup.EncryptionIV = backupMetadata.EncryptionIV
		backup.Encryption = backupMetadata.Encryption
	}

	if err := n.backupRepository.Save(backup); err != nil {
		n.logger.Error("Failed to save backup", "error", err)
		return
	}

	// Save metadata file to storage
	if backupMetadata != nil {
		metadataJSON, err := json.Marshal(backupMetadata)
		if err != nil {
			n.logger.Error("Failed to marshal backup metadata to JSON",
				"backupId", backup.ID,
				"error", err,
			)
		} else {
			metadataReader := bytes.NewReader(metadataJSON)
			metadataFileName := backup.FileName + ".metadata"

			if err := storage.SaveFile(
				context.Background(),
				n.fieldEncryptor,
				n.logger,
				metadataFileName,
				metadataReader,
			); err != nil {
				n.logger.Error("Failed to save backup metadata file to storage",
					"backupId", backup.ID,
					"fileName", metadataFileName,
					"error", err,
				)
			} else {
				n.logger.Info("Backup metadata file saved successfully",
					"backupId", backup.ID,
					"fileName", metadataFileName,
				)
			}
		}
	}

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := n.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		n.logger.Error(
			"Failed to update database last backup time",
			"databaseId",
			databaseID,
			"error",
			updateErr,
		)
	}

	if backup.Status != backups_core.BackupStatusCompleted && !isCallNotifier {
		return
	}

	n.SendBackupNotification(
		backupConfig,
		backup,
		backups_config.NotificationBackupSuccess,
		nil,
	)
}

func (n *BackuperNode) SendBackupNotification(
	backupConfig *backups_config.BackupConfig,
	backup *backups_core.Backup,
	notificationType backups_config.BackupNotificationType,
	errorMessage *string,
) {
	database, err := n.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		return
	}

	workspace, err := n.workspaceService.GetWorkspaceByID(*database.WorkspaceID)
	if err != nil {
		return
	}

	for _, notifier := range database.Notifiers {
		if !slices.Contains(
			backupConfig.SendNotificationsOn,
			notificationType,
		) {
			continue
		}

		title := ""
		switch notificationType {
		case backups_config.NotificationBackupFailed:
			title = fmt.Sprintf(
				"❌ Backup failed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		case backups_config.NotificationBackupSuccess:
			title = fmt.Sprintf(
				"✅ Backup completed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		}

		message := ""
		if errorMessage != nil {
			message = *errorMessage
		} else {
			// Format size conditionally
			var sizeStr string
			if backup.BackupSizeMb < 1024 {
				sizeStr = fmt.Sprintf("%.2f MB", backup.BackupSizeMb)
			} else {
				sizeGB := backup.BackupSizeMb / 1024
				sizeStr = fmt.Sprintf("%.2f GB", sizeGB)
			}

			// Format duration as "0m 0s 0ms"
			totalMs := backup.BackupDurationMs
			minutes := totalMs / (1000 * 60)
			seconds := (totalMs % (1000 * 60)) / 1000
			durationStr := fmt.Sprintf("%dm %ds", minutes, seconds)

			message = fmt.Sprintf(
				"Backup completed successfully in %s.\nCompressed backup size: %s",
				durationStr,
				sizeStr,
			)
		}

		n.notificationSender.SendNotification(
			&notifier,
			title,
			message,
		)
	}
}

func (n *BackuperNode) sendHeartbeat(backupNode *BackupNode) {
	n.lastHeartbeat = time.Now().UTC()
	if err := n.backupNodesRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), *backupNode); err != nil {
		n.logger.Error("Failed to send heartbeat", "error", err)
	}
}
