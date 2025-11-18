package storages

import (
	"errors"
	"fmt"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/encryption"

	"github.com/google/uuid"
)

type StorageService struct {
	storageRepository *StorageRepository
	workspaceService  *workspaces_services.WorkspaceService
	auditLogService   *audit_logs.AuditLogService
	fieldEncryptor    encryption.FieldEncryptor
}

func (s *StorageService) SaveStorage(
	user *users_models.User,
	workspaceID uuid.UUID,
	storage *Storage,
) error {
	canManage, err := s.workspaceService.CanUserManageDBs(workspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to manage storage in this workspace")
	}

	isUpdate := storage.ID != uuid.Nil

	if isUpdate {
		existingStorage, err := s.storageRepository.FindByID(storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != workspaceID {
			return errors.New("storage does not belong to this workspace")
		}

		existingStorage.Update(storage)

		if err := existingStorage.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := existingStorage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.storageRepository.Save(existingStorage)
		if err != nil {
			return err
		}

		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Storage updated: %s", existingStorage.Name),
			&user.ID,
			&workspaceID,
		)
	} else {
		storage.WorkspaceID = workspaceID

		if err := storage.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := storage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.storageRepository.Save(storage)
		if err != nil {
			return err
		}

		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Storage created: %s", storage.Name),
			&user.ID,
			&workspaceID,
		)
	}

	return nil
}

func (s *StorageService) DeleteStorage(
	user *users_models.User,
	storageID uuid.UUID,
) error {
	storage, err := s.storageRepository.FindByID(storageID)
	if err != nil {
		return err
	}

	canManage, err := s.workspaceService.CanUserManageDBs(storage.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to manage storage in this workspace")
	}

	err = s.storageRepository.Delete(storage)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Storage deleted: %s", storage.Name),
		&user.ID,
		&storage.WorkspaceID,
	)

	return nil
}

func (s *StorageService) GetStorage(
	user *users_models.User,
	id uuid.UUID,
) (*Storage, error) {
	storage, err := s.storageRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(storage.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view storage in this workspace")
	}

	storage.HideSensitiveData()

	return storage, nil
}

func (s *StorageService) GetStorages(
	user *users_models.User,
	workspaceID uuid.UUID,
) ([]*Storage, error) {
	canView, _, err := s.workspaceService.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view storages in this workspace")
	}

	storages, err := s.storageRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, storage := range storages {
		storage.HideSensitiveData()
	}

	return storages, nil
}

func (s *StorageService) TestStorageConnection(
	user *users_models.User,
	storageID uuid.UUID,
) error {
	storage, err := s.storageRepository.FindByID(storageID)
	if err != nil {
		return err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(storage.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canView {
		return errors.New("insufficient permissions to test storage in this workspace")
	}

	err = storage.TestConnection(s.fieldEncryptor)
	if err != nil {
		lastSaveError := err.Error()
		storage.LastSaveError = &lastSaveError
		return err
	}

	storage.LastSaveError = nil
	_, err = s.storageRepository.Save(storage)
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageService) TestStorageConnectionDirect(
	storage *Storage,
) error {
	var usingStorage *Storage

	if storage.ID != uuid.Nil {
		existingStorage, err := s.storageRepository.FindByID(storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != storage.WorkspaceID {
			return errors.New("storage does not belong to this workspace")
		}

		existingStorage.Update(storage)

		if err := existingStorage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		usingStorage = existingStorage
	} else {
		usingStorage = storage
	}

	return usingStorage.TestConnection(s.fieldEncryptor)
}

func (s *StorageService) GetStorageByID(
	id uuid.UUID,
) (*Storage, error) {
	return s.storageRepository.FindByID(id)
}

func (s *StorageService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	storages, err := s.storageRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get storages for workspace deletion: %w", err)
	}

	for _, storage := range storages {
		if err := s.storageRepository.Delete(storage); err != nil {
			return fmt.Errorf("failed to delete storage %s: %w", storage.ID, err)
		}
	}

	return nil
}
