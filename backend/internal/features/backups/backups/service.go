package backups

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	audit_logs "postgresus-backend/internal/features/audit_logs"
	"postgresus-backend/internal/features/backups/backups/encryption"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"
	users_models "postgresus-backend/internal/features/users/models"
	users_repositories "postgresus-backend/internal/features/users/repositories"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	util_encryption "postgresus-backend/internal/util/encryption"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BackupService struct {
	databaseService     *databases.DatabaseService
	storageService      *storages.StorageService
	backupRepository    *BackupRepository
	notifierService     *notifiers.NotifierService
	notificationSender  NotificationSender
	backupConfigService *backups_config.BackupConfigService
	secretKeyRepo       *users_repositories.SecretKeyRepository
	fieldEncryptor      util_encryption.FieldEncryptor

	createBackupUseCase CreateBackupUsecase

	logger *slog.Logger

	backupRemoveListeners []BackupRemoveListener

	workspaceService     *workspaces_services.WorkspaceService
	auditLogService      *audit_logs.AuditLogService
	backupContextManager *BackupContextManager
}

func (s *BackupService) AddBackupRemoveListener(listener BackupRemoveListener) {
	s.backupRemoveListeners = append(s.backupRemoveListeners, listener)
}

func (s *BackupService) OnBeforeBackupsStorageChange(databaseID uuid.UUID) error {
	err := s.deleteDbBackups(databaseID)
	if err != nil {
		return err
	}

	return nil
}

func (s *BackupService) OnBeforeDatabaseRemove(databaseID uuid.UUID) error {
	err := s.deleteDbBackups(databaseID)
	if err != nil {
		return err
	}

	return nil
}

func (s *BackupService) MakeBackupWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot create backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to create backup for this database")
	}

	go s.MakeBackup(databaseID, true)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup manually initiated for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return nil
}

func (s *BackupService) GetBackups(
	user *users_models.User,
	databaseID uuid.UUID,
	limit, offset int,
) (*GetBackupsResponse, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot get backups for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access backups for this database")
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	backups, err := s.backupRepository.FindByDatabaseIDWithPagination(databaseID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.backupRepository.CountByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	return &GetBackupsResponse{
		Backups: backups,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *BackupService) DeleteBackup(
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot delete backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete backup for this database")
	}

	if backup.Status == BackupStatusInProgress {
		return errors.New("backup is in progress")
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf(
			"Backup deleted for database: %s (ID: %s)",
			database.Name,
			backupID.String(),
		),
		&user.ID,
		database.WorkspaceID,
	)

	return s.deleteBackup(backup)
}

func (s *BackupService) MakeBackup(databaseID uuid.UUID, isLastTry bool) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		s.logger.Error("Failed to get database by ID", "error", err)
		return
	}

	lastBackup, err := s.backupRepository.FindLastByDatabaseID(databaseID)
	if err != nil {
		s.logger.Error("Failed to find last backup by database ID", "error", err)
		return
	}

	if lastBackup != nil && lastBackup.Status == BackupStatusInProgress {
		s.logger.Error("Backup is in progress")
		return
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		s.logger.Error("Failed to get backup config by database ID", "error", err)
		return
	}

	if !backupConfig.IsBackupsEnabled {
		s.logger.Info("Backups are not enabled for this database")
		return
	}

	if backupConfig.StorageID == nil {
		s.logger.Error("Backup config storage ID is not defined")
		return
	}

	storage, err := s.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		s.logger.Error("Failed to get storage by ID", "error", err)
		return
	}

	backup := &Backup{
		DatabaseID: databaseID,
		StorageID:  storage.ID,

		Status: BackupStatusInProgress,

		BackupSizeMb: 0,

		CreatedAt: time.Now().UTC(),
	}

	if err := s.backupRepository.Save(backup); err != nil {
		s.logger.Error("Failed to save backup", "error", err)
		return
	}

	start := time.Now().UTC()

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		if err := s.backupRepository.Save(backup); err != nil {
			s.logger.Error("Failed to update backup progress", "error", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.backupContextManager.RegisterBackup(backup.ID, cancel)
	defer s.backupContextManager.UnregisterBackup(backup.ID)

	backupMetadata, err := s.createBackupUseCase.Execute(
		ctx,
		backup.ID,
		backupConfig,
		database,
		storage,
		backupProgressListener,
	)
	if err != nil {
		errMsg := err.Error()

		// Check if backup was cancelled (not due to shutdown)
		if strings.Contains(errMsg, "backup cancelled") && !strings.Contains(errMsg, "shutdown") {
			backup.Status = BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := s.backupRepository.Save(backup); err != nil {
				s.logger.Error("Failed to save cancelled backup", "error", err)
			}

			// Delete partial backup from storage
			storage, storageErr := s.storageService.GetStorageByID(backup.StorageID)
			if storageErr == nil {
				if deleteErr := storage.DeleteFile(s.fieldEncryptor, backup.ID); deleteErr != nil {
					s.logger.Error(
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
		backup.Status = BackupStatusFailed
		backup.BackupDurationMs = time.Since(start).Milliseconds()
		backup.BackupSizeMb = 0

		if updateErr := s.databaseService.SetBackupError(databaseID, errMsg); updateErr != nil {
			s.logger.Error(
				"Failed to update database last backup time",
				"databaseId",
				databaseID,
				"error",
				updateErr,
			)
		}

		if err := s.backupRepository.Save(backup); err != nil {
			s.logger.Error("Failed to save backup", "error", err)
		}

		s.SendBackupNotification(
			backupConfig,
			backup,
			backups_config.NotificationBackupFailed,
			&errMsg,
		)

		return
	}

	backup.Status = BackupStatusCompleted
	backup.BackupDurationMs = time.Since(start).Milliseconds()

	// Update backup with encryption metadata if provided
	if backupMetadata != nil {
		backup.EncryptionSalt = backupMetadata.EncryptionSalt
		backup.EncryptionIV = backupMetadata.EncryptionIV
		backup.Encryption = backupMetadata.Encryption
	}

	if err := s.backupRepository.Save(backup); err != nil {
		s.logger.Error("Failed to save backup", "error", err)
		return
	}

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := s.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		s.logger.Error(
			"Failed to update database last backup time",
			"databaseId",
			databaseID,
			"error",
			updateErr,
		)
	}

	if backup.Status != BackupStatusCompleted && !isLastTry {
		return
	}

	s.SendBackupNotification(
		backupConfig,
		backup,
		backups_config.NotificationBackupSuccess,
		nil,
	)
}

func (s *BackupService) SendBackupNotification(
	backupConfig *backups_config.BackupConfig,
	backup *Backup,
	notificationType backups_config.BackupNotificationType,
	errorMessage *string,
) {
	database, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		return
	}

	workspace, err := s.workspaceService.GetWorkspaceByID(*database.WorkspaceID)
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

		s.notificationSender.SendNotification(
			&notifier,
			title,
			message,
		)
	}
}

func (s *BackupService) GetBackup(backupID uuid.UUID) (*Backup, error) {
	return s.backupRepository.FindByID(backupID)
}

func (s *BackupService) CancelBackup(
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot cancel backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to cancel backup for this database")
	}

	if backup.Status != BackupStatusInProgress {
		return errors.New("backup is not in progress")
	}

	if err := s.backupContextManager.CancelBackup(backupID); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf(
			"Backup cancelled for database: %s (ID: %s)",
			database.Name,
			backupID.String(),
		),
		&user.ID,
		database.WorkspaceID,
	)

	return nil
}

