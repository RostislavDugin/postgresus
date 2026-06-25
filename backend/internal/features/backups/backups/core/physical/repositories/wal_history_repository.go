package physical_repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalWalHistoryRepository struct{}

func (r *PhysicalWalHistoryRepository) Insert(file *physical_models.PhysicalWalHistoryFile) error {
	if file.DatabaseID == uuid.Nil || file.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(file).Error
}

func (r *PhysicalWalHistoryRepository) FindByDatabaseTimeline(
	databaseID uuid.UUID,
	timelineID int,
) (*physical_models.PhysicalWalHistoryFile, error) {
	var file physical_models.PhysicalWalHistoryFile

	err := storage.
		GetDb().
		Where("database_id = ? AND timeline_id = ?", databaseID, timelineID).
		First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &file, nil
}

func (r *PhysicalWalHistoryRepository) FindAllByDatabase(
	databaseID uuid.UUID,
) ([]*physical_models.PhysicalWalHistoryFile, error) {
	var files []*physical_models.PhysicalWalHistoryFile

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("timeline_id ASC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	return files, nil
}

func (r *PhysicalWalHistoryRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&physical_models.PhysicalWalHistoryFile{}, "id = ?", id).Error
}
