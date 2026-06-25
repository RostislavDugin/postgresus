package verification_runs

import (
	"errors"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/storage"
)

type VerificationRepository struct{}

func (r *VerificationRepository) Create(verification *RestoreVerification) error {
	if verification.ID == uuid.Nil {
		verification.ID = uuid.New()
	}

	return storage.GetDb().Create(verification).Error
}

func (r *VerificationRepository) FindByID(id uuid.UUID) (*RestoreVerification, error) {
	var verification RestoreVerification

	err := storage.GetDb().
		Preload("TableStats").
		Where("id = ?", id).
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &verification, nil
}

func (r *VerificationRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindByDatabaseIDWithPagination(
	databaseID uuid.UUID,
	limit, offset int,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) CountByDatabaseID(databaseID uuid.UUID) (int64, error) {
	var count int64

	err := storage.GetDb().
		Model(&RestoreVerification{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error

	return count, err
}

func (r *VerificationRepository) ListRunningBackupsByAgentID(
	agentID uuid.UUID,
) ([]*backups_core_logical.LogicalBackup, error) {
	backups := make([]*backups_core_logical.LogicalBackup, 0)

	err := storage.GetDb().
		Table("logical_backups b").
		Select("b.*").
		Joins("JOIN restore_verifications v ON v.backup_id = b.id").
		Where("v.agent_id = ? AND v.status = ?", agentID, VerificationStatusRunning).
		Scan(&backups).Error

	return backups, err
}

// ClaimCandidate pairs a PENDING verification with its backup so the service can run
// the disk-fit check in Go without re-fetching the backup row.
type ClaimCandidate struct {
	Verification *RestoreVerification
	Backup       *backups_core_logical.LogicalBackup
}

// FindOldestPendingClaimablesTop100 locks up to 100 oldest PENDING verification rows
// with SKIP LOCKED and returns each one paired with its backup. The service iterates
// these in order and picks the first that fits the agent's budget — see
// DoesVerificationFit. Locks held for the rest are released when the caller's tx
// commits or rolls back; SKIP LOCKED ensures concurrent agents don't queue on them.
func (r *VerificationRepository) FindOldestPendingClaimablesTop100(
	tx *gorm.DB,
) ([]ClaimCandidate, error) {
	var verifications []*RestoreVerification

	err := tx.
		Where("status = ?", VerificationStatusPending).
		Order("created_at").
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Limit(100).
		Find(&verifications).Error
	if err != nil {
		return nil, err
	}

	if len(verifications) == 0 {
		return nil, nil
	}

	backupIDs := make([]uuid.UUID, len(verifications))
	for i, verification := range verifications {
		backupIDs[i] = verification.BackupID
	}

	var backups []*backups_core_logical.LogicalBackup
	if err := tx.Where("id IN ?", backupIDs).Find(&backups).Error; err != nil {
		return nil, err
	}

	backupByID := make(map[uuid.UUID]*backups_core_logical.LogicalBackup, len(backups))
	for _, backup := range backups {
		backupByID[backup.ID] = backup
	}

	candidates := make([]ClaimCandidate, 0, len(verifications))
	for _, verification := range verifications {
		backup, ok := backupByID[verification.BackupID]
		if !ok {
			continue
		}

		candidates = append(candidates, ClaimCandidate{
			Verification: verification,
			Backup:       backup,
		})
	}

	return candidates, nil
}

func (r *VerificationRepository) UpdateClaim(
	tx *gorm.DB,
	verificationID, agentID uuid.UUID,
	startedAt time.Time,
) error {
	result := tx.Model(&RestoreVerification{}).
		Where("id = ? AND status = ? AND agent_id IS NULL", verificationID, VerificationStatusPending).
		Updates(map[string]any{
			"status":     VerificationStatusRunning,
			"agent_id":   agentID,
			"started_at": startedAt,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("verification claim failed: row no longer PENDING")
	}

	return nil
}

// WriteSuccessReport atomically applies a COMPLETED agent report: CAS-updates the
// verification row (still RUNNING and owned by this agent) and inserts its table
// stats in the same transaction. Returns errVerificationGone when the CAS sees zero
// rows updated — the controller maps that to 410.
func (r *VerificationRepository) WriteSuccessReport(
	verificationID, agentID uuid.UUID,
	report *ReportRequest,
) error {
	return storage.GetDb().Transaction(func(tx *gorm.DB) error {
		fields := map[string]any{
			"status":                      VerificationStatusCompleted,
			"finished_at":                 time.Now().UTC(),
			"pg_restore_exit_code":        report.PgRestoreExitCode,
			"restore_duration_ms":         report.RestoreDurationMs,
			"verify_duration_ms":          report.VerifyDurationMs,
			"db_size_bytes_after_restore": report.DBSizeBytesAfterRestore,
			"table_count":                 report.TableCount,
			"schema_count":                report.SchemaCount,
			"fail_message":                nil,
		}

		result := tx.Model(&RestoreVerification{}).
			Where(
				"id = ? AND agent_id = ? AND status = ?",
				verificationID, agentID, VerificationStatusRunning,
			).
			Updates(fields)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("verification gone")
		}

		return r.insertTableStats(tx, verificationID, report.TableStats)
	})
}

func (r *VerificationRepository) FindNonTerminalForDatabase(
	tx *gorm.DB,
	databaseID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := tx.
		Where(
			"database_id = ? AND status IN ?",
			databaseID,
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindStalePending(
	threshold time.Time,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ? AND created_at < ?", VerificationStatusPending, threshold).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindAllRunning() ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ?", VerificationStatusRunning).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindRunningByAgent(
	agentID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ? AND agent_id = ?", VerificationStatusRunning, agentID).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindNonTerminalForDisabledConfigs() ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Table("restore_verifications v").
		Select("v.*").
		Joins("JOIN backup_verification_configs c ON c.database_id = v.database_id").
		Where(
			"c.is_scheduled_verification_enabled = ? AND v.status IN ?",
			false,
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Scan(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindLatestFinishedAt(
	databaseID uuid.UUID,
) (*time.Time, error) {
	var verification RestoreVerification

	err := storage.GetDb().
		Where(
			"database_id = ? AND status IN ? AND finished_at IS NOT NULL",
			databaseID,
			[]VerificationStatus{
				VerificationStatusCompleted,
				VerificationStatusFailed,
				VerificationStatusCanceled,
			},
		).
		Order("finished_at DESC").
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return verification.FinishedAt, nil
}

// Requeue resets created_at so each requeue gets a fresh PENDING-too-long window.
func (r *VerificationRepository) Requeue(tx *gorm.DB, verificationID uuid.UUID) error {
	db := tx
	if db == nil {
		db = storage.GetDb()
	}

	now := time.Now().UTC()

	result := db.Model(&RestoreVerification{}).
		Where("id = ?", verificationID).
		Updates(map[string]any{
			"status":        VerificationStatusPending,
			"agent_id":      nil,
			"started_at":    nil,
			"finished_at":   nil,
			"created_at":    now,
			"attempt_count": gorm.Expr("attempt_count + 1"),
			"fail_message":  nil,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("requeue failed: verification not found")
	}

	return nil
}

// MarkTerminal runs inside tx when non-nil to avoid deadlock against the caller's
// own row lock (e.g. displacement during EnqueueManualVerification).
func (r *VerificationRepository) MarkTerminal(
	tx *gorm.DB,
	verificationID uuid.UUID,
	status VerificationStatus,
	fields map[string]any,
) error {
	db := tx
	if db == nil {
		db = storage.GetDb()
	}

	updates := make(map[string]any, len(fields)+2)
	maps.Copy(updates, fields)
	updates["status"] = status
	updates["finished_at"] = time.Now().UTC()

	result := db.Model(&RestoreVerification{}).
		Where("id = ?", verificationID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("mark terminal failed: verification not found")
	}

	return nil
}

func (r *VerificationRepository) insertTableStats(
	tx *gorm.DB,
	verificationID uuid.UUID,
	stats []ReportTableStat,
) error {
	if len(stats) == 0 {
		return nil
	}

	rows := make([]RestoreVerificationTableStat, len(stats))
	for i, stat := range stats {
		rows[i] = RestoreVerificationTableStat{
			ID:                    uuid.New(),
			RestoreVerificationID: verificationID,
			SchemaName:            stat.SchemaName,
			Name:                  stat.Name,
			RowCount:              stat.RowCount,
		}
	}

	return tx.Create(&rows).Error
}
