package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	chain_view "databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
)

type CreateFullBackupUsecase struct{}

func NewCreateFullBackupUsecase() *CreateFullBackupUsecase { return &CreateFullBackupUsecase{} }

func (uc *CreateFullBackupUsecase) Execute(ctx context.Context, spec FullBackupSpec) (PhysicalBackupResult, error) {
	start := time.Now().UTC()

	password, err := postgresql_shared.DecryptFieldIfNeeded(spec.SourceDB.Password, spec.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed, "decrypt password", err), nil
	}

	creds, err := postgresql_shared.WriteCredentialFilesToTempDir(
		spec.SourceDB.CredentialSpec(), password, spec.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed, "write credentials", err), nil
	}
	defer creds.Remove()

	// nil onFailover: a detected failover does not refuse a FULL — it proceeds on
	// the live TL, and the scheduler observes the bumped TL across two FULL rows.
	refusalResult, canProceed := verifyTimelineCompatibility(ctx, spec.CommonBackupSpec, nil)
	if !canProceed {
		return refusalResult, nil
	}

	fileName := buildObjectName(spec.DatabaseName, spec.Backup.ID, start, "FULL")

	spec.Backup.FileName = &fileName
	if err := spec.FullRepo.Save(spec.Backup); err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorStorageUploadFailed,
			"persist file_name at upload-start", err), nil
	}

	var result PhysicalBackupResult

	slotErr := WithBackupSlot(ctx, spec.SourceDB, spec.FieldEncryptor, spec.Logger, func() error {
		streamResult, err := streamWithCodecFallback(
			ctx,
			spec.CommonBackupSpec,
			spec.Backup.ID,
			creds,
			fileName,
			spec.SourceDB.SystemIdentifierUint64(),
			"",
			classifyFullStreamError,
		)
		if err != nil {
			result = errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed,
				"pg_basebackup stream", err)
			return nil
		}

		if streamResult.Status != physical_enums.PhysicalBackupStatusCompleted {
			result = streamResult
			return nil
		}

		validation, valErr := ValidateStartLsnAgainstHistory(
			spec.SourceDB.ParentDatabaseID(),
			streamResult.TimelineID,
			streamResult.StartLSN,
			spec.HistoryRepo,
		)
		if valErr != nil {
			spec.Logger.Warn("start-LSN history validation failed",
				"error", valErr)
		}

		if validation.Status == chain_view.ValidationStatusChainBroken {
			removeUploadedArtifactsAfterChainBroken(spec.Storage, spec.FieldEncryptor, fileName, spec.Logger)

			reason := physical_enums.PhysicalBackupErrorStartLsnOutsideTimeline

			result = PhysicalBackupResult{
				Status:       physical_enums.PhysicalBackupStatusChainBroken,
				ErrorReason:  &reason,
				ErrorMessage: validation.Message,
				FileName:     fileName,
				TimelineID:   streamResult.TimelineID,
				StartLSN:     streamResult.StartLSN,
				StopLSN:      streamResult.StopLSN,
				CompletedAt:  time.Now().UTC(),
			}

			return nil
		}

		if validation.Status == chain_view.ValidationStatusOKWithWarning {
			spec.Logger.Info(validation.Message,
				"timeline_id", streamResult.TimelineID)
		}

		if streamResult.TimelineID > 1 {
			uploadHistoryForTimelineSwitch(ctx, spec.CommonBackupSpec, streamResult.TimelineID)
		}

		streamResult.BackupDurationMs = time.Since(start).Milliseconds()
		streamResult.CompletedAt = time.Now().UTC()
		streamResult.FileName = fileName

		if err := uploadFullMetadata(
			spec.Logger, spec.FieldEncryptor, spec.Storage, spec.SourceDB, spec.Backup, streamResult,
		); err != nil {
			result = errorResult(physical_enums.PhysicalBackupErrorStorageUploadFailed, "upload metadata", err)
			return nil
		}

		result = streamResult

		return nil
	})

	if slotErr != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"per-backup slot lifecycle", slotErr), nil
	}

	return result, nil
}

// uploadHistoryForTimelineSwitch best-effort uploads the .history file for a FULL
// that ran on a timeline > 1. A failure here leaves the FULL COMPLETED: the
// artifact itself is valid, and a missing .history only degrades later chain
// validation, which the scheduler tolerates.
func uploadHistoryForTimelineSwitch(ctx context.Context, common CommonBackupSpec, timelineID int) {
	historyConn, err := common.SourceDB.OpenInspectionConn(ctx, common.FieldEncryptor)
	if err != nil {
		common.Logger.Warn("could not open connection for history upload; FULL stays COMPLETED",
			"timeline_id", timelineID,
			"error", err)

		return
	}
	defer func() { _ = historyConn.Close(ctx) }()

	if _, err := UploadHistoryFile(
		ctx,
		historyConn,
		timelineID,
		common.Storage,
		common.SourceDB,
		common.StorageID,
		common.HistoryRepo,
		common.Encryption,
		common.MasterKey,
		common.FieldEncryptor,
		common.Logger,
	); err != nil {
		common.Logger.Warn("history upload failed; FULL stays COMPLETED",
			"timeline_id", timelineID,
			"error", err)
	}
}

func classifyFullStreamError(streamErr error, stderr []byte) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
	}
}

// removeUploadedArtifactsAfterChainBroken deletes the streamed artifact and its
// reconstructed-manifest sidecar after a post-stream CHAIN_BROKEN verdict, so a
// rejected FULL leaves nothing dangling in storage. The .metadata sidecar is not
// touched here: it is written only after this point (see uploadFullMetadata), so
// it does not yet exist. DeleteFile is idempotent on not-found.
func removeUploadedArtifactsAfterChainBroken(
	storage storages.StorageFileSaver,
	encryptor util_encryption.FieldEncryptor,
	fileName string,
	logger *slog.Logger,
) {
	manifestName := fileName + manifestSuffix

	if err := storage.DeleteFile(encryptor, manifestName); err != nil {
		logger.Warn("failed to remove manifest after CHAIN_BROKEN",
			"file_name", manifestName,
			"error", err)
	}

	if err := storage.DeleteFile(encryptor, fileName); err != nil {
		logger.Warn("failed to remove artifact after CHAIN_BROKEN",
			"file_name", fileName,
			"error", err)
	}
}
