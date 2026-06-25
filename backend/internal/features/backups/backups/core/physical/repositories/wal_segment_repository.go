package physical_repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/walmath"
)

type PhysicalWalSegmentRepository struct{}

func (r *PhysicalWalSegmentRepository) Insert(segment *physical_models.PhysicalWalSegment) error {
	if segment.DatabaseID == uuid.Nil || segment.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	if segment.ID == uuid.Nil {
		segment.ID = uuid.New()
	}

	now := time.Now().UTC()
	if segment.ReceivedAt.IsZero() {
		segment.ReceivedAt = now
	}
	if segment.ClaimedAt.IsZero() {
		segment.ClaimedAt = now
	}

	return storage.GetDb().Create(segment).Error
}

func (r *PhysicalWalSegmentRepository) FindByID(id uuid.UUID) (*physical_models.PhysicalWalSegment, error) {
	var segment physical_models.PhysicalWalSegment

	if err := storage.GetDb().Where("id = ?", id).First(&segment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &segment, nil
}

// FindByChainSpan returns the timeline's WAL segments overlapping [startLSN,
// endLSN), ordered by start_lsn. Overlap (end_lsn > startLSN) — not start_lsn >=
// startLSN — is deliberate: a FULL's start_lsn is usually mid-segment, so the
// segment that physically holds the bytes from the chain's start onward has a
// file-boundary start_lsn BELOW startLSN. Excluding it would drop the very
// segment that bridges the FULL's stop_lsn into the next segment, manufacturing
// a false WAL gap at the start of every restore replay window.
func (r *PhysicalWalSegmentRepository) FindByChainSpan(
	databaseID uuid.UUID,
	timelineID int,
	startLSN, endLSN walmath.LSN,
) ([]*physical_models.PhysicalWalSegment, error) {
	var segments []*physical_models.PhysicalWalSegment

	if err := storage.
		GetDb().
		Where(
			"database_id = ? AND timeline_id = ? AND end_lsn > ?::pg_lsn AND start_lsn < ?::pg_lsn",
			databaseID, timelineID, startLSN.String(), endLSN.String(),
		).
		Order("start_lsn ASC").
		Find(&segments).Error; err != nil {
		return nil, err
	}

	return segments, nil
}

// Anti-join not covered by idx_physical_wal_segments_database_id_received_at;
// expect a seq scan on physical_full_backups when invoked in bulk.
func (r *PhysicalWalSegmentRepository) FindOrphans(
	databaseID uuid.UUID,
) ([]*physical_models.PhysicalWalSegment, error) {
	var orphans []*physical_models.PhysicalWalSegment

	if err := storage.
		GetDb().
		Raw(`
			SELECT *
			FROM physical_wal_segments w
			WHERE w.database_id = ?
			  AND NOT EXISTS (
			    SELECT 1
			    FROM physical_full_backups f
			    WHERE f.database_id = w.database_id
			      AND f.timeline_id = w.timeline_id
			      AND f.start_lsn IS NOT NULL
			      AND f.start_lsn <= w.start_lsn
			      AND f.status = ?
			  )
			ORDER BY w.start_lsn ASC
		`, databaseID, physical_enums.PhysicalBackupStatusCompleted).
		Scan(&orphans).Error; err != nil {
		return nil, err
	}

	return orphans, nil
}

func (r *PhysicalWalSegmentRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&physical_models.PhysicalWalSegment{}, "id = ?", id).Error
}

// DeleteAbandonedClaims removes insert-first WAL claim rows whose upload never
// finished (file_name still NULL) and that have aged past the grace period. A
// NULL file_name is proof no bytes were ever written under any name, so there
// is no storage object to delete. Returns the number of rows removed.
func (r *PhysicalWalSegmentRepository) DeleteAbandonedClaims(
	databaseID uuid.UUID,
	olderThan time.Time,
) (int64, error) {
	result := storage.
		GetDb().
		Where("database_id = ? AND file_name IS NULL AND claimed_at < ?", databaseID, olderThan).
		Delete(&physical_models.PhysicalWalSegment{})

	return result.RowsAffected, result.Error
}

