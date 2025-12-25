package audit_logs

import (
	"errors"
	"log/slog"
	"time"

	user_enums "databasus-backend/internal/features/users/enums"
	user_models "databasus-backend/internal/features/users/models"

	"github.com/google/uuid"
)

type AuditLogService struct {
	auditLogRepository *AuditLogRepository
	logger             *slog.Logger
}

func (s *AuditLogService) WriteAuditLog(
	message string,
	userID *uuid.UUID,
	workspaceID *uuid.UUID,
) {
	auditLog := &AuditLog{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Message:     message,
		CreatedAt:   time.Now().UTC(),
	}

	err := s.auditLogRepository.Create(auditLog)
	if err != nil {
		s.logger.Error("failed to create audit log", "error", err)
		return
	}
}

func (s *AuditLogService) CreateAuditLog(auditLog *AuditLog) error {
	return s.auditLogRepository.Create(auditLog)
}

func (s *AuditLogService) GetGlobalAuditLogs(
	user *user_models.User,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	if user.Role != user_enums.UserRoleAdmin {
		return nil, errors.New("only administrators can view global audit logs")
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetGlobal(limit, offset, request.BeforeDate)
	if err != nil {
		return nil, err
	}

	total, err := s.auditLogRepository.CountGlobal(request.BeforeDate)
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
	targetUserID uuid.UUID,
	user *user_models.User,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	// Users can view their own logs, ADMIN can view any user's logs
	if user.Role != user_enums.UserRoleAdmin && user.ID != targetUserID {
		return nil, errors.New("insufficient permissions to view user audit logs")
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByUser(
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
	workspaceID uuid.UUID,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByWorkspace(
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
