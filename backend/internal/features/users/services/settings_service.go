package users_services

import (
	"context"
	"fmt"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_interfaces "databasus-backend/internal/features/users/interfaces"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
)

type SettingsService struct {
	userSettingsRepository *users_repositories.UsersSettingsRepository
	auditLogWriter         users_interfaces.AuditLogWriter
}

func (s *SettingsService) SetAuditLogWriter(writer users_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *SettingsService) GetSettings(ctx context.Context) (*users_models.UsersSettings, error) {
	return s.userSettingsRepository.GetSettings(ctx)
}

func (s *SettingsService) UpdateSettings(
	ctx context.Context,
	request users_models.UsersSettings,
	updatedBy *users_models.User,
) (*users_models.UsersSettings, error) {
	if !updatedBy.CanUpdateSettings() {
		return nil, fmt.Errorf("insufficient permissions to update settings")
	}

	existingSettings, err := s.userSettingsRepository.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current settings: %w", err)
	}

	auditLogMessages := []string{}

	if request.IsAllowExternalRegistrations != existingSettings.IsAllowExternalRegistrations {
		existingSettings.IsAllowExternalRegistrations = request.IsAllowExternalRegistrations
		auditLogMessages = append(
			auditLogMessages,
			fmt.Sprintf(
				"isAllowExternalRegistrations: %t -> %t",
				existingSettings.IsAllowExternalRegistrations,
				request.IsAllowExternalRegistrations,
			),
		)
	}

	if request.IsAllowMemberInvitations != existingSettings.IsAllowMemberInvitations {
		existingSettings.IsAllowMemberInvitations = request.IsAllowMemberInvitations
		auditLogMessages = append(
			auditLogMessages,
			fmt.Sprintf(
				"isAllowMemberInvitations: %t -> %t",
				existingSettings.IsAllowMemberInvitations,
				request.IsAllowMemberInvitations,
			),
		)
	}

	if request.IsMemberAllowedToCreateWorkspaces != existingSettings.IsMemberAllowedToCreateWorkspaces {
		existingSettings.IsMemberAllowedToCreateWorkspaces = request.IsMemberAllowedToCreateWorkspaces
	}

	if err := s.userSettingsRepository.UpdateSettings(ctx, existingSettings); err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	for _, message := range auditLogMessages {
		s.auditLogWriter.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     message,
			UserID:      &updatedBy.ID,
			WorkspaceID: nil,
		})
	}

	return existingSettings, nil
}
