package backuping

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/period"
)

const (
	cleanerTickerInterval   = 1 * time.Minute
	recentBackupGracePeriod = 60 * time.Minute
)

type BackupCleaner struct {
	backupRepository      *backups_core.BackupRepository
	storageService        *storages.StorageService
	backupConfigService   *backups_config.BackupConfigService
	fieldEncryptor        util_encryption.FieldEncryptor
	logger                *slog.Logger
	backupRemoveListeners []backups_core.BackupRemoveListener

	runOnce sync.Once
	hasRun  atomic.Bool
}

func (c *BackupCleaner) Run(ctx context.Context) {
	wasAlreadyRun := c.hasRun.Load()

	c.runOnce.Do(func() {
		c.hasRun.Store(true)

		if ctx.Err() != nil {
			return
		}

		ticker := time.NewTicker(cleanerTickerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.cleanByRetentionPolicy(); err != nil {
					c.logger.Error("Failed to clean backups by retention policy", "error", err)
				}

				if err := c.cleanExceededBackups(); err != nil {
					c.logger.Error("Failed to clean exceeded backups", "error", err)
				}
			}
		}
	})

	if wasAlreadyRun {
		panic(fmt.Sprintf("%T.Run() called multiple times", c))
	}
}

func (c *BackupCleaner) DeleteBackup(backup *backups_core.Backup) error {
	for _, listener := range c.backupRemoveListeners {
		if err := listener.OnBeforeBackupRemove(backup); err != nil {
			return err
		}
	}

	storage, err := c.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return err
	}

	err = storage.DeleteFile(c.fieldEncryptor, backup.FileName)
	if err != nil {
		// we do not return error here, because sometimes clean up performed
		// before unavailable storage removal or change - therefore we should
		// proceed even in case of error. It's possible that some S3 or
		// storage is not available yet, it should not block us
		c.logger.Error("Failed to delete backup file", "error", err)
	}

	metadataFileName := backup.FileName + ".metadata"
	if err := storage.DeleteFile(c.fieldEncryptor, metadataFileName); err != nil {
		c.logger.Error("Failed to delete backup metadata file", "error", err)
	}

	return c.backupRepository.DeleteByID(backup.ID)
}

func (c *BackupCleaner) AddBackupRemoveListener(listener backups_core.BackupRemoveListener) {
	c.backupRemoveListeners = append(c.backupRemoveListeners, listener)
}

func (c *BackupCleaner) cleanByRetentionPolicy() error {
	enabledBackupConfigs, err := c.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		var cleanErr error

		switch backupConfig.RetentionPolicyType {
		case backups_config.RetentionPolicyTypeCount:
			cleanErr = c.cleanByCount(backupConfig)
		case backups_config.RetentionPolicyTypeGFS:
			cleanErr = c.cleanByGFS(backupConfig)
		default:
			cleanErr = c.cleanByTimePeriod(backupConfig)
		}

		if cleanErr != nil {
			c.logger.Error(
				"Failed to clean backups by retention policy",
				"databaseId", backupConfig.DatabaseID,
				"configId", backupConfig.ID,
				"policy", backupConfig.RetentionPolicyType,
				"error", cleanErr,
			)
		}
	}

	return nil
}

func (c *BackupCleaner) cleanExceededBackups() error {
	enabledBackupConfigs, err := c.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		if backupConfig.MaxBackupsTotalSizeMB <= 0 {
			continue
		}

		if err := c.cleanExceededBackupsForDatabase(
			backupConfig,
			backupConfig.MaxBackupsTotalSizeMB,
		); err != nil {
			c.logger.Error(
				"Failed to clean exceeded backups for database config",
				"databaseId",
				backupConfig.DatabaseID,
				"configId",
				backupConfig.ID,
				"error",
				err,
			)
			continue
		}
	}

	return nil
}

