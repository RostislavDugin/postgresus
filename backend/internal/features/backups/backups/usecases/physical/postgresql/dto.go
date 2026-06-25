package usecases_physical_postgresql

import (
	"log/slog"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

type CommonBackupSpec struct {
	SourceDB       *postgresql_physical.PostgresqlPhysicalDatabase
	DatabaseName   string
	StorageID      uuid.UUID
	Storage        storages.StorageFileSaver
	Encryption     backups_core_enums.BackupEncryption
	MasterKey      string
	FieldEncryptor util_encryption.FieldEncryptor
	FullRepo       *physical_repositories.PhysicalFullBackupRepository
	HistoryRepo    *physical_repositories.PhysicalWalHistoryRepository
	Logger         *slog.Logger

	// ProgressListener, when set, receives the streamed size and elapsed time
	// periodically during the backup so the row's size grows in near-real-time.
	// nil disables progress reporting.
	ProgressListener func(completedMb float64, elapsedMs int64)
}

type FullBackupSpec struct {
	CommonBackupSpec
	Backup *physical_models.PhysicalFullBackup
}

type IncrementalBackupSpec struct {
	CommonBackupSpec

	Backup         *physical_models.PhysicalIncrementalBackup
	ParentManifest ParentManifestRef
	IncrRepo       *physical_repositories.PhysicalIncrementalBackupRepository

	// IncrementalCadence is the configured INCR interval, used only to size the
	// bounded summarizer wait window (min(cadence/4, cap)). Zero is acceptable —
	// the executor then waits only up to its own cap.
	IncrementalCadence time.Duration
}

type PhysicalBackupResult struct {
	Status       physical_enums.PhysicalBackupStatus
	ErrorReason  *physical_enums.PhysicalBackupErrorReason
	ErrorMessage string

	FileName string

	TimelineID int
	StartLSN   walmath.LSN
	StopLSN    walmath.LSN

	BackupSizeMb     float64
	BackupDurationMs int64

	EncryptionAlgo backups_core_enums.BackupEncryption
	EncryptionSalt string
	EncryptionIV   string

	Compression physical_enums.PhysicalBackupCompression

	ManifestFileName       string
	ManifestEncryptionSalt string
	ManifestEncryptionIV   string

	CompletedAt time.Time
}

// ParentManifestRef is everything an incremental needs about its parent: the
// reconstructed manifest sidecar to fetch+decrypt, plus the parent's stop_lsn
// (the lower bound the WAL summarizer must cover before this INCR can run).
type ParentManifestRef struct {
	BackupID   uuid.UUID
	FileName   string
	Encryption backups_core_enums.BackupEncryption
	Salt       string
	IV         string
	StopLSN    walmath.LSN
}
