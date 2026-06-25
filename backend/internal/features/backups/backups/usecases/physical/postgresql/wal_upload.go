package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// walSegmentUploadTimeout bounds a single WAL-segment SaveFile. A segment is one
// wal_segment_size (16 MB default), so this is a short bounded upload, not a
// multi-hour stream like a FULL.
const walSegmentUploadTimeout = 2 * time.Minute

// WalUploadDeps carries everything the WAL uploader needs. WalSegmentSizeBytes is
// the source cluster's wal_segment_size (captured on the DB row), used to derive
// per-segment LSN bounds WITHOUT the walmath.WalSegmentSize package global — so a
// cluster with a non-default segsize is parsed correctly even when several DBs
// stream concurrently in one process.
type WalUploadDeps struct {
	DatabaseID          uuid.UUID
	StorageID           uuid.UUID
	Storage             storages.StorageFileSaver
	Encryption          backups_core_enums.BackupEncryption
	MasterKey           string
	FieldEncryptor      util_encryption.FieldEncryptor
	WalSegmentRepo      *physical_repositories.PhysicalWalSegmentRepository
	WalSegmentSizeBytes int64
	Logger              *slog.Logger

	// OnGapDetected fires at most once per newly-observed WAL gap (de-duplicated
	// by lastNotifiedGapEnd inside the uploader). nil disables notification.
	OnGapDetected func(gapStart, gapEnd walmath.LSN)
}

// WalUploader runs the insert-first claim -> upload -> commit -> gap-probe flow
// for fully-rotated WAL segments. It is the integrity core of the streamer and is
// unit-testable independent of pg_receivewal: hand it a deps bundle and a path to
// a finalized segment file.
type WalUploader struct {
	deps WalUploadDeps

	mu                 sync.Mutex
	lastNotifiedGapEnd walmath.LSN
}

func NewWalUploader(deps WalUploadDeps) *WalUploader {
	return &WalUploader{deps: deps}
}

// ProcessSegment runs the full insert-first claim flow for one finalized WAL
// segment file. localPath must point at a finalized segment (the caller filters
// out *.partial via walmath.IsWalFilename). It is safe to call repeatedly for the
// same segment: the claim/commit guards make re-processing idempotent.
func (u *WalUploader) ProcessSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	claim := &physical_models.PhysicalWalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  time.Now().UTC(),
		ClaimedAt:   time.Now().UTC(),
	}

	inserted, err := u.deps.WalSegmentRepo.ClaimInsert(claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if !inserted {
		return u.handleClaimConflict(localPath, claim)
	}

	return u.uploadAndCommit(ctx, localPath, claim)
}