func (c *BackupCleaner) cleanByTimePeriod(backupConfig *backups_config.BackupConfig) error {
	if backupConfig.RetentionTimePeriod == "" {
		return nil
	}

	if backupConfig.RetentionTimePeriod == period.PeriodForever {
		return nil
	}

	storeDuration := backupConfig.RetentionTimePeriod.ToDuration()
	dateBeforeBackupsShouldBeDeleted := time.Now().UTC().Add(-storeDuration)

	oldBackups, err := c.backupRepository.FindBackupsBeforeDateByConfigID(
		backupConfig.ID,
		dateBeforeBackupsShouldBeDeleted,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to find old backups for config %s (database %s): %w",
			backupConfig.ID,
			backupConfig.DatabaseID,
			err,
		)
	}

	for _, backup := range oldBackups {
		if isRecentBackup(backup) {
			continue
		}

		if err := c.DeleteBackup(backup); err != nil {
			c.logger.Error("Failed to delete old backup", "backupId", backup.ID, "error", err)
			continue
		}

		c.logger.Info(
			"Deleted old backup",
			"backupId", backup.ID,
			"databaseId", backupConfig.DatabaseID,
			"configId", backupConfig.ID,
		)
	}

	return nil
}

func (c *BackupCleaner) cleanByCount(backupConfig *backups_config.BackupConfig) error {
	if backupConfig.RetentionCount <= 0 {
		return nil
	}

	completedBackups, err := c.backupRepository.FindByConfigIdAndStatus(
		backupConfig.ID,
		backups_core.BackupStatusCompleted,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to find completed backups for config %s (database %s): %w",
			backupConfig.ID,
			backupConfig.DatabaseID,
			err,
		)
	}

	// completedBackups are ordered newest first; delete everything beyond position RetentionCount
	if len(completedBackups) <= backupConfig.RetentionCount {
		return nil
	}

	toDelete := completedBackups[backupConfig.RetentionCount:]
	for _, backup := range toDelete {
		if isRecentBackup(backup) {
			continue
		}

		if err := c.DeleteBackup(backup); err != nil {
			c.logger.Error(
				"Failed to delete backup by count policy",
				"backupId",
				backup.ID,
				"error",
				err,
			)
			continue
		}

		c.logger.Info(
			"Deleted backup by count policy",
			"backupId", backup.ID,
			"databaseId", backupConfig.DatabaseID,
			"configId", backupConfig.ID,
			"retentionCount", backupConfig.RetentionCount,
		)
	}

	return nil
}

func (c *BackupCleaner) cleanByGFS(backupConfig *backups_config.BackupConfig) error {
	if backupConfig.RetentionGfsHours <= 0 && backupConfig.RetentionGfsDays <= 0 &&
		backupConfig.RetentionGfsWeeks <= 0 && backupConfig.RetentionGfsMonths <= 0 &&
		backupConfig.RetentionGfsYears <= 0 {
		return nil
	}

	completedBackups, err := c.backupRepository.FindByConfigIdAndStatus(
		backupConfig.ID,
		backups_core.BackupStatusCompleted,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to find completed backups for config %s (database %s): %w",
			backupConfig.ID,
			backupConfig.DatabaseID,
			err,
		)
	}

	keepSet := buildGFSKeepSet(
		completedBackups,
		backupConfig.RetentionGfsHours,
		backupConfig.RetentionGfsDays,
		backupConfig.RetentionGfsWeeks,
		backupConfig.RetentionGfsMonths,
		backupConfig.RetentionGfsYears,
	)

	for _, backup := range completedBackups {
		if keepSet[backup.ID] {
			continue
		}

		if isRecentBackup(backup) {
			continue
		}

		if err := c.DeleteBackup(backup); err != nil {
			c.logger.Error(
				"Failed to delete backup by GFS policy",
				"backupId",
				backup.ID,
				"error",
				err,
			)
			continue
		}

		c.logger.Info(
			"Deleted backup by GFS policy",
			"backupId", backup.ID,
			"databaseId", backupConfig.DatabaseID,
			"configId", backupConfig.ID,
		)
	}

	return nil
}

