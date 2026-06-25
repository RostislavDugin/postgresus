package restores_core

import (
	"github.com/google/uuid"

	"databasus-backend/internal/storage"
)

type RestoreRepository struct{}

func (r *RestoreRepository) Save(restore *Restore) error {
	db := storage.GetDb()

	isNew := restore.ID == uuid.Nil
	if isNew {
		restore.ID = uuid.New()
		return db.Create(restore).
			Omit("Backup", "PostgresqlLogicalDatabase", "MysqlDatabase", "MariadbDatabase", "MongodbDatabase").
			Error
	}

	return db.Save(restore).
		Omit("Backup", "PostgresqlLogicalDatabase", "MysqlDatabase", "MariadbDatabase", "MongodbDatabase").
		Error
}

func (r *RestoreRepository) FindByBackupID(backupID uuid.UUID) ([]*Restore, error) {
	var restores []*Restore

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

func (r *RestoreRepository) FindByID(id uuid.UUID) (*Restore, error) {
	var restore Restore

	if err := storage.
		GetDb().
		Preload("Backup").
		Where("id = ?", id).
		First(&restore).Error; err != nil {
		return nil, err
	}

	return &restore, nil
}

func (r *RestoreRepository) FindByStatus(status RestoreStatus) ([]*Restore, error) {
	var restores []*Restore

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

func (r *RestoreRepository) FindInProgressRestoresByDatabaseID(
	databaseID uuid.UUID,
) ([]*Restore, error) {
	var restores []*Restore

	if err := storage.
		GetDb().
		Preload("Backup").
		Joins("JOIN logical_backups ON logical_backups.id = restores.backup_id").
		Where("logical_backups.database_id = ? AND restores.status = ?", databaseID, RestoreStatusInProgress).
		Order("restores.created_at DESC").
		Find(&restores).Error; err != nil {
		return nil, err
	}

	return restores, nil
}

func (r *RestoreRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&Restore{}, "id = ?", id).Error
}
