package usecases_physical_postgresql

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

type CreateIncrementalBackupUsecase struct{}

func NewCreateIncrementalBackupUsecase() *CreateIncrementalBackupUsecase {
	return &CreateIncrementalBackupUsecase{}
}

func (uc *CreateIncrementalBackupUsecase) Execute(
	ctx context.Context,
	spec IncrementalBackupSpec,
) (PhysicalBackupResult, error) {
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

	refusalResult, canProceed := verifyIncrTimelineCompatibility(ctx, spec)
	if !canProceed {
		return refusalResult, nil
	}

	// Gate on WAL-summarizer readiness BEFORE any work: a doomed
	// pg_basebackup --incremental (summarizer off / summaries expired / falling
	// behind) is turned into a deterministic CHAIN_BROKEN here, so the next
	// scheduler tick re-anchors on a fresh FULL instead of looping transient
	// ERRORs. Runs before the parent-manifest download so a bail costs nothing.
	preCheckResult, canProceed := runSummarizerPreCheck(ctx, spec)
	if !canProceed {
		return preCheckResult, nil
	}

	// Manifest System-Identifier from the stored row (see create_full_uc.go Execute).
	systemID := spec.SourceDB.SystemIdentifierUint64()

	manifestPath, manifestCleanup, err := downloadParentManifest(spec)
	if err != nil {
		reason := physical_enums.PhysicalBackupErrorParentManifestMissing

		return PhysicalBackupResult{
			Status:       physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: fmt.Sprintf("download parent manifest: %v", err),
			CompletedAt:  time.Now().UTC(),
		}, nil
	}
	defer manifestCleanup()

	fileName := buildObjectName(spec.DatabaseName, spec.Backup.ID, start, "INCR")

	spec.Backup.FileName = &fileName

	if err := spec.IncrRepo.Save(spec.Backup); err != nil {
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
			systemID,
			manifestPath,
			classifyIncrStreamError,
		)
		if err != nil {
			result = errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed,
				"pg_basebackup --incremental stream", err)
			return nil
		}

		if streamResult.Status != physical_enums.PhysicalBackupStatusCompleted {
			result = streamResult
			return nil
		}

		streamResult.BackupDurationMs = time.Since(start).Milliseconds()
		streamResult.CompletedAt = time.Now().UTC()
		streamResult.FileName = fileName

		if err := uploadIncrMetadata(
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

// verifyIncrTimelineCompatibility gates an INCR on timeline/cluster
// compatibility. Unlike a FULL, an INCR cannot extend across a timeline switch —
// a detected failover is chain-killing, and the chain must re-anchor on a fresh
// FULL on the new TL.
func verifyIncrTimelineCompatibility(
	ctx context.Context,
	spec IncrementalBackupSpec,
) (PhysicalBackupResult, bool) {
	return verifyTimelineCompatibility(ctx, spec.CommonBackupSpec,
		func(decision *TimelineDecision) (PhysicalBackupResult, bool) {
			reason := physical_enums.PhysicalBackupErrorTimelineRegression

			return PhysicalBackupResult{
				Status:      physical_enums.PhysicalBackupStatusChainBroken,
				ErrorReason: &reason,
				ErrorMessage: fmt.Sprintf(
					"timeline switch detected (live TL %d > known TL): incremental refused, new FULL required",
					decision.NewTLI,
				),
				CompletedAt: time.Now().UTC(),
			}, false
		})
}

// runSummarizerPreCheck opens an inspection connection and resolves the
// WAL-summarizer decision (including the bounded wait for a lagging-but-catching-up
// summarizer). It returns proceed=true to run the incremental, or a terminal
// result the caller must return verbatim:
//   - DecisionGoIncremental         → proceed
//   - DecisionFullSameChain / wait timeout → CHAIN_BROKEN / SUMMARIZER_FALLING_BEHIND
//   - DecisionFullNewChain          → CHAIN_BROKEN / SUMMARIZER_OFF | SUMMARIES_EXPIRED
//   - ctx cancelled mid-wait        → CANCELED / CANCELED_BY_USER
func runSummarizerPreCheck(ctx context.Context, spec IncrementalBackupSpec) (PhysicalBackupResult, bool) {
	conn, err := spec.SourceDB.OpenInspectionConn(ctx, spec.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"open inspection conn for summarizer pre-check", err), false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	decision, err := resolveSummarizerDecision(ctx, conn, spec.ParentManifest.StopLSN, spec.IncrementalCadence)
	if err != nil {
		if ctx.Err() != nil {
			return canceledResult(physical_enums.PhysicalBackupErrorCanceledByUser,
				"incremental cancelled during summarizer wait"), false
		}

		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure, "summarizer pre-check", err), false
	}

	switch decision.Decision {
	case DecisionGoIncremental:
		return PhysicalBackupResult{}, true

	case DecisionFullSameChain:
		return summarizerChainBroken(physical_enums.PhysicalBackupErrorSummarizerFallingBehind,
			"summarizer trailing current WAL; closing chain, new FULL required"), false

	default: // DecisionFullNewChain — Reason is always set on this branch
		return summarizerChainBroken(*decision.Reason,
			"summarizer pre-check refused incremental; new FULL required"), false
	}
}

