package physical_dto

import (
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

type PhysicalBackupMetadata struct {
	BackupID         uuid.UUID                         `json:"backupId"`
	DatabaseID       uuid.UUID                         `json:"databaseId"`
	BackupType       physical_enums.PhysicalBackupType `json:"backupType"`
	SystemIdentifier uint64                            `json:"systemIdentifier"`
	PgVersion        int                               `json:"pgVersion"`
	TimelineID       int                               `json:"timelineId"`

	StartLSN string `json:"startLsn"`
	StopLSN  string `json:"stopLsn"`

	UncompressedSizeBytes int64 `json:"uncompressedSizeBytes"`
	CompressedSizeBytes   int64 `json:"compressedSizeBytes"`

	RootFullBackupID          *uuid.UUID `json:"rootFullBackupId,omitempty"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId,omitempty"`

	Encryption     backups_core_enums.BackupEncryption `json:"encryption"`
	EncryptionSalt string                              `json:"encryptionSalt,omitempty"`
	EncryptionIV   string                              `json:"encryptionIv,omitempty"`

	Compression physical_enums.PhysicalBackupCompression `json:"compression"`

	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt"`
}

type PhysicalWalSegmentMetadata struct {
	WalSegmentID uuid.UUID `json:"walSegmentId"`
	DatabaseID   uuid.UUID `json:"databaseId"`
	TimelineID   int       `json:"timelineId"`
	WalFilename  string    `json:"walFilename"`

	StartLSN string `json:"startLsn"`
	EndLSN   string `json:"endLsn"`

	CompressedSizeBytes int64 `json:"compressedSizeBytes"`

	Encryption     backups_core_enums.BackupEncryption `json:"encryption"`
	EncryptionSalt string                              `json:"encryptionSalt,omitempty"`
	EncryptionIV   string                              `json:"encryptionIv,omitempty"`

	ReceivedAt time.Time `json:"receivedAt"`
}

type PhysicalWalHistoryMetadata struct {
	WalHistoryFileID uuid.UUID `json:"walHistoryFileId"`
	DatabaseID       uuid.UUID `json:"databaseId"`
	TimelineID       int       `json:"timelineId"`
	HistoryFilename  string    `json:"historyFilename"`

	CompressedSizeBytes int64 `json:"compressedSizeBytes"`

	Encryption     backups_core_enums.BackupEncryption `json:"encryption"`
	EncryptionSalt string                              `json:"encryptionSalt,omitempty"`
	EncryptionIV   string                              `json:"encryptionIv,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// One interval of the WAL stream statistics series. Every value is nil when no
// sample falls into the interval, so the hole stays a hole instead of a zero.
type PhysicalWalStreamSeriesPoint struct {
	StartedAt time.Time `json:"startedAt" gorm:"column:started_at"`

	SourceWriteBytesPerSecond    *float64 `json:"sourceWriteBytesPerSecond"    gorm:"column:source_write_bytes_per_second"`
	ArchiveBytesPerSecond        *float64 `json:"archiveBytesPerSecond"        gorm:"column:archive_bytes_per_second"`
	ArchivedStoredBytesPerSecond *float64 `json:"archivedStoredBytesPerSecond" gorm:"column:archived_stored_bytes_per_second"`

	MaxSlotLagBytes       *int64 `json:"maxSlotLagBytes"       gorm:"column:max_slot_lag_bytes"`
	MaxUnarchivedWalBytes *int64 `json:"maxUnarchivedWalBytes" gorm:"column:max_unarchived_wal_bytes"`
	MaxLocalQueueBytes    *int64 `json:"maxLocalQueueBytes"    gorm:"column:max_local_queue_bytes"`
	LocalQueuePauseBytes  *int64 `json:"localQueuePauseBytes"  gorm:"column:local_queue_pause_bytes"`

	Restarts      *int64 `json:"restarts"      gorm:"column:restarts"`
	Pauses        *int64 `json:"pauses"        gorm:"column:pauses"`
	SessionStarts *int64 `json:"sessionStarts" gorm:"column:session_starts"`

	HighestConcurrentUploads *int64 `json:"highestConcurrentUploads" gorm:"column:highest_concurrent_uploads"`
	SegmentWriteAvgMillis    *int64 `json:"segmentWriteAvgMillis"    gorm:"column:segment_write_avg_millis"`
	SegmentWriteMaxMillis    *int64 `json:"segmentWriteMaxMillis"    gorm:"column:segment_write_max_millis"`
}