func (s *BackupService) GetBackupFile(
	user *users_models.User,
	backupID uuid.UUID,
) (io.ReadCloser, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot download backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		*database.WorkspaceID,
		user,
	)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to download backup for this database")
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf(
			"Backup file downloaded for database: %s (ID: %s)",
			database.Name,
			backupID.String(),
		),
		&user.ID,
		database.WorkspaceID,
	)

	return s.getBackupReader(backupID)
}

func (s *BackupService) deleteBackup(backup *Backup) error {
	for _, listener := range s.backupRemoveListeners {
		if err := listener.OnBeforeBackupRemove(backup); err != nil {
			return err
		}
	}

	storage, err := s.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return err
	}

	err = storage.DeleteFile(s.fieldEncryptor, backup.ID)
	if err != nil {
		// we do not return error here, because sometimes clean up performed
		// before unavailable storage removal or change - therefore we should
		// proceed even in case of error
		s.logger.Error("Failed to delete backup file", "error", err)
	}

	return s.backupRepository.DeleteByID(backup.ID)
}

func (s *BackupService) deleteDbBackups(databaseID uuid.UUID) error {
	dbBackupsInProgress, err := s.backupRepository.FindByDatabaseIdAndStatus(
		databaseID,
		BackupStatusInProgress,
	)
	if err != nil {
		return err
	}

	if len(dbBackupsInProgress) > 0 {
		return errors.New("backup is in progress, storage cannot be removed")
	}

	dbBackups, err := s.backupRepository.FindByDatabaseID(
		databaseID,
	)
	if err != nil {
		return err
	}

	for _, dbBackup := range dbBackups {
		err := s.deleteBackup(dbBackup)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetBackupReader returns a reader for the backup file
// If encrypted, wraps with DecryptionReader
func (s *BackupService) getBackupReader(backupID uuid.UUID) (io.ReadCloser, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup: %w", err)
	}

	storage, err := s.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileReader, err := storage.GetFile(s.fieldEncryptor, backup.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file: %w", err)
	}

	// If not encrypted, return raw reader
	if backup.Encryption == backups_config.BackupEncryptionNone {
		s.logger.Info("Returning non-encrypted backup", "backupId", backupID)
		return fileReader, nil
	}

	// Decrypt on-the-fly for encrypted backups
	if backup.Encryption != backups_config.BackupEncryptionEncrypted {
		if err := fileReader.Close(); err != nil {
			s.logger.Error("Failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("unsupported encryption type: %s", backup.Encryption)
	}

	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		if err := fileReader.Close(); err != nil {
			s.logger.Error("Failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("backup marked as encrypted but missing encryption metadata")
	}

	// Get master key
	masterKey, err := s.secretKeyRepo.GetSecretKey()
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	// Decode salt and IV
	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	// Wrap with decrypting reader
	decryptionReader, err := encryption.NewDecryptionReader(
		fileReader,
		masterKey,
		backup.ID,
		salt,
		iv,
	)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create decrypting reader: %w", err)
	}

	s.logger.Info("Returning encrypted backup with decryption", "backupId", backupID)

	return &decryptionReaderCloser{
		decryptionReader,
		fileReader,
	}, nil
}
