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
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"
)

const partialBackupCleanupTimeout = 30 * time.Second

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

func (b *Backuper) MakeBackup(ctx context.Context, backupID uuid.UUID, isCallNotifier bool) {
	backup, err := b.backupRepository.FindByID(backupID)
	if err != nil {
		b.logger.ErrorContext(ctx, "failed to get backup by ID", "backup_id", backupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID
	logger := b.logger.With("backup_id", backupID, "database_id", databaseID)

	database, err := b.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database by ID", "error", err)
		return
	}

	backupConfig, err := b.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		logger.ErrorContext(ctx, "backup config storage ID is not defined")
		return
	}

	// Detached from the caller so a finished HTTP request cannot cancel a running backup.
	executionCtx, cancel := context.WithCancel(context.Background())
	b.backupCancelManager.RegisterTask(backup.ID, cancel)
	defer b.backupCancelManager.UnregisterTask(backup.ID)

	storage, err := b.storageService.GetStorageByID(executionCtx, *backupConfig.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		if err := b.backupRepository.Save(backup); err != nil {
			logger.ErrorContext(ctx, "failed to update backup progress", "error", err)
		}
	}

	backupMetadata, err := b.createBackupUseCase.Execute(
		executionCtx,
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
			logger.WarnContext(ctx,
				"backup already marked as failed by progress listener, skipping error handling",
				"backup_id",
				backup.ID,
				"fail_message",
				*currentBackup.FailMessage,
			)

			// Still call notification for size limit failures
			b.SendBackupNotification(ctx,
				backupConfig,
				currentBackup,
				backups_config_logical.NotificationBackupFailed,
				currentBackup.FailMessage,
			)

			return
		}

		errMsg := err.Error()

		// Log detailed error information for debugging
		logger.ErrorContext(ctx, "backup execution failed",
			"backup_id", backup.ID,
			"database_id", databaseID,
			"database_type", database.Type,
			"storage_id", storage.ID,
			"storage_type", storage.Type,
			"error", err,
		)

		// Check if backup was cancelled (not due to shutdown)
		isCancelled := strings.Contains(errMsg, "backup cancelled") ||
			strings.Contains(errMsg, "context canceled") ||
			errors.Is(err, context.Canceled)
		isShutdown := strings.Contains(errMsg, "shutdown")

		if isCancelled && !isShutdown {
			logger.WarnContext(ctx, "backup was cancelled by user or system",
				"backup_id", backup.ID,
				"is_cancelled", isCancelled,
				"is_shutdown", isShutdown,
			)

			backup.Status = backups_core_logical.BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := b.backupRepository.Save(backup); err != nil {
				logger.ErrorContext(ctx, "failed to save cancelled backup", "error", err)
			}

			// This branch is reached because executionCtx was cancelled, so the cleanup needs a live
			// context of its own or the partial file is never deleted.
			cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), partialBackupCleanupTimeout)
			defer cancelCleanup()

			partialBackupStorage, storageErr := b.storageService.GetStorageByID(cleanupCtx, backup.StorageID)
			if storageErr == nil {
				if deleteErr := partialBackupStorage.DeleteFile(
					cleanupCtx,
					b.fieldEncryptor,
					logger,
					backup.FileName,
				); deleteErr != nil {
					logger.ErrorContext(ctx,
						"failed to delete partial backup file",
						"backup_id",
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
			logger.ErrorContext(ctx,
				"failed to update database last backup time",
				"error",
				updateErr,
			)
		}

		if err := b.backupRepository.Save(backup); err != nil {
			logger.ErrorContext(ctx, "failed to save backup", "error", err)
		}

		logger.ErrorContext(ctx, fmt.Sprintf("logical backup failed after %d ms: %s",
			backup.BackupDurationMs, errMsg))

		b.SendBackupNotification(ctx,
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
			logger.ErrorContext(ctx, "failed to validate backup metadata", "error", err)
			return
		}

		backup.EncryptionSalt = backupMetadata.EncryptionSalt
		backup.EncryptionIV = backupMetadata.EncryptionIV
		backup.Encryption = backupMetadata.Encryption
	}

	if backupMetadata != nil {
		metadataJSON, err := json.Marshal(backupMetadata)
		if err != nil {
			logger.ErrorContext(ctx, "failed to marshal backup metadata to JSON",
				"backup_id", backup.ID,
				"error", err,
			)
		} else {
			metadataReader := bytes.NewReader(metadataJSON)
			metadataFileName := backup.FileName + ".metadata"

			// Not executionCtx: the dump itself already succeeded, and a cancel landing here would
			// leave a completed backup whose encryption metadata is missing, making it unrestorable.
			if err := storage.SaveFile(
				context.Background(),
				b.fieldEncryptor,
				logger,
				metadataFileName,
				metadataReader,
			); err != nil {
				logger.ErrorContext(ctx, "failed to save backup metadata file to storage",
					"backup_id", backup.ID,
					"file_name", metadataFileName,
					"error", err,
				)
			} else {
				logger.DebugContext(ctx, "backup metadata file saved", "file_name", metadataFileName)
			}
		}
	}

	backup.Status = backups_core_logical.BackupStatusCompleted

	if err := b.backupRepository.Save(backup); err != nil {
		logger.ErrorContext(ctx, "failed to save backup", "error", err)
		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("logical backup finished: %.1f MB in %d ms",
		backup.BackupSizeMb, backup.BackupDurationMs), "file_name", backup.FileName)

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := b.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		logger.ErrorContext(ctx,
			"failed to update database last backup time",
			"error",
			updateErr,
		)
	}

	if backup.Status != backups_core_logical.BackupStatusCompleted && !isCallNotifier {
		return
	}

	b.SendBackupNotification(ctx,
		backupConfig,
		backup,
		backups_config_logical.NotificationBackupSuccess,
		nil,
	)
}

func (b *Backuper) SendBackupNotification(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	backup *backups_core_logical.LogicalBackup,
	notificationType backups_config_logical.BackupNotificationType,
	errorMessage *string,
) {
	logger := b.logger.With("backup_id", backup.ID, "database_id", backupConfig.DatabaseID)

	database, err := b.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database for the backup notification", "error", err)

		return
	}

	workspace, err := b.workspaceService.GetWorkspaceByID(*database.WorkspaceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get workspace for the backup notification", "error", err)

		return
	}

	for _, notifier := range database.Notifiers {
		if !slices.Contains(
			backupConfig.SendNotificationsOn,
			notificationType,
		) {
			logger.DebugContext(ctx, fmt.Sprintf("skipping %s notification, not in the configured set",
				notificationType), "notifier_id", notifier.ID)

			continue
		}

		title := ""
		sentNotificationType := notifier_models.NotificationTypeBackupSuccess

		switch notificationType {
		case backups_config_logical.NotificationBackupFailed:
			sentNotificationType = notifier_models.NotificationTypeBackupFailed
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

		b.notificationSender.SendNotification(ctx,
			&notifier,
			notifier_models.Notification{
				Type:    sentNotificationType,
				Heading: title,
				Message: message,
			},
		)
	}
}
