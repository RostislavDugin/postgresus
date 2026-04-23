package backups_core

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	backups_config "databasus-backend/internal/features/backups/config"
	files_utils "databasus-backend/internal/util/files"
)

// TableHealthReport is stored as JSON in the backup row.
type TableHealthReport json.RawMessage

type PgWalBackupType string

const (
	PgWalBackupTypeFullBackup PgWalBackupType = "PG_FULL_BACKUP"
	PgWalBackupTypeWalSegment PgWalBackupType = "PG_WAL_SEGMENT"
)

type Backup struct {
	ID       uuid.UUID `json:"id"       gorm:"column:id;type:uuid;primaryKey"`
	FileName string    `json:"fileName" gorm:"column:file_name;type:text;not null"`

	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	Status      BackupStatus `json:"status"      gorm:"column:status;not null"`
	FailMessage *string      `json:"failMessage" gorm:"column:fail_message"`
	IsSkipRetry bool         `json:"isSkipRetry" gorm:"column:is_skip_retry;type:boolean;not null"`

	BackupSizeMb float64 `json:"backupSizeMb" gorm:"column:backup_size_mb;default:0"`

	BackupDurationMs int64 `json:"backupDurationMs" gorm:"column:backup_duration_ms;default:0"`

	EncryptionSalt *string                         `json:"-"          gorm:"column:encryption_salt"`
	EncryptionIV   *string                         `json:"-"          gorm:"column:encryption_iv"`
	Encryption     backups_config.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	// Postgres WAL backup specific fields
	PgWalBackupType                 *PgWalBackupType `json:"pgWalBackupType"                 gorm:"column:pg_wal_backup_type;type:text"`
	PgFullBackupWalStartSegmentName *string          `json:"pgFullBackupWalStartSegmentName" gorm:"column:pg_wal_start_segment;type:text"`
	PgFullBackupWalStopSegmentName  *string          `json:"pgFullBackupWalStopSegmentName"  gorm:"column:pg_wal_stop_segment;type:text"`
	PgVersion                       *string          `json:"pgVersion"                       gorm:"column:pg_version;type:text"`
	PgWalSegmentName                *string          `json:"pgWalSegmentName"                gorm:"column:pg_wal_segment_name;type:text"`

	TableHealthReportJSON *json.RawMessage `json:"tableHealthReport" gorm:"column:table_health_report;type:jsonb"`

	// Transient fields (not persisted to DB, used during backup execution)
	HealthReport any    `json:"-" gorm:"-"`
	StderrOutput string `json:"-" gorm:"-"`

	UploadCompletedAt *time.Time `json:"uploadCompletedAt" gorm:"column:upload_completed_at"`
	CreatedAt         time.Time  `json:"createdAt"         gorm:"column:created_at"`
}

func (b *Backup) GenerateFilename(dbName string) {
	timestamp := time.Now().UTC()

	b.FileName = fmt.Sprintf(
		"%s-%s-%s",
		files_utils.SanitizeFilename(dbName),
		timestamp.Format("20060102-150405"),
		b.ID.String(),
	)
}
