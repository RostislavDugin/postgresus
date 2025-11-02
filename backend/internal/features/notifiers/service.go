package notifiers

import (
	"errors"
	"fmt"
	"log/slog"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"

	"github.com/google/uuid"
)

type NotifierService struct {
	notifierRepository *NotifierRepository
	logger             *slog.Logger
	workspaceService   *workspaces_services.WorkspaceService
	auditLogService    *audit_logs.AuditLogService
}

func (s *NotifierService) SaveNotifier(
	user *users_models.User,
	workspaceID uuid.UUID,
	notifier *Notifier,
) error {
	canManage, err := s.workspaceService.CanUserManageDBs(workspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to manage notifier in this workspace")
	}

	isUpdate := notifier.ID != uuid.Nil

	if isUpdate {
		existingNotifier, err := s.notifierRepository.FindByID(notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.WorkspaceID != workspaceID {
			return errors.New("notifier does not belong to this workspace")
		}

		notifier.WorkspaceID = existingNotifier.WorkspaceID
	} else {
		notifier.WorkspaceID = workspaceID
	}

	_, err = s.notifierRepository.Save(notifier)
	if err != nil {
		return err
	}

	if isUpdate {
		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Notifier updated: %s", notifier.Name),
			&user.ID,
			&workspaceID,
		)
	} else {
		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Notifier created: %s", notifier.Name),
			&user.ID,
			&workspaceID,
		)
	}

	return nil
}

func (s *NotifierService) DeleteNotifier(
	user *users_models.User,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(notifierID)
	if err != nil {
		return err
	}

	canManage, err := s.workspaceService.CanUserManageDBs(notifier.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to manage notifier in this workspace")
	}

	err = s.notifierRepository.Delete(notifier)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Notifier deleted: %s", notifier.Name),
		&user.ID,
		&notifier.WorkspaceID,
	)

	return nil
}

func (s *NotifierService) GetNotifier(
	user *users_models.User,
	id uuid.UUID,
) (*Notifier, error) {
	notifier, err := s.notifierRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(notifier.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view notifier in this workspace")
	}

	return notifier, nil
}

func (s *NotifierService) GetNotifiers(
	user *users_models.User,
	workspaceID uuid.UUID,
) ([]*Notifier, error) {
	canView, _, err := s.workspaceService.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view notifiers in this workspace")
	}

	return s.notifierRepository.FindByWorkspaceID(workspaceID)
}

func (s *NotifierService) SendTestNotification(
	user *users_models.User,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(notifierID)
	if err != nil {
		return err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(notifier.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canView {
		return errors.New("insufficient permissions to test notifier in this workspace")
	}

	err = notifier.Send(s.logger, "Test message", "This is a test message")
	if err != nil {
		return err
	}

	_, err = s.notifierRepository.Save(notifier)
	if err != nil {
		return err
	}

	return nil
}

func (s *NotifierService) SendTestNotificationToNotifier(
	notifier *Notifier,
) error {
	return notifier.Send(s.logger, "Test message", "This is a test message")
}

func (s *NotifierService) SendNotification(
	notifier *Notifier,
	title string,
	message string,
) {
	// Truncate message to 2000 characters if it's too long
	messageRunes := []rune(message)
	if len(messageRunes) > 2000 {
		message = string(messageRunes[:2000])
	}

	notifiedFromDb, err := s.notifierRepository.FindByID(notifier.ID)
	if err != nil {
		return
	}

	err = notifiedFromDb.Send(s.logger, title, message)
	if err != nil {
		errMsg := err.Error()
		notifiedFromDb.LastSendError = &errMsg

		_, err = s.notifierRepository.Save(notifiedFromDb)
		if err != nil {
			s.logger.Error("Failed to save notifier", "error", err)
		}
	}

	notifiedFromDb.LastSendError = nil
	_, err = s.notifierRepository.Save(notifiedFromDb)
	if err != nil {
		s.logger.Error("Failed to save notifier", "error", err)
	}
}

func (s *NotifierService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	notifiers, err := s.notifierRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get notifiers for workspace deletion: %w", err)
	}

	for _, notifier := range notifiers {
		if err := s.notifierRepository.Delete(notifier); err != nil {
			return fmt.Errorf("failed to delete notifier %s: %w", notifier.ID, err)
		}
	}

	return nil
}
