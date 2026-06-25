package physical_repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalIncrementalBackupRepository struct{}

func (r *PhysicalIncrementalBackupRepository) Save(incrementalBackup *physical_models.PhysicalIncrementalBackup) error {
	if incrementalBackup.DatabaseID == uuid.Nil || incrementalBackup.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}
	if incrementalBackup.RootFullBackupID == uuid.Nil {
		return errors.New("root full backup ID is required")
	}

	db := storage.GetDb()

	if incrementalBackup.ID == uuid.Nil {
		incrementalBackup.ID = uuid.New()
		if incrementalBackup.CreatedAt.IsZero() {
			incrementalBackup.CreatedAt = time.Now().UTC()
		}

		return db.Create(incrementalBackup).Error
	}

	return db.Save(incrementalBackup).Error
}

func (r *PhysicalIncrementalBackupRepository) FindByID(
	id uuid.UUID,
) (*physical_models.PhysicalIncrementalBackup, error) {
	var backup physical_models.PhysicalIncrementalBackup

	if err := storage.GetDb().Where("id = ?", id).First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *PhysicalIncrementalBackupRepository) FindLatestCompletedByRootFull(
	rootFullBackupID uuid.UUID,
) (*physical_models.PhysicalIncrementalBackup, error) {
	var backup physical_models.PhysicalIncrementalBackup

	err := storage.
		GetDb().
		Where("root_full_backup_id = ? AND status = ?",
			rootFullBackupID, physical_enums.PhysicalBackupStatusCompleted).
		Order("start_lsn DESC").
		First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *PhysicalIncrementalBackupRepository) FindAllByRootFull(
	rootFullBackupID uuid.UUID,
) ([]*physical_models.PhysicalIncrementalBackup, error) {
	var backups []*physical_models.PhysicalIncrementalBackup

	if err := storage.
		GetDb().
		Where("root_full_backup_id = ?", rootFullBackupID).
		Order("start_lsn ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalIncrementalBackupRepository) FindByIncrementalParent(
	parentIncrementalBackupID uuid.UUID,
) ([]*physical_models.PhysicalIncrementalBackup, error) {
	var backups []*physical_models.PhysicalIncrementalBackup

	if err := storage.
		GetDb().
		Where("parent_incremental_backup_id = ?", parentIncrementalBackupID).
		Order("start_lsn ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalIncrementalBackupRepository) UpdateStatus(
	id uuid.UUID,
	status physical_enums.PhysicalBackupStatus,
	errorReason *physical_enums.PhysicalBackupErrorReason,
) error {
	updates := map[string]any{"status": status, "error_reason": errorReason}

	if status == physical_enums.PhysicalBackupStatusCompleted {
		updates["completed_at"] = new(time.Now().UTC())
	}

	return storage.
		GetDb().
		Model(&physical_models.PhysicalIncrementalBackup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateProgress writes the in-flight size and duration for a still-running
// backup. It touches only those two columns and is gated on IN_PROGRESS — the
// same supersession guard as saveTerminalResultIfInProgress — so a late progress
// write can never resurrect a row a restart-recovery sweep already finalized.
func (r *PhysicalIncrementalBackupRepository) UpdateProgress(id uuid.UUID, sizeMb float64, durationMs int64) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalIncrementalBackup{}).
		Where("id = ? AND status = ?", id, physical_enums.PhysicalBackupStatusInProgress).
		Updates(map[string]any{"backup_size_mb": sizeMb, "backup_duration_ms": durationMs}).Error
}

func (r *PhysicalIncrementalBackupRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&physical_models.PhysicalIncrementalBackup{}, "id = ?", id).Error
}

func (r *PhysicalIncrementalBackupRepository) FindLastByDatabase(
	databaseID uuid.UUID,
) (*physical_models.PhysicalIncrementalBackup, error) {
	var backup physical_models.PhysicalIncrementalBackup

	err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

// FindAllInProgress returns every INCR still marked IN_PROGRESS across all
// databases — the INCR counterpart to the FULL restart sweep.
func (r *PhysicalIncrementalBackupRepository) FindAllInProgress() ([]*physical_models.PhysicalIncrementalBackup, error) {
	var backups []*physical_models.PhysicalIncrementalBackup

	if err := storage.
		GetDb().
		Where("status = ?", physical_enums.PhysicalBackupStatusInProgress).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}
