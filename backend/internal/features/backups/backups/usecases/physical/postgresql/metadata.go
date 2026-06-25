package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const metadataUploadTimeout = 30 * time.Second

// uploadFullMetadata writes the `<artifact>.metadata` sidecar describing a
// completed FULL, next to the artifact / manifest / history the stream already
// uploaded. Living in the executor keeps the backuper out of the upload path and
// makes the sidecar atomic with the COMPLETED result.
func uploadFullMetadata(
	logger *slog.Logger,
	fieldEncryptor util_encryption.FieldEncryptor,
	storage storages.StorageFileSaver,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	fullBackup *physical_models.PhysicalFullBackup,
	result PhysicalBackupResult,
) error {
	if result.FileName == "" {
		return errors.New("cannot upload metadata: file_name is empty")
	}

	metadata := physical_dto.PhysicalBackupMetadata{
		BackupID:              fullBackup.ID,
		DatabaseID:            fullBackup.DatabaseID,
		BackupType:            physical_enums.PhysicalBackupTypeFull,
		SystemIdentifier:      sourceDB.SystemIdentifierUint64(),
		PgVersion:             pgVersionFromTag(sourceDB.Version),
		TimelineID:            result.TimelineID,
		StartLSN:              result.StartLSN.String(),
		StopLSN:               result.StopLSN.String(),
		UncompressedSizeBytes: 0,
		CompressedSizeBytes:   int64(result.BackupSizeMb * 1024 * 1024),
		Encryption:            result.EncryptionAlgo,
		EncryptionSalt:        result.EncryptionSalt,
		EncryptionIV:          result.EncryptionIV,
		Compression:           result.Compression,
		CreatedAt:             fullBackup.CreatedAt,
		CompletedAt:           result.CompletedAt,
	}

	return uploadMetadata(logger, fieldEncryptor, storage, result.FileName, metadata)
}

// uploadIncrMetadata writes the `<artifact>.metadata` sidecar for a completed
// INCR, carrying the chain references (root full + parent incremental) restore
// needs to walk the chain.
func uploadIncrMetadata(
	logger *slog.Logger,
	fieldEncryptor util_encryption.FieldEncryptor,
	storage storages.StorageFileSaver,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	result PhysicalBackupResult,
) error {
	if result.FileName == "" {
		return errors.New("cannot upload metadata: file_name is empty")
	}

	rootID := incrBackup.RootFullBackupID

	metadata := physical_dto.PhysicalBackupMetadata{
		BackupID:                  incrBackup.ID,
		DatabaseID:                incrBackup.DatabaseID,
		BackupType:                physical_enums.PhysicalBackupTypeIncremental,
		SystemIdentifier:          sourceDB.SystemIdentifierUint64(),
		PgVersion:                 pgVersionFromTag(sourceDB.Version),
		TimelineID:                result.TimelineID,
		StartLSN:                  result.StartLSN.String(),
		StopLSN:                   result.StopLSN.String(),
		UncompressedSizeBytes:     0,
		CompressedSizeBytes:       int64(result.BackupSizeMb * 1024 * 1024),
		RootFullBackupID:          &rootID,
		ParentIncrementalBackupID: incrBackup.ParentIncrementalBackupID,
		Encryption:                result.EncryptionAlgo,
		EncryptionSalt:            result.EncryptionSalt,
		EncryptionIV:              result.EncryptionIV,
		Compression:               result.Compression,
		CreatedAt:                 incrBackup.CreatedAt,
		CompletedAt:               result.CompletedAt,
	}

	return uploadMetadata(logger, fieldEncryptor, storage, result.FileName, metadata)
}

// uploadMetadata marshals the metadata and PUTs `<artifact>.metadata`. It uses a
// fresh Background context with its own timeout: the stream is already done, so a
// cancelled backup context must not abort the sidecar write.
func uploadMetadata(
	logger *slog.Logger,
	fieldEncryptor util_encryption.FieldEncryptor,
	storage storages.StorageFileSaver,
	artifactFileName string,
	metadata physical_dto.PhysicalBackupMetadata,
) error {
	body, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata JSON: %w", err)
	}

	metadataName := artifactFileName + metadataSuffix

	ctx, cancel := context.WithTimeout(context.Background(), metadataUploadTimeout)
	defer cancel()

	if err := storage.SaveFile(ctx, fieldEncryptor, logger, metadataName, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("upload metadata: %w", err)
	}

	return nil
}

// pgVersionFromTag converts the tools.PostgresqlVersion enum ("17", "18") into
// the canonical server_version_num style (170000, 180000) so the metadata
// carries a value pg_combinebackup can compare against on restore. The exact
// patch level is unknown at this layer — major.minor is enough for the check.
func pgVersionFromTag(v tools.PostgresqlVersion) int {
	major, err := strconv.Atoi(strings.TrimSpace(string(v)))
	if err != nil {
		return 0
	}

	return major * 10000
}
