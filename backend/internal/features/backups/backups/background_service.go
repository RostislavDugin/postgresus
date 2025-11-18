package backups

import (
	"log/slog"
	"postgresus-backend/internal/config"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/storages"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/period"
	"time"
)

type BackupBackgroundService struct {
	backupService       *BackupService
	backupRepository    *BackupRepository
	backupConfigService *backups_config.BackupConfigService
	storageService      *storages.StorageService

	lastBackupTime time.Time
	logger         *slog.Logger
}

func (s *BackupBackgroundService) Run() {
	s.lastBackupTime = time.Now().UTC()

	if err := s.failBackupsInProgress(); err != nil {
		s.logger.Error("Failed to fail backups in progress", "error", err)
		panic(err)
	}

	if config.IsShouldShutdown() {
		return
	}

	for {
		if config.IsShouldShutdown() {
			return
		}

		if err := s.cleanOldBackups(); err != nil {
			s.logger.Error("Failed to clean old backups", "error", err)
		}

		if err := s.runPendingBackups(); err != nil {
			s.logger.Error("Failed to run pending backups", "error", err)
		}

		s.lastBackupTime = time.Now().UTC()
		time.Sleep(1 * time.Minute)
	}
}

func (s *BackupBackgroundService) IsBackupsWorkerRunning() bool {
	// if last backup time is more than 5 minutes ago, return false
	return s.lastBackupTime.After(time.Now().UTC().Add(-5 * time.Minute))
}

func (s *BackupBackgroundService) failBackupsInProgress() error {
	backupsInProgress, err := s.backupRepository.FindByStatus(BackupStatusInProgress)
	if err != nil {
		return err
	}

	for _, backup := range backupsInProgress {
		backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(backup.DatabaseID)
		if err != nil {
			s.logger.Error("Failed to get backup config by database ID", "error", err)
			continue
		}

		failMessage := "Backup failed due to application restart"
		backup.FailMessage = &failMessage
		backup.Status = BackupStatusFailed
		backup.BackupSizeMb = 0

		s.backupService.SendBackupNotification(
			backupConfig,
			backup,
			backups_config.NotificationBackupFailed,
			&failMessage,
		)

		if err := s.backupRepository.Save(backup); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupBackgroundService) cleanOldBackups() error {
	enabledBackupConfigs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		backupStorePeriod := backupConfig.StorePeriod

		if backupStorePeriod == period.PeriodForever {
			continue
		}

		storeDuration := backupStorePeriod.ToDuration()
		dateBeforeBackupsShouldBeDeleted := time.Now().UTC().Add(-storeDuration)

		oldBackups, err := s.backupRepository.FindBackupsBeforeDate(
			backupConfig.DatabaseID,
			dateBeforeBackupsShouldBeDeleted,
		)
		if err != nil {
			s.logger.Error(
				"Failed to find old backups for database",
				"databaseId",
				backupConfig.DatabaseID,
				"error",
				err,
			)
			continue
		}

		for _, backup := range oldBackups {
			storage, err := s.storageService.GetStorageByID(backup.StorageID)
			if err != nil {
				s.logger.Error(
					"Failed to get storage by ID",
					"storageId",
					backup.StorageID,
					"error",
					err,
				)
				continue
			}

			encryptor := encryption.GetFieldEncryptor()
			err = storage.DeleteFile(encryptor, backup.ID)
			if err != nil {
				s.logger.Error("Failed to delete backup file", "backupId", backup.ID, "error", err)
			}

			if err := s.backupRepository.DeleteByID(backup.ID); err != nil {
				s.logger.Error("Failed to delete old backup", "backupId", backup.ID, "error", err)
				continue
			}

			s.logger.Info(
				"Deleted old backup",
				"backupId",
				backup.ID,
				"databaseId",
				backupConfig.DatabaseID,
			)
		}
	}

	return nil
}

func (s *BackupBackgroundService) runPendingBackups() error {
	enabledBackupConfigs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		if backupConfig.BackupInterval == nil {
			continue
		}

		lastBackup, err := s.backupRepository.FindLastByDatabaseID(backupConfig.DatabaseID)
		if err != nil {
			s.logger.Error(
				"Failed to get last backup for database",
				"databaseId",
				backupConfig.DatabaseID,
				"error",
				err,
			)
			continue
		}

		var lastBackupTime *time.Time
		if lastBackup != nil {
			lastBackupTime = &lastBackup.CreatedAt
		}

		remainedBackupTryCount := s.GetRemainedBackupTryCount(lastBackup)

		if backupConfig.BackupInterval.ShouldTriggerBackup(time.Now().UTC(), lastBackupTime) ||
			remainedBackupTryCount > 0 {
			s.logger.Info(
				"Triggering scheduled backup",
				"databaseId",
				backupConfig.DatabaseID,
				"intervalType",
				backupConfig.BackupInterval.Interval,
			)

			go s.backupService.MakeBackup(backupConfig.DatabaseID, remainedBackupTryCount == 1)
			s.logger.Info(
				"Successfully triggered scheduled backup",
				"databaseId",
				backupConfig.DatabaseID,
			)
		}
	}

	return nil
}

// GetRemainedBackupTryCount returns the number of remaining backup tries for a given backup.
// If the backup is not failed or the backup config does not allow retries, it returns 0.
// If the backup is failed and the backup config allows retries, it returns the number of remaining tries.
// If the backup is failed and the backup config does not allow retries, it returns 0.
func (s *BackupBackgroundService) GetRemainedBackupTryCount(lastBackup *Backup) int {
	if lastBackup == nil {
		return 0
	}

	if lastBackup.Status != BackupStatusFailed {
		return 0
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(lastBackup.DatabaseID)
	if err != nil {
		s.logger.Error("Failed to get backup config by database ID", "error", err)
		return 0
	}

	if !backupConfig.IsRetryIfFailed {
		return 0
	}

	maxFailedTriesCount := backupConfig.MaxFailedTriesCount

	lastBackups, err := s.backupRepository.FindByDatabaseIDWithLimit(
		lastBackup.DatabaseID,
		maxFailedTriesCount,
	)
	if err != nil {
		s.logger.Error("Failed to find last backups by database ID", "error", err)
		return 0
	}

	lastFailedBackups := make([]*Backup, 0)

	for _, backup := range lastBackups {
		if backup.Status == BackupStatusFailed {
			lastFailedBackups = append(lastFailedBackups, backup)
		}
	}

	return maxFailedTriesCount - len(lastFailedBackups)
}