func summarizerChainBroken(
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) PhysicalBackupResult {
	return PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusChainBroken,
		ErrorReason:  &reason,
		ErrorMessage: message,
		CompletedAt:  time.Now().UTC(),
	}
}

func canceledResult(
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) PhysicalBackupResult {
	return PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusCanceled,
		ErrorReason:  &reason,
		ErrorMessage: message,
		CompletedAt:  time.Now().UTC(),
	}
}

func downloadParentManifest(
	spec IncrementalBackupSpec,
) (manifestPath string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "pgincr_"+uuid.New().String())
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp dir: %w", err)
	}

	cleanupAll := func() { _ = os.RemoveAll(tmpDir) }

	storageHandle, ok := spec.Storage.(parentManifestFetcher)
	if !ok {
		cleanupAll()

		return "", func() {}, errors.New("storage does not support GetFile")
	}

	reader, err := storageHandle.GetFile(spec.FieldEncryptor, spec.ParentManifest.FileName)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("fetch parent manifest: %w", err)
	}
	defer func() { _ = reader.Close() }()

	decoded := io.Reader(reader)

	if spec.ParentManifest.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		decoded, err = decryptParentManifest(reader, spec)
		if err != nil {
			cleanupAll()

			return "", func() {}, err
		}
	}

	manifestPath = filepath.Join(tmpDir, "backup_manifest")

	out, err := os.Create(manifestPath)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("create manifest temp file: %w", err)
	}

	if _, err := io.Copy(out, decoded); err != nil {
		_ = out.Close()
		cleanupAll()

		return "", func() {}, fmt.Errorf("copy manifest into temp file: %w", err)
	}

	if err := out.Close(); err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("close manifest temp file: %w", err)
	}

	return manifestPath, cleanupAll, nil
}

func decryptParentManifest(reader io.Reader, spec IncrementalBackupSpec) (io.Reader, error) {
	salt, err := base64.StdEncoding.DecodeString(spec.ParentManifest.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(spec.ParentManifest.IV)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest nonce: %w", err)
	}

	dec, err := backup_encryption.NewDecryptionReader(reader, spec.MasterKey, spec.ParentManifest.BackupID, salt, nonce)
	if err != nil {
		return nil, fmt.Errorf("init decryption reader: %w", err)
	}

	return dec, nil
}

func classifyIncrStreamError(streamErr error, stderr []byte) streamOutcome {
	if isSummariesExpiredError(stderr) {
		reason := physical_enums.PhysicalBackupErrorSummariesExpired

		return streamOutcome{
			Status:       physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
		}
	}

	reason := physical_enums.PhysicalBackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
	}
}

// isSummariesExpiredError detects the post-readiness-check race where the source cluster
// pruned WAL summaries between CheckSummarizerReadiness returning OK and pg_basebackup
// --incremental opening the actual range. PG surfaces this as "WAL summary file
// ... not found" or "could not open WAL summary".
func isSummariesExpiredError(stderr []byte) bool {
	msg := string(stderr)

	for _, needle := range []string{
		"WAL summary file",
		"could not open WAL summary",
		"WAL summary not found",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}

	return false
}
