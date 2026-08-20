package audit_logs

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/storage"
)

type AuditLogRepository struct{}

// Create is the one method here without a context, and deliberately so: the reads may be abandoned
// when the caller goes away, but the row describing an action that already happened must be written
// even if the client disconnected mid-request.
func (r *AuditLogRepository) Create(auditLog *AuditLog) error {
	if auditLog.ID == uuid.Nil {
		auditLog.ID = uuid.New()
	}

	return storage.GetDb().Create(auditLog).Error
}

func (r *AuditLogRepository) GetGlobal(
	ctx context.Context,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.workspace_id,
			al.message,
			al.created_at,
			u.email as user_email,
			u.name as user_name,
			w.name as workspace_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN workspaces w ON al.workspace_id = w.id`

	args := []any{}

	if beforeDate != nil {
		sql += " WHERE al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.workspace_id,
			al.message,
			al.created_at,
			u.email as user_email,
			u.name as user_name,
			w.name as workspace_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN workspaces w ON al.workspace_id = w.id
		WHERE al.user_id = ?`

	args := []any{userID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.workspace_id,
			al.message,
			al.created_at,
			u.email as user_email,
			u.name as user_name,
			w.name as workspace_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN workspaces w ON al.workspace_id = w.id
		WHERE al.workspace_id = ?`

	args := []any{workspaceID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) CountGlobal(ctx context.Context, beforeDate *time.Time) (int64, error) {
	var count int64
	query := storage.GetDb().WithContext(ctx).Model(&AuditLog{})

	if beforeDate != nil {
		query = query.Where("created_at < ?", *beforeDate)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, beforeDate time.Time) (int64, error) {
	result := storage.GetDb().WithContext(ctx).
		Where("created_at < ?", beforeDate).
		Delete(&AuditLog{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
