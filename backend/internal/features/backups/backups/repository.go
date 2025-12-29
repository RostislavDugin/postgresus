package backups

import (
	"databasus-backend/internal/storage"
	"errors"

	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupRepository struct{}

func (r *BackupRepository) Save(backup *Backup) error {
	if backup.DatabaseID == uuid.Nil || backup.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	db := storage.GetDb()

	isNew := backup.ID == uuid.Nil
	if isNew {
		backup.ID = uuid.New()
		return db.Create(backup).
			Error
	}

	return db.Save(backup).
		Error
}

func (r *BackupRepository) FindByDatabaseID(databaseID uuid.UUID) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIDWithLimit(
	databaseID uuid.UUID,
	limit int,
) ([]*Backup, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than 0")
	}

	var backups []*Backup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByStorageID(storageID uuid.UUID) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("storage_id = ?", storageID).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindLastByDatabaseID(databaseID uuid.UUID) (*Backup, error) {
	var backup Backup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&backup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *BackupRepository) FindByID(id uuid.UUID) (*Backup, error) {
	var backup Backup

	if err := storage.
		GetDb().
		Where("id = ?", id).
		First(&backup).Error; err != nil {
		return nil, err
	}

	return &backup, nil
}

func (r *BackupRepository) FindByStatus(status BackupStatus) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByStorageIdAndStatus(
	storageID uuid.UUID,
	status BackupStatus,
) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("storage_id = ? AND status = ?", storageID, status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIdAndStatus(
	databaseID uuid.UUID,
	status BackupStatus,
) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("database_id = ? AND status = ?", databaseID, status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&Backup{}, "id = ?", id).Error
}

func (r *BackupRepository) FindBackupsBeforeDate(
	databaseID uuid.UUID,
	date time.Time,
) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("database_id = ? AND created_at < ?", databaseID, date).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIDWithPagination(
	databaseID uuid.UUID,
	limit, offset int,
) ([]*Backup, error) {
	var backups []*Backup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) CountByDatabaseID(databaseID uuid.UUID) (int64, error) {
	var count int64

	if err := storage.
		GetDb().
		Model(&Backup{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
