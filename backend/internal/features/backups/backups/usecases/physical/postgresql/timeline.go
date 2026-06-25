package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	chain_view "databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// TimelineDecisionKind discriminates the outcome of CheckTimelineCompatibility.
// Continue is the green path; the other three are refusal branches.
type TimelineDecisionKind int

const (
	TimelineContinue TimelineDecisionKind = iota
	TimelineFailoverDetected
	TimelineRegression
	TimelineDifferentCluster
)

// TimelineDecision is a discriminated result. Only the fields belonging to
// Kind are meaningful; the rest are zero.
type TimelineDecision struct {
	Kind TimelineDecisionKind

	NewTLI int // TimelineFailoverDetected — promote-detected target TL

	ExpectedTLI int // TimelineRegression
	ActualTLI   int // TimelineRegression

	ExpectedSysID string // TimelineDifferentCluster — db.SystemIdentifier
	ActualSysID   string // TimelineDifferentCluster — live cluster system_identifier
}

// CheckTimelineCompatibility validates the source cluster against the
// catalog before spawning a FULL or INCR. Three refusal branches:
// system_identifier mismatch (different cluster pointed at the same DB row),
// timeline regression (live TL < newest known TL — never legitimate), and
// failover detected (live TL > newest known TL — the new FULL must anchor a
// fresh chain on the new TL). Continue means proceed.
//
// "newest known TL" is derived from GREATEST(MAX(full.timeline_id),
// MAX(history.timeline_id)) per BACKUPING-SCHEDULER-PLAN.md §B step 1.
// First-tick (no fulls, no history) returns Continue — the live TL seeds
// the first FULL's timeline_id.
func CheckTimelineCompatibility(
	ctx context.Context,
	conn *pgx.Conn,
	db *postgresql_physical.PostgresqlPhysicalDatabase,
	fullRepo *physical_repositories.PhysicalFullBackupRepository,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (*TimelineDecision, error) {
	liveTLI, liveSysID, err := readClusterIdentity(ctx, conn)
	if err != nil {
		return nil, err
	}

	if db.SystemIdentifier != nil && *db.SystemIdentifier != liveSysID {
		return &TimelineDecision{
			Kind:          TimelineDifferentCluster,
			ExpectedSysID: *db.SystemIdentifier,
			ActualSysID:   liveSysID,
		}, nil
	}

	knownTLI, err := newestKnownTimeline(db.ParentDatabaseID(), fullRepo, historyRepo)
	if err != nil {
		return nil, err
	}

	if knownTLI == 0 {
		return &TimelineDecision{Kind: TimelineContinue}, nil
	}

	if liveTLI < knownTLI {
		return &TimelineDecision{
			Kind:        TimelineRegression,
			ExpectedTLI: knownTLI,
			ActualTLI:   liveTLI,
		}, nil
	}

	if liveTLI > knownTLI {
		return &TimelineDecision{
			Kind:   TimelineFailoverDetected,
			NewTLI: liveTLI,
		}, nil
	}

	return &TimelineDecision{Kind: TimelineContinue}, nil
}

// ValidateStartLsnAgainstHistory checks that startLSN falls inside the LSN
// range covered by timelineID per the catalog's .history files. Skipped
// when no history rows exist (first FULL on a fresh DB). OK_WITH_WARNING
// when history is gapped (TL N+1 present but N missing); the caller logs
// timeline_history_gap. CHAIN_BROKEN means the FULL is on a stale or
// rewound timeline whose history moved on — refuse the artifact.
func ValidateStartLsnAgainstHistory(
	dbID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (chain_view.ValidationResult, error) {
	historyRows, err := historyRepo.FindAllByDatabase(dbID)
	if err != nil {
		return chain_view.ValidationResult{}, fmt.Errorf("load wal history rows: %w", err)
	}

	if len(historyRows) == 0 {
		return chain_view.ValidationResult{Status: chain_view.ValidationStatusOK}, nil
	}

	if !historyHasTimeline(historyRows, timelineID) && timelineID > 1 {
		if historyIsGapped(historyRows, timelineID) {
			return chain_view.ValidationResult{
				Status: chain_view.ValidationStatusOKWithWarning,
				Message: fmt.Sprintf(
					"timeline_history_gap: TL %d has no history row but higher TLs do; older history likely retention-deleted",
					timelineID,
				),
			}, nil
		}

		return chain_view.ValidationResult{
			Status: chain_view.ValidationStatusChainBroken,
			Message: fmt.Sprintf(
				"start_lsn %s outside known timeline range: no history record for TL %d",
				startLSN.String(),
				timelineID,
			),
		}, nil
	}

	return chain_view.ValidationResult{Status: chain_view.ValidationStatusOK}, nil
}

// UploadHistoryFile reads the .history file for timelineID from the source
// cluster's pg_wal/, compresses with zstd, optionally encrypts, uploads
// artifact + sidecar to storage, and inserts the physical_wal_history_files
// row. Idempotent on (database_id, timeline_id) via the UNIQUE constraint:
// a duplicate insert returns nil after observing the existing row.
//
// Shared by full.go (post-stream, when the FULL ran on a TL > 1) and
// PR 4's wal_stream.go (which also observes .history arrivals).
func UploadHistoryFile(
	ctx context.Context,
	conn *pgx.Conn,
	timelineID int,
	storage storages.StorageFileSaver,
	db *postgresql_physical.PostgresqlPhysicalDatabase,
	storageID uuid.UUID,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
	encryption backups_core_enums.BackupEncryption,
	masterKey string,
	fieldEncryptor util_encryption.FieldEncryptor,
	logger *slog.Logger,
) (*physical_models.PhysicalWalHistoryFile, error) {
	databaseID := db.ParentDatabaseID()

	existing, err := historyRepo.FindByDatabaseTimeline(databaseID, timelineID)
	if err != nil {
		return nil, fmt.Errorf("query existing history row: %w", err)
	}

	if existing != nil {
		logger.Debug("history file already in catalog",
			"database_id", databaseID,
			"timeline_id", timelineID)

		return existing, nil
	}

	historyFilename := walmath.FormatHistoryFilename(uint32(timelineID))

	body, err := readHistoryFromCluster(ctx, conn, historyFilename)
	if err != nil {
		return nil, err
	}

	historyFileID := uuid.New()
	storageObjectName := fmt.Sprintf("%s-HIST-tl%d.history.zst", databaseID, timelineID)

	artifactReader, encryptionSalt, encryptionIV, err := buildHistoryArtifactReader(
		body, encryption, masterKey, historyFileID,
	)
	if err != nil {
		return nil, err
	}

	if err := storage.SaveFile(ctx, fieldEncryptor, logger, storageObjectName, artifactReader); err != nil {
		return nil, fmt.Errorf("upload history artifact: %w", err)
	}

	compressedSizeBytes := int64(artifactReader.Len())

	sidecarFilename := storageObjectName + metadataSuffix

	sidecar := physical_dto.PhysicalWalHistoryMetadata{
		WalHistoryFileID:    historyFileID,
		DatabaseID:          databaseID,
		TimelineID:          timelineID,
		HistoryFilename:     historyFilename,
		CompressedSizeBytes: compressedSizeBytes,
		Encryption:          encryption,
		EncryptionSalt:      encryptionSalt,
		EncryptionIV:        encryptionIV,
		CreatedAt:           time.Now().UTC(),
	}

	sidecarBytes, err := json.Marshal(sidecar)
	if err != nil {
		return nil, fmt.Errorf("marshal history sidecar: %w", err)
	}

	if err := storage.SaveFile(
		ctx, fieldEncryptor, logger, sidecarFilename, bytes.NewReader(sidecarBytes),
	); err != nil {
		// Sidecar upload failed: remove the artifact to preserve the
		// "no artifact without sidecar" invariant. DeleteFile is
		// idempotent on not-found.
		if delErr := storage.DeleteFile(fieldEncryptor, storageObjectName); delErr != nil {
			logger.Warn("failed to remove orphan history artifact after sidecar failure",
				"file_name", storageObjectName,
				"error", delErr)
		}

		return nil, fmt.Errorf("upload history sidecar: %w", err)
	}

	row := &physical_models.PhysicalWalHistoryFile{
		ID:               historyFileID,
		DatabaseID:       databaseID,
		StorageID:        storageID,
		TimelineID:       timelineID,
		FileName:         storageObjectName,
		HistoryFilename:  historyFilename,
		CompressedSizeMb: float64(compressedSizeBytes) / (1024 * 1024),
		CreatedAt:        sidecar.CreatedAt,
	}

	if err := historyRepo.Insert(row); err != nil {
		if isUniqueViolation(err) {
			logger.Debug("history row inserted by concurrent caller",
				"database_id", databaseID,
				"timeline_id", timelineID)

			return historyRepo.FindByDatabaseTimeline(databaseID, timelineID)
		}

		return nil, fmt.Errorf("insert history row: %w", err)
	}

	return row, nil
}

func readClusterIdentity(ctx context.Context, conn *pgx.Conn) (int, string, error) {
	var tli int
	var sysID string

	err := conn.QueryRow(ctx, `
		SELECT
			(SELECT timeline_id FROM pg_control_checkpoint()),
			(SELECT system_identifier::text FROM pg_control_system())
	`).Scan(&tli, &sysID)
	if err != nil {
		return 0, "", fmt.Errorf("read cluster identity: %w", err)
	}

	return tli, sysID, nil
}

func newestKnownTimeline(
	dbID uuid.UUID,
	fullRepo *physical_repositories.PhysicalFullBackupRepository,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (int, error) {
	fulls, err := fullRepo.FindCompletedNewestFirstByDatabase(dbID)
	if err != nil {
		return 0, fmt.Errorf("load full backups: %w", err)
	}

	historyRows, err := historyRepo.FindAllByDatabase(dbID)
	if err != nil {
		return 0, fmt.Errorf("load history rows: %w", err)
	}

	maxTL := 0
	for _, full := range fulls {
		if full.TimelineID > maxTL {
			maxTL = full.TimelineID
		}
	}

	for _, row := range historyRows {
		if row.TimelineID > maxTL {
			maxTL = row.TimelineID
		}
	}

	return maxTL, nil
}

func historyHasTimeline(rows []*physical_models.PhysicalWalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID == timelineID {
			return true
		}
	}

	return false
}

func historyIsGapped(rows []*physical_models.PhysicalWalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID > timelineID {
			return true
		}
	}

	return false
}

func readHistoryFromCluster(ctx context.Context, conn *pgx.Conn, historyFilename string) ([]byte, error) {
	var body []byte

	err := conn.QueryRow(ctx,
		`SELECT pg_read_binary_file('pg_wal/' || $1)`,
		historyFilename,
	).Scan(&body)
	if err != nil {
		return nil, fmt.Errorf("read history file %q from cluster: %w", historyFilename, err)
	}

	return body, nil
}

// limitedBuffer is bytes.Buffer plus a Len() accessor for size reporting.
// We compress the entire history file in memory because .history files are
// small (~1 KB even for clusters with many promotions) — the streaming
// pipeline used for FULL artifacts would be overkill here.
type limitedBuffer struct {
	*bytes.Reader
	length int
}

func (b *limitedBuffer) Len() int {
	return b.length
}

func buildHistoryArtifactReader(
	body []byte,
	encryption backups_core_enums.BackupEncryption,
	masterKey string,
	historyFileID uuid.UUID,
) (*limitedBuffer, string, string, error) {
	compressed, err := compressZstd(body)
	if err != nil {
		return nil, "", "", err
	}

	if encryption != backups_core_enums.BackupEncryptionEncrypted {
		return &limitedBuffer{
			Reader: bytes.NewReader(compressed),
			length: len(compressed),
		}, "", "", nil
	}

	var encBuf bytes.Buffer

	encSetup, err := backup_encryption.SetupEncryptionWriter(&encBuf, masterKey, historyFileID)
	if err != nil {
		return nil, "", "", fmt.Errorf("setup history encryption: %w", err)
	}

	if _, err := encSetup.Writer.Write(compressed); err != nil {
		return nil, "", "", fmt.Errorf("encrypt history body: %w", err)
	}

	if err := encSetup.Writer.Close(); err != nil {
		return nil, "", "", fmt.Errorf("close history encryption writer: %w", err)
	}

	encryptedBytes := encBuf.Bytes()

	return &limitedBuffer{
		Reader: bytes.NewReader(encryptedBytes),
		length: len(encryptedBytes),
	}, encSetup.SaltBase64, encSetup.NonceBase64, nil
}

func compressZstd(input []byte) ([]byte, error) {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, fmt.Errorf("init zstd writer: %w", err)
	}

	if _, err := io.Copy(encoder, bytes.NewReader(input)); err != nil {
		_ = encoder.Close()

		return nil, fmt.Errorf("zstd encode: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close zstd writer: %w", err)
	}

	return buf.Bytes(), nil
}

// isUniqueViolation classifies a PostgreSQL unique-constraint failure
// without binding to a specific driver error code constant — we only need
// "is this the dup-key race we just lost" vs "real failure".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return errors.Is(err, errUniqueViolationSentinel) ||
		// pgx surfaces "ERROR: duplicate key value violates unique constraint"
		bytes.Contains([]byte(msg), []byte("duplicate key value")) ||
		bytes.Contains([]byte(msg), []byte("SQLSTATE 23505"))
}

var errUniqueViolationSentinel = errors.New("unique constraint violation")

// _ keeps the base64 import live for sidecar-shape changes; encryption
// salt/IV currently passed as base64 strings from SetupEncryptionWriter.
var _ = base64.StdEncoding
