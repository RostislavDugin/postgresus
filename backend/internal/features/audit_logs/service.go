package audit_logs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	user_enums "databasus-backend/internal/features/users/enums"
	user_models "databasus-backend/internal/features/users/models"
)

type AuditLogService struct {
	auditLogRepository *AuditLogRepository
	logger             *slog.Logger
	auditLogger        *slog.Logger
}

// The log-pipeline write is what makes the trail survive the database it describes: an operator
// who wipes audit_logs cannot erase what already left for the file and the remote backend.
func (s *AuditLogService) WriteAuditLog(ctx context.Context, entry audit_logs_models.AuditEntry) {
	auditLog := &AuditLog{
		UserID:      entry.UserID,
		WorkspaceID: entry.WorkspaceID,
		Message:     entry.Message,
		CreatedAt:   time.Now().UTC(),
	}

	s.auditLogger.InfoContext(ctx, entry.Message, "user_id", entry.UserID, "workspace_id", entry.WorkspaceID)

	if err := s.auditLogRepository.Create(auditLog); err != nil {
		s.logger.ErrorContext(ctx, "failed to create audit log",
			"error", err, "user_id", entry.UserID, "workspace_id", entry.WorkspaceID)
	}
}

func (s *AuditLogService) GetGlobalAuditLogs(
	ctx context.Context,
	user *user_models.User,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	if user.Role != user_enums.UserRoleAdmin {
		return nil, ErrOnlyAdminsCanViewGlobalLogs
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetGlobal(ctx, limit, offset, request.BeforeDate)
	if err != nil {
		return nil, err
	}

	total, err := s.auditLogRepository.CountGlobal(ctx, request.BeforeDate)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *AuditLogService) GetUserAuditLogs(
	ctx context.Context,
	targetUserID uuid.UUID,
	user *user_models.User,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	// Users can view their own logs, ADMIN can view any user's logs
	if user.Role != user_enums.UserRoleAdmin && user.ID != targetUserID {
		return nil, ErrInsufficientPermissionsToViewLogs
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByUser(
		ctx,
		targetUserID,
		limit,
		offset,
		request.BeforeDate,
	)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     int64(len(auditLogs)),
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *AuditLogService) GetWorkspaceAuditLogs(
	ctx context.Context,
	workspaceID uuid.UUID,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByWorkspace(
		ctx,
		workspaceID,
		limit,
		offset,
		request.BeforeDate,
	)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     int64(len(auditLogs)),
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *AuditLogService) CleanOldAuditLogs(ctx context.Context, logger *slog.Logger) error {
	oneYearAgo := time.Now().UTC().Add(-365 * 24 * time.Hour)

	deletedCount, err := s.auditLogRepository.DeleteOlderThan(ctx, oneYearAgo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete old audit logs", "error", err)

		return err
	}

	// Logged even at zero, so that "ran and found nothing" stays distinguishable from "never ran".
	logger.InfoContext(ctx, fmt.Sprintf("deleted %d audit logs older than a year", deletedCount),
		"older_than", oneYearAgo)

	return nil
}
