package backups_config

import (
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/period"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupConfig struct {
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey;not null"`

	IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`

	StorePeriod period.Period `json:"storePeriod" gorm:"column:store_period;type:text;not null"`

	BackupIntervalID uuid.UUID           `json:"backupIntervalId"         gorm:"column:backup_interval_id;type:uuid;not null"`
	BackupInterval   *intervals.Interval `json:"backupInterval,omitempty" gorm:"foreignKey:BackupIntervalID"`

	Storage   *storages.Storage `json:"storage"   gorm:"foreignKey:StorageID"`
	StorageID *uuid.UUID        `json:"storageId" gorm:"column:storage_id;type:uuid;"`

	SendNotificationsOn       []BackupNotificationType `json:"sendNotificationsOn" gorm:"-"`
	SendNotificationsOnString string                   `json:"-"                   gorm:"column:send_notifications_on;type:text;not null"`

	IsRetryIfFailed     bool `json:"isRetryIfFailed"     gorm:"column:is_retry_if_failed;type:boolean;not null"`
	MaxFailedTriesCount int  `json:"maxFailedTriesCount" gorm:"column:max_failed_tries_count;type:int;not null"`

	Encryption BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
}

func (h *BackupConfig) TableName() string {
	return "backup_configs"
}

func (b *BackupConfig) BeforeSave(tx *gorm.DB) error {
	// Convert SendNotificationsOn array to string
	if len(b.SendNotificationsOn) > 0 {
		notificationTypes := make([]string, len(b.SendNotificationsOn))

		for i, notificationType := range b.SendNotificationsOn {
			notificationTypes[i] = string(notificationType)
		}

		b.SendNotificationsOnString = strings.Join(notificationTypes, ",")
	} else {
		b.SendNotificationsOnString = ""
	}

	return nil
}

func (b *BackupConfig) AfterFind(tx *gorm.DB) error {
	// Convert SendNotificationsOnString to array
	if b.SendNotificationsOnString != "" {
		notificationTypes := strings.Split(b.SendNotificationsOnString, ",")
		b.SendNotificationsOn = make([]BackupNotificationType, len(notificationTypes))

		for i, notificationType := range notificationTypes {
			b.SendNotificationsOn[i] = BackupNotificationType(notificationType)
		}
	} else {
		b.SendNotificationsOn = []BackupNotificationType{}
	}

	return nil
}

func (b *BackupConfig) Validate() error {
	// Backup interval is required either as ID or as object
	if b.BackupIntervalID == uuid.Nil && b.BackupInterval == nil {
		return errors.New("backup interval is required")
	}

	if b.StorePeriod == "" {
		return errors.New("store period is required")
	}

	if b.IsRetryIfFailed && b.MaxFailedTriesCount <= 0 {
		return errors.New("max failed tries count must be greater than 0")
	}

	if b.Encryption != "" && b.Encryption != BackupEncryptionNone &&
		b.Encryption != BackupEncryptionEncrypted {
		return errors.New("encryption must be NONE or ENCRYPTED")
	}

	return nil
}

func (b *BackupConfig) Copy(newDatabaseID uuid.UUID) *BackupConfig {
	return &BackupConfig{
		DatabaseID:          newDatabaseID,
		IsBackupsEnabled:    b.IsBackupsEnabled,
		StorePeriod:         b.StorePeriod,
		BackupIntervalID:    uuid.Nil,
		BackupInterval:      b.BackupInterval.Copy(),
		StorageID:           b.StorageID,
		SendNotificationsOn: b.SendNotificationsOn,
		IsRetryIfFailed:     b.IsRetryIfFailed,
		MaxFailedTriesCount: b.MaxFailedTriesCount,
		Encryption:          b.Encryption,
	}
}
