package backups_config

import (
	"databasus-backend/internal/storage"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupConfigRepository struct{}

func (r *BackupConfigRepository) Save(
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	db := storage.GetDb()

	err := db.Transaction(func(tx *gorm.DB) error {
		// Handle BackupInterval
		if backupConfig.BackupInterval != nil {
			if backupConfig.BackupInterval.ID == uuid.Nil {
				if err := tx.Create(backupConfig.BackupInterval).Error; err != nil {
					return err
				}

				backupConfig.BackupIntervalID = backupConfig.BackupInterval.ID
			} else {
				if err := tx.Save(backupConfig.BackupInterval).Error; err != nil {
					return err
				}

				backupConfig.BackupIntervalID = backupConfig.BackupInterval.ID
			}
		}

		// Set storage ID
		if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
			backupConfig.StorageID = &backupConfig.Storage.ID
		}

		// Use Save which handles both create and update based on primary key
		if err := tx.Save(backupConfig).
			Omit("BackupInterval", "Storage").
			Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return backupConfig, nil
}

func (r *BackupConfigRepository) FindByDatabaseID(databaseID uuid.UUID) (*BackupConfig, error) {
	var backupConfig BackupConfig

	if err := storage.
		GetDb().
		Preload("BackupInterval").
		Preload("Storage").
		Where("database_id = ?", databaseID).
		First(&backupConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backupConfig, nil
}

func (r *BackupConfigRepository) GetWithEnabledBackups() ([]*BackupConfig, error) {
	var backupConfigs []*BackupConfig

	if err := storage.
		GetDb().
		Preload("BackupInterval").
		Preload("Storage").
		Where("is_backups_enabled = ?", true).
		Find(&backupConfigs).Error; err != nil {
		return nil, err
	}

	return backupConfigs, nil
}

func (r *BackupConfigRepository) IsStorageUsing(storageID uuid.UUID) (bool, error) {
	var count int64

	if err := storage.
		GetDb().
		Table("backup_configs").
		Where("storage_id = ?", storageID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *BackupConfigRepository) GetDatabasesIDsByStorageID(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	var databasesIDs []uuid.UUID

	if err := storage.
		GetDb().
		Table("backup_configs").
		Where("storage_id = ?", storageID).
		Pluck("database_id", &databasesIDs).Error; err != nil {
		return nil, err
	}

	return databasesIDs, nil
}
