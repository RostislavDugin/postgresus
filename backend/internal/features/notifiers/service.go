package notifiers

import (
	"fmt"
	"log/slog"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"

	"github.com/google/uuid"
)

type NotifierService struct {
	notifierRepository      *NotifierRepository
	logger                  *slog.Logger
	workspaceService        *workspaces_services.WorkspaceService
	auditLogService         *audit_logs.AuditLogService
	fieldEncryptor          encryption.FieldEncryptor
	notifierDatabaseCounter NotifierDatabaseCounter
}

func (s *NotifierService) SetNotifierDatabaseCounter(
	notifierDatabaseCounter NotifierDatabaseCounter,
) {
	s.notifierDatabaseCounter = notifierDatabaseCounter
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
		return ErrInsufficientPermissionsToManageNotifier
	}

	isUpdate := notifier.ID != uuid.Nil

	if isUpdate {
		existingNotifier, err := s.notifierRepository.FindByID(notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.WorkspaceID != workspaceID {
			return ErrNotifierDoesNotBelongToWorkspace
		}

		existingNotifier.Update(notifier)

		if err := existingNotifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := existingNotifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.notifierRepository.Save(existingNotifier)
		if err != nil {
			return err
		}

		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Notifier updated: %s", existingNotifier.Name),
			&user.ID,
			&workspaceID,
		)
	} else {
		notifier.WorkspaceID = workspaceID

		if err := notifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := notifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.notifierRepository.Save(notifier)
		if err != nil {
			return err
		}

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
		return ErrInsufficientPermissionsToManageNotifier
	}

	attachedDatabasesIDs, err := s.notifierDatabaseCounter.GetNotifierAttachedDatabasesIDs(
		notifier.ID,
	)
	if err != nil {
		return err
	}
	if len(attachedDatabasesIDs) > 0 {
		return ErrNotifierHasAttachedDatabases
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
		return nil, ErrInsufficientPermissionsToViewNotifier
	}

	notifier.HideSensitiveData()
	return notifier, nil
}

func (s *NotifierService) GetNotifierByID(id uuid.UUID) (*Notifier, error) {
	return s.notifierRepository.FindByID(id)
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
		return nil, ErrInsufficientPermissionsToViewNotifiers
	}

	notifiers, err := s.notifierRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, notifier := range notifiers {
		notifier.HideSensitiveData()
	}

	return notifiers, nil
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
		return ErrInsufficientPermissionsToTestNotifier
	}

	err = notifier.Send(s.fieldEncryptor, s.logger, "Test message", "This is a test message")
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
	var usingNotifier *Notifier

	if notifier.ID != uuid.Nil {
		existingNotifier, err := s.notifierRepository.FindByID(notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.WorkspaceID != notifier.WorkspaceID {
			return ErrNotifierDoesNotBelongToWorkspace
		}

		existingNotifier.Update(notifier)

		if err := existingNotifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := existingNotifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		usingNotifier = existingNotifier
	} else {
		if err := notifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		usingNotifier = notifier
	}

	return usingNotifier.Send(s.fieldEncryptor, s.logger, "Test message", "This is a test message")
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

	err = notifiedFromDb.Send(s.fieldEncryptor, s.logger, title, message)
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

func (s *NotifierService) TransferNotifierToWorkspace(
	user *users_models.User,
	notifierID uuid.UUID,
	targetWorkspaceID uuid.UUID,
	transferingWithDbID *uuid.UUID,
) error {
	existingNotifier, err := s.notifierRepository.FindByID(notifierID)
	if err != nil {
		return err
	}

	canManageSource, err := s.workspaceService.CanUserManageDBs(existingNotifier.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	canManageTarget, err := s.workspaceService.CanUserManageDBs(targetWorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	attachedDatabasesIDs, err := s.notifierDatabaseCounter.GetNotifierAttachedDatabasesIDs(
		existingNotifier.ID,
	)
	if err != nil {
		return err
	}

	if transferingWithDbID != nil {
		for _, dbID := range attachedDatabasesIDs {
			if dbID != *transferingWithDbID {
				return ErrNotifierHasOtherAttachedDatabasesCannotTransfer
			}
		}
	} else {
		if len(attachedDatabasesIDs) > 0 {
			return ErrNotifierHasAttachedDatabasesCannotTransfer
		}
	}

	sourceWorkspaceID := existingNotifier.WorkspaceID
	existingNotifier.WorkspaceID = targetWorkspaceID

	_, err = s.notifierRepository.Save(existingNotifier)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Notifier transferred: %s from workspace %s to workspace %s",
			existingNotifier.Name, sourceWorkspaceID, targetWorkspaceID),
		&user.ID,
		&targetWorkspaceID,
	)

	return nil
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
