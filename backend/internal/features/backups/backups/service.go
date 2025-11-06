package backups

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"
	users_models "postgresus-backend/internal/features/users/models"
	"slices"
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

	createBackupUseCase CreateBackupUsecase

	logger *slog.Logger

	backupRemoveListeners []BackupRemoveListener
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

	if database.UserID != user.ID {
		return errors.New("user does not have access to this database")
	}

	go s.MakeBackup(databaseID, true)

	return nil
}

func (s *BackupService) GetBackups(
	user *users_models.User,
	databaseID uuid.UUID,
) ([]*Backup, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.UserID != user.ID {
		return nil, errors.New("user does not have access to this database")
	}

	backups, err := s.backupRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

func (s *BackupService) DeleteBackup(
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	if backup.Database.UserID != user.ID {
		return errors.New("user does not have access to this backup")
	}

	if backup.Status == BackupStatusInProgress {
		return errors.New("backup is in progress")
	}

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
		Database:   database,

		StorageID: storage.ID,
		Storage:   storage,

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

	err = s.createBackupUseCase.Execute(
		backup.ID,
		backupConfig,
		database,
		storage,
		backupProgressListener,
	)
	if err != nil {
		errMsg := err.Error()
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

	for _, notifier := range database.Notifiers {
		if !slices.Contains(
			backupConfig.SendNotificationsOn,
			notificationType,
		) {
			continue
		}

		// Determine status icon and status text
		var statusIcon, statusText string
		if notificationType == backups_config.NotificationBackupFailed {
			statusIcon = "❌"
			statusText = "failed"
		} else {
			statusIcon = "✅"
			statusText = "success"
		}

		title := ""
		switch notificationType {
		case backups_config.NotificationBackupFailed:
			title = fmt.Sprintf("%s Backup failed for database \"%s\"", statusIcon, database.Name)
		case backups_config.NotificationBackupSuccess:
			title = fmt.Sprintf("%s Backup completed for database \"%s\"", statusIcon, database.Name)
		}

		message := ""
		var sizeStr, durationStr, errorMsg string

		if errorMessage != nil {
			message = *errorMessage
			errorMsg = *errorMessage
		} else {
			// Format size conditionally
			if backup.BackupSizeMb < 1024 {
				sizeStr = fmt.Sprintf("%.2f MB", backup.BackupSizeMb)
			} else {
				sizeGB := backup.BackupSizeMb / 1024
				sizeStr = fmt.Sprintf("%.2f GB", sizeGB)
			}

			// Format duration as "0m 0s"
			totalMs := backup.BackupDurationMs
			minutes := totalMs / (1000 * 60)
			seconds := (totalMs % (1000 * 60)) / 1000
			durationStr = fmt.Sprintf("%dm %ds", minutes, seconds)

			message = fmt.Sprintf(
				"Backup completed successfully in %s.\nCompressed backup size: %s",
				durationStr,
				sizeStr,
			)
		}

		// Build the vars map with all available variables
		vars := map[string]string{
			"status":        statusIcon,
			"status_text":   statusText,
			"database_name": database.Name,
			"duration":      durationStr,
			"size":          sizeStr,
			"error":         errorMsg,
			// For backward compatibility:
			"heading": title,
			"message": message,
		}

		s.notificationSender.SendNotification(
			&notifier,
			vars,
		)
	}
}

func (s *BackupService) GetBackup(backupID uuid.UUID) (*Backup, error) {
	return s.backupRepository.FindByID(backupID)
}

func (s *BackupService) GetBackupFile(
	user *users_models.User,
	backupID uuid.UUID,
) (io.ReadCloser, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	if backup.Database.UserID != user.ID {
		return nil, errors.New("user does not have access to this backup")
	}

	storage, err := s.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return nil, err
	}

	return storage.GetFile(backup.ID)
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

	err = storage.DeleteFile(backup.ID)
	if err != nil {
		return err
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
