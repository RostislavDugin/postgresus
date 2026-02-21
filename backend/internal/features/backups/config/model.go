package backups_config

import (
	"databasus-backend/internal/config"
	"databasus-backend/internal/features/intervals"
	plans "databasus-backend/internal/features/plan"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/period"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupConfig struct {
	ID         uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`

	// Human-readable name for the schedule
	Name string `json:"name" gorm:"column:name;type:text;not null"`

	IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`

	RetentionPolicyType RetentionPolicyType `json:"retentionPolicyType" gorm:"column:retention_policy_type;type:text;not null;default:'TIME_PERIOD'"`
	RetentionTimePeriod period.TimePeriod   `json:"retentionTimePeriod" gorm:"column:retention_time_period;type:text;not null;default:''"`

	RetentionCount     int `json:"retentionCount"     gorm:"column:retention_count;type:int;not null;default:0"`
	RetentionGfsHours  int `json:"retentionGfsHours"  gorm:"column:retention_gfs_hours;type:int;not null;default:0"`
	RetentionGfsDays   int `json:"retentionGfsDays"   gorm:"column:retention_gfs_days;type:int;not null;default:0"`
	RetentionGfsWeeks  int `json:"retentionGfsWeeks"  gorm:"column:retention_gfs_weeks;type:int;not null;default:0"`
	RetentionGfsMonths int `json:"retentionGfsMonths" gorm:"column:retention_gfs_months;type:int;not null;default:0"`
	RetentionGfsYears  int `json:"retentionGfsYears"  gorm:"column:retention_gfs_years;type:int;not null;default:0"`

	BackupIntervalID uuid.UUID           `json:"backupIntervalId"         gorm:"column:backup_interval_id;type:uuid;not null"`
	BackupInterval   *intervals.Interval `json:"backupInterval,omitempty" gorm:"foreignKey:BackupIntervalID"`

	Storage   *storages.Storage `json:"storage"   gorm:"foreignKey:StorageID"`
	StorageID *uuid.UUID        `json:"storageId" gorm:"column:storage_id;type:uuid;"`

	SendNotificationsOn       []BackupNotificationType `json:"sendNotificationsOn" gorm:"-"`
	SendNotificationsOnString string                   `json:"-"                   gorm:"column:send_notifications_on;type:text;not null"`

	IsRetryIfFailed     bool `json:"isRetryIfFailed"     gorm:"column:is_retry_if_failed;type:boolean;not null"`
	MaxFailedTriesCount int  `json:"maxFailedTriesCount" gorm:"column:max_failed_tries_count;type:int;not null"`

	Encryption BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	// MaxBackupSizeMB limits individual backup size. 0 = unlimited.
	MaxBackupSizeMB int64 `json:"maxBackupSizeMb"       gorm:"column:max_backup_size_mb;type:int;not null"`
	// MaxBackupsTotalSizeMB limits total size of all backups. 0 = unlimited.
	MaxBackupsTotalSizeMB int64 `json:"maxBackupsTotalSizeMb" gorm:"column:max_backups_total_size_mb;type:int;not null"`
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

func (b *BackupConfig) Validate(plan *plans.DatabasePlan) error {
	if b.Name == "" {
		return errors.New("name is required")
	}

	if b.BackupIntervalID == uuid.Nil && b.BackupInterval == nil {
		return errors.New("backup interval is required")
	}

	if err := b.validateRetentionPolicy(plan); err != nil {
		return err
	}

	if b.IsRetryIfFailed && b.MaxFailedTriesCount <= 0 {
		return errors.New("max failed tries count must be greater than 0")
	}

	if b.Encryption != "" && b.Encryption != BackupEncryptionNone &&
		b.Encryption != BackupEncryptionEncrypted {
		return errors.New("encryption must be NONE or ENCRYPTED")
	}

	if config.GetEnv().IsCloud {
		if b.Encryption != BackupEncryptionEncrypted {
			return errors.New("encryption is mandatory for cloud storage")
		}
	}

	if b.MaxBackupSizeMB < 0 {
		return errors.New("max backup size must be non-negative")
	}

	if b.MaxBackupsTotalSizeMB < 0 {
		return errors.New("max backups total size must be non-negative")
	}

	if plan.MaxBackupSizeMB > 0 {
		if b.MaxBackupSizeMB == 0 || b.MaxBackupSizeMB > plan.MaxBackupSizeMB {
			return errors.New("max backup size exceeds plan limit")
		}
	}

	if plan.MaxBackupsTotalSizeMB > 0 {
		if b.MaxBackupsTotalSizeMB == 0 ||
			b.MaxBackupsTotalSizeMB > plan.MaxBackupsTotalSizeMB {
			return errors.New("max total backups size exceeds plan limit")
		}
	}

	return nil
}

func (b *BackupConfig) Copy(newDatabaseID uuid.UUID) *BackupConfig {
	return &BackupConfig{
		DatabaseID:            newDatabaseID,
		Name:                  b.Name,
		IsBackupsEnabled:      b.IsBackupsEnabled,
		RetentionPolicyType:   b.RetentionPolicyType,
		RetentionTimePeriod:   b.RetentionTimePeriod,
		RetentionCount:        b.RetentionCount,
		RetentionGfsHours:     b.RetentionGfsHours,
		RetentionGfsDays:      b.RetentionGfsDays,
		RetentionGfsWeeks:     b.RetentionGfsWeeks,
		RetentionGfsMonths:    b.RetentionGfsMonths,
		RetentionGfsYears:     b.RetentionGfsYears,
		BackupIntervalID:      uuid.Nil,
		BackupInterval:        b.BackupInterval.Copy(),
		StorageID:             b.StorageID,
		SendNotificationsOn:   b.SendNotificationsOn,
		IsRetryIfFailed:       b.IsRetryIfFailed,
		MaxFailedTriesCount:   b.MaxFailedTriesCount,
		Encryption:            b.Encryption,
		MaxBackupSizeMB:       b.MaxBackupSizeMB,
		MaxBackupsTotalSizeMB: b.MaxBackupsTotalSizeMB,
	}
}

func (b *BackupConfig) validateRetentionPolicy(plan *plans.DatabasePlan) error {
	switch b.RetentionPolicyType {
	case RetentionPolicyTypeTimePeriod, "":
		if b.RetentionTimePeriod == "" {
			return errors.New("retention time period is required")
		}

		if plan.MaxStoragePeriod != period.PeriodForever {
			if b.RetentionTimePeriod.CompareTo(plan.MaxStoragePeriod) > 0 {
				return errors.New("storage period exceeds plan limit")
			}
		}

	case RetentionPolicyTypeCount:
		if b.RetentionCount <= 0 {
			return errors.New("retention count must be greater than 0")
		}

	case RetentionPolicyTypeGFS:
		if b.RetentionGfsHours <= 0 && b.RetentionGfsDays <= 0 && b.RetentionGfsWeeks <= 0 &&
			b.RetentionGfsMonths <= 0 && b.RetentionGfsYears <= 0 {
			return errors.New("at least one GFS retention field must be greater than 0")
		}

	default:
		return errors.New("invalid retention policy type")
	}

	return nil
}
