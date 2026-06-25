package backuping_logical

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"
)

type Backuper struct {
	databaseService     *databases.DatabaseService
	fieldEncryptor      util_encryption.FieldEncryptor
	workspaceService    *workspaces_services.WorkspaceService
	backupRepository    *backups_core_logical.BackupRepository
	backupConfigService *backups_config_logical.BackupConfigService
	storageService      *storages.StorageService
	notificationSender  backups_core_logical.NotificationSender
	backupCancelManager *tasks_cancellation.TaskCancelManager
	logger              *slog.Logger
	createBackupUseCase backups_core_logical.CreateBackupUsecase
}

func (b *Backuper) MakeBackup(backupID uuid.UUID, isCallNotifier bool) {
	backup, err := b.backupRepository.FindByID(backupID)
	if err != nil {
		b.logger.Error("Failed to get backup by ID", "backupId", backupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := b.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		b.logger.Error("Failed to get database by ID", "databaseId", databaseID, "error", err)
		return
	}

	backupConfig, err := b.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		b.logger.Error("Failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		b.logger.Error("Backup config storage ID is not defined")
		return
	}

	storage, err := b.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		b.logger.Error("Failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	ctx, cancel := context.WithCancel(context.Background())
	b.backupCancelManager.RegisterTask(backup.ID, cancel)
	defer b.backupCancelManager.UnregisterTask(backup.ID)

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		if err := b.backupRepository.Save(backup); err != nil {
			b.logger.Error("Failed to update backup progress", "error", err)
		}
	}

	backupMetadata, err := b.createBackupUseCase.Execute(
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
		currentBackup, fetchErr := b.backupRepository.FindByID(backup.ID)
		if fetchErr == nil && currentBackup.Status == backups_core_logical.BackupStatusFailed {
			b.logger.Warn(
				"Backup already marked as failed by progress listener, skipping error handling",
				"backupId",
				backup.ID,
				"failMessage",
				*currentBackup.FailMessage,
			)

			// Still call notification for size limit failures
			b.SendBackupNotification(
				backupConfig,
				currentBackup,
				backups_config_logical.NotificationBackupFailed,
				currentBackup.FailMessage,
			)

			return
		}

		errMsg := err.Error()

		// Log detailed error information for debugging
		b.logger.Error("Backup execution failed",
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
			b.logger.Warn("Backup was cancelled by user or system",
				"backupId", backup.ID,
				"isCancelled", isCancelled,
				"isShutdown", isShutdown,
			)

			backup.Status = backups_core_logical.BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := b.backupRepository.Save(backup); err != nil {
				b.logger.Error("Failed to save cancelled backup", "error", err)
			}

			// Delete partial backup from storage
			storage, storageErr := b.storageService.GetStorageByID(backup.StorageID)
			if storageErr == nil {
				if deleteErr := storage.DeleteFile(b.fieldEncryptor, backup.FileName); deleteErr != nil {
					b.logger.Error(
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
		backup.Status = backups_core_logical.BackupStatusFailed
		backup.BackupDurationMs = time.Since(start).Milliseconds()
		backup.BackupSizeMb = 0

		if updateErr := b.databaseService.SetBackupError(databaseID, errMsg); updateErr != nil {
			b.logger.Error(
				"Failed to update database last backup time",
				"databaseId",
				databaseID,
				"error",
				updateErr,
			)
		}

		if err := b.backupRepository.Save(backup); err != nil {
			b.logger.Error("Failed to save backup", "error", err)
		}

		b.SendBackupNotification(
			backupConfig,
			backup,
			backups_config_logical.NotificationBackupFailed,
			&errMsg,
		)

		return
	}

	backup.BackupDurationMs = time.Since(start).Milliseconds()

	// Update backup with encryption metadata if provided
	if backupMetadata != nil {
		backupMetadata.BackupID = backup.ID

		if err := backupMetadata.Validate(); err != nil {
			b.logger.Error("Failed to validate backup metadata", "error", err)
			return
		}

		backup.EncryptionSalt = backupMetadata.EncryptionSalt
		backup.EncryptionIV = backupMetadata.EncryptionIV
		backup.Encryption = backupMetadata.Encryption
	}

	if backupMetadata != nil {
		metadataJSON, err := json.Marshal(backupMetadata)
		if err != nil {
			b.logger.Error("Failed to marshal backup metadata to JSON",
				"backupId", backup.ID,
				"error", err,
			)
		} else {
			metadataReader := bytes.NewReader(metadataJSON)
			metadataFileName := backup.FileName + ".metadata"

			if err := storage.SaveFile(
				context.Background(),
				b.fieldEncryptor,
				b.logger,
				metadataFileName,
				metadataReader,
			); err != nil {
				b.logger.Error("Failed to save backup metadata file to storage",
					"backupId", backup.ID,
					"fileName", metadataFileName,
					"error", err,
				)
			} else {
				b.logger.Info("Backup metadata file saved successfully",
					"backupId", backup.ID,
					"fileName", metadataFileName,
				)
			}
		}
	}

	backup.Status = backups_core_logical.BackupStatusCompleted

	if err := b.backupRepository.Save(backup); err != nil {
		b.logger.Error("Failed to save backup", "error", err)
		return
	}

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := b.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		b.logger.Error(
			"Failed to update database last backup time",
			"databaseId",
			databaseID,
			"error",
			updateErr,
		)
	}

	if backup.Status != backups_core_logical.BackupStatusCompleted && !isCallNotifier {
		return
	}

	b.SendBackupNotification(
		backupConfig,
		backup,
		backups_config_logical.NotificationBackupSuccess,
		nil,
	)
}

func (b *Backuper) SendBackupNotification(
	backupConfig *backups_config_logical.LogicalBackupConfig,
	backup *backups_core_logical.LogicalBackup,
	notificationType backups_config_logical.BackupNotificationType,
	errorMessage *string,
) {
	database, err := b.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		return
	}

	workspace, err := b.workspaceService.GetWorkspaceByID(*database.WorkspaceID)
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
		case backups_config_logical.NotificationBackupFailed:
			title = fmt.Sprintf(
				"❌ Backup failed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		case backups_config_logical.NotificationBackupSuccess:
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

		b.notificationSender.SendNotification(
			&notifier,
			title,
			message,
		)
	}
}