// ClaimInsert inserts an upload-claim row (file_name still NULL) for a freshly
// rotated WAL segment, deduplicated on (database_id, timeline_id, start_lsn) via
// ON CONFLICT DO NOTHING. inserted=true means this caller won the claim and must
// proceed to upload; inserted=false means another uploader already holds (or
// completed) the slot — the caller probes FindByChainKey to decide what to do.
func (r *PhysicalWalSegmentRepository) ClaimInsert(
	segment *physical_models.PhysicalWalSegment,
) (inserted bool, err error) {
	if segment.DatabaseID == uuid.Nil || segment.StorageID == uuid.Nil {
		return false, errors.New("database ID and storage ID are required")
	}

	if segment.ID == uuid.Nil {
		segment.ID = uuid.New()
	}

	now := time.Now().UTC()
	if segment.ReceivedAt.IsZero() {
		segment.ReceivedAt = now
	}
	if segment.ClaimedAt.IsZero() {
		segment.ClaimedAt = now
	}

	result := storage.
		GetDb().
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "database_id"}, {Name: "timeline_id"}, {Name: "start_lsn"}},
			DoNothing: true,
		}).
		Create(segment)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// FindByChainKey returns the row for a specific (database_id, timeline_id,
// start_lsn) — the uploader's conflict probe after ClaimInsert lost the race,
// to distinguish a durably-completed segment (file_name NOT NULL) from an
// in-flight claim (file_name NULL). Returns nil when absent.
func (r *PhysicalWalSegmentRepository) FindByChainKey(
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) (*physical_models.PhysicalWalSegment, error) {
	var segment physical_models.PhysicalWalSegment

	err := storage.
		GetDb().
		Where("database_id = ? AND timeline_id = ? AND start_lsn = ?::pg_lsn",
			databaseID, timelineID, startLSN.String()).
		First(&segment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &segment, nil
}

// MarkUploaded flips file_name from NULL to the durable object key, guarded by
// file_name IS NULL so a DeleteFull cascade that removed the claim mid-upload
// makes this a no-op. updated=true means the upload is durably committed;
// updated=false means the NULL claim no longer exists (cascade caught it) and the
// caller must DeleteFile the now-orphaned storage object.
func (r *PhysicalWalSegmentRepository) MarkUploaded(
	id uuid.UUID,
	fileName string,
	compressedSizeMb float64,
	encryptionSalt, encryptionIV *string,
) (updated bool, err error) {
	result := storage.
		GetDb().
		Model(&physical_models.PhysicalWalSegment{}).
		Where("id = ? AND file_name IS NULL", id).
		Updates(map[string]any{
			"file_name":          fileName,
			"compressed_size_mb": compressedSizeMb,
			"encryption_salt":    encryptionSalt,
			"encryption_iv":      encryptionIV,
		})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// DeleteClaim removes an upload-claim row, guarded by file_name IS NULL so it can
// never delete a durably-committed segment. Idempotent. Used on SaveFile failure
// to release the claim so the next uploader tick can re-claim the same segment.
func (r *PhysicalWalSegmentRepository) DeleteClaim(id uuid.UUID) error {
	return storage.
		GetDb().
		Where("id = ? AND file_name IS NULL", id).
		Delete(&physical_models.PhysicalWalSegment{}).Error
}

// FindLatestCommittedBefore returns the highest end_lsn among committed
// (file_name NOT NULL) segments on (database_id, timeline_id) whose start_lsn is
// strictly before startLSN — the predecessor the post-upload gap probe compares
// against. Returns nil when there is no committed predecessor (the just-uploaded
// segment is the first on its timeline). In-flight claims are excluded because
// their end_lsn is predicted, not durable.
func (r *PhysicalWalSegmentRepository) FindLatestCommittedBefore(
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) (*walmath.LSN, error) {
	var segment physical_models.PhysicalWalSegment

	err := storage.
		GetDb().
		Where("database_id = ? AND timeline_id = ? AND start_lsn < ?::pg_lsn AND file_name IS NOT NULL",
			databaseID, timelineID, startLSN.String()).
		Order("start_lsn DESC").
		First(&segment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	endLSN := segment.EndLSN

	return &endLSN, nil
}