func (c *BackupCleaner) cleanExceededBackupsForDatabase(
	backupConfig *backups_config.BackupConfig,
	limitperDbMB int64,
) error {
	for {
		backupsTotalSizeMB, err := c.backupRepository.GetTotalSizeByConfigID(backupConfig.ID)
		if err != nil {
			return err
		}

		if backupsTotalSizeMB <= float64(limitperDbMB) {
			break
		}

		oldestBackups, err := c.backupRepository.FindOldestByConfigIDExcludingInProgress(
			backupConfig.ID,
			1,
		)
		if err != nil {
			return err
		}

		if len(oldestBackups) == 0 {
			c.logger.Warn(
				"No backups to delete but still over limit",
				"databaseId",
				backupConfig.DatabaseID,
				"configId",
				backupConfig.ID,
				"totalSizeMB",
				backupsTotalSizeMB,
				"limitMB",
				limitperDbMB,
			)
			break
		}

		backup := oldestBackups[0]
		if isRecentBackup(backup) {
			c.logger.Warn(
				"Oldest backup is too recent to delete, stopping size cleanup",
				"databaseId",
				backupConfig.DatabaseID,
				"configId",
				backupConfig.ID,
				"backupId",
				backup.ID,
				"totalSizeMB",
				backupsTotalSizeMB,
				"limitMB",
				limitperDbMB,
			)
			break
		}

		if err := c.DeleteBackup(backup); err != nil {
			c.logger.Error(
				"Failed to delete exceeded backup",
				"backupId",
				backup.ID,
				"databaseId",
				backupConfig.DatabaseID,
				"configId",
				backupConfig.ID,
				"error",
				err,
			)
			return err
		}

		c.logger.Info(
			"Deleted exceeded backup",
			"backupId",
			backup.ID,
			"databaseId",
			backupConfig.DatabaseID,
			"configId",
			backupConfig.ID,
			"backupSizeMB",
			backup.BackupSizeMb,
			"totalSizeMB",
			backupsTotalSizeMB,
			"limitMB",
			limitperDbMB,
		)
	}

	return nil
}

func isRecentBackup(backup *backups_core.Backup) bool {
	return time.Since(backup.CreatedAt) < recentBackupGracePeriod
}

// buildGFSKeepSet determines which backups to retain under the GFS rotation scheme.
// Backups must be sorted newest-first. A backup can fill multiple slots simultaneously
// (e.g. the newest backup of a year also fills the monthly, weekly, daily, and hourly slot).
func buildGFSKeepSet(
	backups []*backups_core.Backup,
	hours, days, weeks, months, years int,
) map[uuid.UUID]bool {
	keep := make(map[uuid.UUID]bool)

	hoursSeen := make(map[string]bool)
	daysSeen := make(map[string]bool)
	weeksSeen := make(map[string]bool)
	monthsSeen := make(map[string]bool)
	yearsSeen := make(map[string]bool)

	hoursKept, daysKept, weeksKept, monthsKept, yearsKept := 0, 0, 0, 0, 0

	for _, backup := range backups {
		t := backup.CreatedAt

		hourKey := t.Format("2006-01-02-15")
		dayKey := t.Format("2006-01-02")
		weekYear, week := t.ISOWeek()
		weekKey := fmt.Sprintf("%d-%02d", weekYear, week)
		monthKey := t.Format("2006-01")
		yearKey := t.Format("2006")

		if hours > 0 && hoursKept < hours && !hoursSeen[hourKey] {
			keep[backup.ID] = true
			hoursSeen[hourKey] = true
			hoursKept++
		}

		if days > 0 && daysKept < days && !daysSeen[dayKey] {
			keep[backup.ID] = true
			daysSeen[dayKey] = true
			daysKept++
		}

		if weeks > 0 && weeksKept < weeks && !weeksSeen[weekKey] {
			keep[backup.ID] = true
			weeksSeen[weekKey] = true
			weeksKept++
		}

		if months > 0 && monthsKept < months && !monthsSeen[monthKey] {
			keep[backup.ID] = true
			monthsSeen[monthKey] = true
			monthsKept++
		}

		if years > 0 && yearsKept < years && !yearsSeen[yearKey] {
			keep[backup.ID] = true
			yearsSeen[yearKey] = true
			yearsKept++
		}
	}

	return keep
}
