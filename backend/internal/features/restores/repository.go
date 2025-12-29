package restores

import (
	"databasus-backend/internal/features/restores/enums"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/storage"

	"github.com/google/uuid"
)

type RestoreRepository struct{}

func (r *RestoreRepository) Save(restore *models.Restore) error {
	db := storage.GetDb()

	isNew := restore.ID == uuid.Nil
	if isNew {
		restore.ID = uuid.New()
		return db.Create(restore).
			Omit("Backup").
			Error
	}

	return db.Save(restore).
		Omit("Backup").
		Error
}

func (r *RestoreRepository) FindByBackupID(backupID uuid.UUID) ([]*models.Restore, error) {
	var restores []*models.Restore

	if err := storage.
		GetDb().
		Preload("Backup").
		Where("backup_id = ?", backupID).
		Order("created_at DESC").
		Find(&restores).Error; err != nil {
		return nil, err
	}

	return restores, nil
}

func (r *RestoreRepository) FindByID(id uuid.UUID) (*models.Restore, error) {
	var restore models.Restore

	if err := storage.
		GetDb().
		Preload("Backup").
		Where("id = ?", id).
		First(&restore).Error; err != nil {
		return nil, err
	}

	return &restore, nil
}

func (r *RestoreRepository) FindByStatus(status enums.RestoreStatus) ([]*models.Restore, error) {
	var restores []*models.Restore

	if err := storage.
		GetDb().
		Preload("Backup").
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&restores).Error; err != nil {
		return nil, err
	}

	return restores, nil
}

func (r *RestoreRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&models.Restore{}, "id = ?", id).Error
}