// RecoverSegment runs the insert-first flow for a leftover finalized segment
// found in watch_dir at supervisor startup (crash recovery). It differs from
// ProcessSegment in exactly one branch: a pre-existing claim with file_name NULL
// is taken over and uploaded rather than left in place. Under the
// single-supervisor-per-DB invariant such a claim is always THIS database's own
// pre-crash claim (no concurrent uploader can exist), so finishing it is safe and
// avoids leaving the segment stuck NULL until the cleaner's 1h grace sweep.
func (u *WalUploader) RecoverSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	claim := &physical_models.PhysicalWalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  time.Now().UTC(),
		ClaimedAt:   time.Now().UTC(),
	}

	inserted, err := u.deps.WalSegmentRepo.ClaimInsert(claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if inserted {
		return u.uploadAndCommit(ctx, localPath, claim)
	}

	existing, err := u.deps.WalSegmentRepo.FindByChainKey(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	if existing == nil {
		return nil
	}

	if existing.FileName != nil {
		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	// Own pre-crash NULL claim: take it over by uploading against the existing row.
	return u.uploadAndCommit(ctx, localPath, existing)
}

// handleClaimConflict resolves a lost ClaimInsert race by probing the existing
// row: a durably-completed segment (file_name NOT NULL) means our local copy is
// redundant; an in-flight claim (file_name NULL) means a live or
// not-yet-reaped owner holds it, so we leave the local file untouched.
func (u *WalUploader) handleClaimConflict(localPath string, claim *physical_models.PhysicalWalSegment) error {
	existing, err := u.deps.WalSegmentRepo.FindByChainKey(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	if existing == nil {
		// Row vanished between the conflict and the probe — leave the local file;
		// the next uploader tick re-claims.
		return nil
	}

	if existing.FileName != nil {
		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	// In-flight claim: do not touch storage, leave the local file for a later tick
	// (the cleaner reaps the stale NULL claim after its grace period).
	return nil
}

// uploadAndCommit uploads the artifact + sidecar, then flips file_name to non-NULL.
// The commit is guarded by file_name IS NULL, so a DeleteFull cascade that removed
// the claim mid-upload yields committed=false and the orphaned ciphertext is
// deleted. On any pre-commit failure the claim row is released so the next tick
// re-claims from the retained local file.
func (u *WalUploader) uploadAndCommit(
	ctx context.Context,
	localPath string,
	claim *physical_models.PhysicalWalSegment,
) error {
	objectName := walSegmentObjectName(claim.DatabaseID, claim.TimelineID, claim.WalFilename)

	artifact, salt, iv, err := buildWalSegmentArtifactReader(localPath, u.deps.Encryption, u.deps.MasterKey, claim.ID)
	if err != nil {
		u.releaseClaim(claim.ID)

		return fmt.Errorf("build wal segment artifact: %w", err)
	}

	saveCtx, cancel := context.WithTimeout(ctx, walSegmentUploadTimeout)
	defer cancel()

	if err := u.deps.Storage.SaveFile(saveCtx, u.deps.FieldEncryptor, u.deps.Logger, objectName, artifact); err != nil {
		artifact.abort(err)
		_, _ = artifact.wait()
		u.deleteObject(objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("upload wal segment: %w", err)
	}

	compressedSizeBytes, err := artifact.wait()
	if err != nil {
		u.deleteObject(objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("stream wal segment artifact: %w", err)
	}

	if err := u.uploadSidecar(saveCtx, claim, salt, iv, compressedSizeBytes); err != nil {
		u.deleteObject(objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("upload wal segment sidecar: %w", err)
	}

	committed, err := u.deps.WalSegmentRepo.MarkUploaded(
		claim.ID, objectName, float64(compressedSizeBytes)/(1024*1024), nilIfEmpty(salt), nilIfEmpty(iv),
	)
	if err != nil {
		return fmt.Errorf("commit wal segment: %w", err)
	}

	if !committed {
		// DeleteFull cascade caught the claim mid-upload: the bytes are an orphan
		// the cleaner can no longer see (its row is gone), so delete them here.
		u.deleteObject(objectName)
		u.deleteObject(objectName + metadataSuffix)
		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	u.probeChainGap(claim)
	u.removeLocal(localPath, claim.WalFilename)

	return nil
}

// probeChainGap emits one notification when the just-committed segment is not
// LSN-contiguous with the previous committed segment on its timeline. It is a
// read-only check (no catalog row is written for the gap) de-duplicated by
// lastNotifiedGapEnd so a backlog drained after a storage outage notifies once.
func (u *WalUploader) probeChainGap(claim *physical_models.PhysicalWalSegment) {
	prevEnd, err := u.deps.WalSegmentRepo.FindLatestCommittedBefore(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		u.deps.Logger.Error("wal gap probe query failed", "wal_filename", claim.WalFilename, "error", err)

		return
	}

	if prevEnd == nil || *prevEnd == claim.StartLSN {
		return
	}

	u.mu.Lock()
	alreadyNotified := claim.StartLSN <= u.lastNotifiedGapEnd
	if !alreadyNotified {
		u.lastNotifiedGapEnd = claim.StartLSN
	}
	u.mu.Unlock()

	// Log and notify once per outage: a drained backlog re-enters this path for
	// the first segment past the gap only (later segments are contiguous and
	// returned above), and the lastNotifiedGapEnd guard covers re-processing.
	if alreadyNotified {
		return
	}

	u.deps.Logger.Warn(
		fmt.Sprintf("wal_gap_detected_post_upload: gap [%s, %s)", prevEnd.String(), claim.StartLSN.String()),
		"database_id", claim.DatabaseID,
		"timeline_id", claim.TimelineID,
	)

	if u.deps.OnGapDetected == nil {
		return
	}

	u.deps.OnGapDetected(*prevEnd, claim.StartLSN)
}

// uploadSidecar writes the `<object>.metadata` JSON describing the segment so a
// fresh Databasus can rebuild the catalog row from storage alone (import-from-DR).
func (u *WalUploader) uploadSidecar(
	ctx context.Context,
	claim *physical_models.PhysicalWalSegment,
	salt, iv string,
	compressedSizeBytes int64,
) error {
	sidecar := physical_dto.PhysicalWalSegmentMetadata{
		WalSegmentID:        claim.ID,
		DatabaseID:          claim.DatabaseID,
		TimelineID:          claim.TimelineID,
		WalFilename:         claim.WalFilename,
		StartLSN:            claim.StartLSN.String(),
		EndLSN:              claim.EndLSN.String(),
		CompressedSizeBytes: compressedSizeBytes,
		Encryption:          u.deps.Encryption,
		EncryptionSalt:      salt,
		EncryptionIV:        iv,
		ReceivedAt:          claim.ReceivedAt,
	}

	body, err := json.Marshal(sidecar)
	if err != nil {
		return fmt.Errorf("marshal wal segment sidecar: %w", err)
	}

	objectName := walSegmentObjectName(claim.DatabaseID, claim.TimelineID, claim.WalFilename) + metadataSuffix

	return u.deps.Storage.SaveFile(ctx, u.deps.FieldEncryptor, u.deps.Logger, objectName, bytes.NewReader(body))
}

func (u *WalUploader) releaseClaim(id uuid.UUID) {
	if err := u.deps.WalSegmentRepo.DeleteClaim(id); err != nil {
		u.deps.Logger.Error("failed to release wal segment claim", "wal_segment_id", id, "error", err)
	}
}

func (u *WalUploader) deleteObject(objectName string) {
	if err := u.deps.Storage.DeleteFile(u.deps.FieldEncryptor, objectName); err != nil {
		u.deps.Logger.Warn("failed to delete orphaned wal storage object", "file_name", objectName, "error", err)
	}
}

func (u *WalUploader) removeLocal(localPath, walFilename string) {
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		u.deps.Logger.Warn("failed to remove local wal segment", "wal_filename", walFilename, "error", err)
	}
}

func segmentBounds(walFilename string, segSizeBytes int64) (timelineID int, startLSN, endLSN walmath.LSN, err error) {
	timeline, segmentNo, err := walmath.ParseWALFilenameWithSize(walFilename, uint64(segSizeBytes))
	if err != nil {
		return 0, 0, 0, err
	}

	start := walmath.LSN(segmentNo * uint64(segSizeBytes))

	return int(timeline), start, start + walmath.LSN(segSizeBytes), nil
}

// walSegmentObjectName is the deterministic storage key for a WAL segment:
// "<db>-WAL-tl<TL>-<wal_filename>.zst". No UUID — the insert-first claim model
// guarantees a single writer per (database_id, timeline_id, wal_filename), so the
// deterministic name is safe and dedup-friendly (matches the .history convention).
func walSegmentObjectName(databaseID uuid.UUID, timelineID int, walFilename string) string {
	return fmt.Sprintf("%s-WAL-tl%d-%s.zst", databaseID, timelineID, walFilename)
}

type walSegmentArtifactReader struct {
	reader     *countingReader
	pipeReader *io.PipeReader
	done       chan error
}

func (r *walSegmentArtifactReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *walSegmentArtifactReader) abort(err error) {
	_ = r.pipeReader.CloseWithError(err)
}

func (r *walSegmentArtifactReader) wait() (int64, error) {
	err := <-r.done

	return r.reader.bytesRead, err
}

type countingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)

	return n, err
}

func buildWalSegmentArtifactReader(
	localPath string,
	encryption backups_core_enums.BackupEncryption,
	masterKey string,
	segmentID uuid.UUID,
) (*walSegmentArtifactReader, string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("open wal segment file: %w", err)
	}

	pipeReader, pipeWriter := io.Pipe()
	output := io.Writer(pipeWriter)
	var encryptionWriter io.Closer
	var salt, iv string

	if encryption == backups_core_enums.BackupEncryptionEncrypted {
		encSetup, err := backup_encryption.SetupEncryptionWriter(pipeWriter, masterKey, segmentID)
		if err != nil {
			_ = file.Close()
			_ = pipeReader.Close()
			_ = pipeWriter.Close()

			return nil, "", "", fmt.Errorf("setup wal segment encryption: %w", err)
		}

		output = encSetup.Writer
		encryptionWriter = encSetup.Writer
		salt = encSetup.SaltBase64
		iv = encSetup.NonceBase64
	}

	artifact := &walSegmentArtifactReader{
		reader:     &countingReader{reader: pipeReader},
		pipeReader: pipeReader,
		done:       make(chan error, 1),
	}

	go func() {
		defer func() { _ = file.Close() }()

		err := streamWalSegmentArtifact(file, output, encryptionWriter)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			artifact.done <- err

			return
		}

		artifact.done <- pipeWriter.Close()
	}()

	return artifact, salt, iv, nil
}

func streamWalSegmentArtifact(source io.Reader, output io.Writer, encryptionWriter io.Closer) error {
	encoder, err := zstd.NewWriter(output, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return fmt.Errorf("init zstd writer: %w", err)
	}

	if _, err := io.Copy(encoder, source); err != nil {
		_ = encoder.Close()

		return fmt.Errorf("zstd encode: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close zstd writer: %w", err)
	}

	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			return fmt.Errorf("close wal segment encryption writer: %w", err)
		}
	}

	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
