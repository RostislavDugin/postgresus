package backups

import (
	"time"

	"github.com/google/uuid"
)

type Backup struct {
	ID uuid.UUID `json:"id" gorm:"column:id;type:uuid;primaryKey"`

	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	Status      BackupStatus `json:"status"      gorm:"column:status;not null"`
	FailMessage *string      `json:"failMessage" gorm:"column:fail_message"`

	BackupSizeMb float64 `json:"backupSizeMb" gorm:"column:backup_size_mb;default:0"`

	BackupDurationMs int64 `json:"backupDurationMs" gorm:"column:backup_duration_ms;default:0"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}
