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
