package backups_core

import (
	backups_config "databasus-backend/internal/features/backups/config"
	"time"

	"github.com/google/uuid"
)

type Backup struct {
	ID       uuid.UUID `json:"id"       gorm:"column:id;type:uuid;primaryKey"`
	FileName string    `json:"fileName" gorm:"column:file_name;type:text;not null"`

	DatabaseID     uuid.UUID  `json:"databaseId"     gorm:"column:database_id;type:uuid;not null"`
	BackupConfigID *uuid.UUID `json:"backupConfigId" gorm:"column:backup_config_id;type:uuid"`
	StorageID      uuid.UUID  `json:"storageId"      gorm:"column:storage_id;type:uuid;not null"`

	Status      BackupStatus `json:"status"      gorm:"column:status;not null"`
	FailMessage *string      `json:"failMessage" gorm:"column:fail_message"`
	IsSkipRetry bool         `json:"isSkipRetry" gorm:"column:is_skip_retry;type:boolean;not null"`

	BackupSizeMb float64 `json:"backupSizeMb" gorm:"column:backup_size_mb;default:0"`

	BackupDurationMs int64 `json:"backupDurationMs" gorm:"column:backup_duration_ms;default:0"`

	EncryptionSalt *string                         `json:"-"          gorm:"column:encryption_salt"`
	EncryptionIV   *string                         `json:"-"          gorm:"column:encryption_iv"`
	Encryption     backups_config.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}
