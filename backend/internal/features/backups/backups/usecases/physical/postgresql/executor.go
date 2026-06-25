package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

// runStreamParams builds the codec-independent parameters runStream needs from
// the shared spec. backupID and systemID come from the variant-specific row and
// the live timeline check, so they are passed explicitly rather than read off the spec.
func (s CommonBackupSpec) runStreamParams(
	fileName string,
	backupID uuid.UUID,
	systemID uint64,
	codec physical_enums.PhysicalBackupCompression,
) runStreamParams {
	return runStreamParams{
		Storage:          s.Storage,
		FieldEncryptor:   s.FieldEncryptor,
		Logger:           s.Logger,
		FileName:         fileName,
		Encryption:       s.Encryption,
		MasterKey:        s.MasterKey,
		BackupID:         backupID,
		SystemID:         systemID,
		Codec:            codec,
		ProgressListener: s.ProgressListener,
	}
}

// streamWithCodecFallback owns the ZSTD -> GZIP -> NONE codec-fallback loop and
// the final mapping of the settled streamOutcome to a PhysicalBackupResult,
// shared by FULL and INCR. It runs INSIDE the per-backup replication slot (the
// caller invokes it from Execute's WithBackupSlot callback), so the slot is held
// across every attempt; only the --compress flag and the recorded codec differ
// between attempts, and every attempt streams to the same object key (a rejected
// attempt wrote 0 bytes, and SaveFile is overwrite-PUT).
//
// incrementalManifestPath is "" for a FULL; a non-empty path is the downloaded
// parent-manifest temp file that makes this an INCR (--incremental=<path>).
func streamWithCodecFallback(
	ctx context.Context,
	common CommonBackupSpec,
	backupID uuid.UUID,
	creds *postgresql_shared.CredentialTempFiles,
	fileName string,
	systemID uint64,
	incrementalManifestPath string,
	classify streamErrorClassifier,
) (PhysicalBackupResult, error) {
	pgBin := tools.GetPostgresqlExecutable(common.SourceDB.Version, tools.PostgresqlExecutablePgBasebackup)

	var settled streamOutcome

	for i, codec := range codecFallbackOrder {
		buildCmd := func(streamCtx context.Context) (*exec.Cmd, error) {
			return newPgBasebackupCommand(
				streamCtx,
				pgBin,
				common.SourceDB,
				creds,
				fileName,
				codec,
				incrementalManifestPath,
			)
		}

		outcome, err := runStream(
			ctx,
			common.runStreamParams(fileName, backupID, systemID, codec),
			buildCmd,
			classify,
		)
		if err != nil {
			return PhysicalBackupResult{}, err
		}

		if outcome.isCompressionUnsupported {
			if i+1 < len(codecFallbackOrder) {
				common.Logger.Warn(
					fmt.Sprintf("compression downgraded: %s -> %s", codec, codecFallbackOrder[i+1]),
					"backup_id", backupID)

				continue
			}

			settled = compressionExhaustedOutcome(outcome.Stderr)

			break
		}

		settled = outcome

		break
	}

	return resultFromOutcome(settled, fileName), nil
}

func resultFromOutcome(outcome streamOutcome, fileName string) PhysicalBackupResult {
	return PhysicalBackupResult{
		Status:                 outcome.Status,
		ErrorReason:            outcome.ErrorReason,
		ErrorMessage:           outcome.ErrorMessage,
		FileName:               fileName,
		TimelineID:             outcome.TimelineID,
		StartLSN:               outcome.StartLSN,
		StopLSN:                outcome.StopLSN,
		BackupSizeMb:           outcome.BackupSizeMb,
		EncryptionAlgo:         outcome.EncryptionAlgo,
		EncryptionSalt:         outcome.EncryptionSalt,
		EncryptionIV:           outcome.EncryptionIV,
		Compression:            outcome.Compression,
		ManifestFileName:       outcome.ManifestFileName,
		ManifestEncryptionSalt: outcome.ManifestEncryptionSalt,
		ManifestEncryptionIV:   outcome.ManifestEncryptionIV,
	}
}

// errorResult is the terminal result for a pre-stream failure (credential setup,
// timeline-check plumbing, slot lifecycle). stage names the step for the persisted
// message.
func errorResult(
	reason physical_enums.PhysicalBackupErrorReason,
	stage string,
	err error,
) PhysicalBackupResult {
	r := reason

	return PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &r,
		ErrorMessage: fmt.Sprintf("%s: %v", stage, err),
		CompletedAt:  time.Now().UTC(),
	}
}

// buildObjectName is the storage object key for a backup artifact:
// "<dbName>-<kind>-<timestamp>-<backupID>", kind being FULL or INCR. The name is
// sanitized for storage portability (same as logical backups); uniqueness comes
// from the trailing backupID. The codec is recorded on the row, never in the
// name, so the key is extension-less.
func buildObjectName(
	databaseName string,
	backupID uuid.UUID,
	now time.Time,
	kind string,
) string {
	return fmt.Sprintf("%s-%s-%s-%s",
		files_utils.SanitizeFilename(databaseName),
		kind,
		now.Format("20060102-150405"),
		backupID.String(),
	)
}

// verifyTimelineCompatibility gates a backup on the live timeline/cluster
// verdict. FailoverDetected is the one kind FULL and INCR treat differently — a
// FULL proceeds on the live TL while an INCR must re-anchor on a fresh FULL — so
// the caller supplies onFailover; a nil onFailover treats failover as
// non-breaking (FULL). canProceed=false means the caller returns refusal and stops.
func verifyTimelineCompatibility(
	ctx context.Context,
	common CommonBackupSpec,
	onFailover func(decision *TimelineDecision) (refusal PhysicalBackupResult, canProceed bool),
) (refusal PhysicalBackupResult, canProceed bool) {
	conn, err := common.SourceDB.OpenInspectionConn(ctx, common.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"open inspection connection", err), false
	}
	defer func() { _ = conn.Close(ctx) }()

	decision, err := CheckTimelineCompatibility(ctx, conn, common.SourceDB, common.FullRepo, common.HistoryRepo)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"timeline compatibility check", err), false
	}

	switch decision.Kind {
	case TimelineContinue:
		return PhysicalBackupResult{}, true

	case TimelineFailoverDetected:
		if onFailover == nil {
			return PhysicalBackupResult{}, true
		}

		return onFailover(decision)

	case TimelineRegression:
		reason := physical_enums.PhysicalBackupErrorTimelineRegression

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline regression: expected TL %d, live TL %d",
				decision.ExpectedTLI, decision.ActualTLI,
			),
			CompletedAt: time.Now().UTC(),
		}, false

	case TimelineDifferentCluster:
		reason := physical_enums.PhysicalBackupErrorSystemIdentifierMismatch

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"system_identifier mismatch: catalog %s, live %s",
				decision.ExpectedSysID, decision.ActualSysID,
			),
			CompletedAt: time.Now().UTC(),
		}, false
	}

	return PhysicalBackupResult{}, true
}
