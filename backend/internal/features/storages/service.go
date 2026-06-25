package storages

import (
	"fmt"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
)

type StorageService struct {
	storageRepository       *StorageRepository
	workspaceService        *workspaces_services.WorkspaceService
	auditLogService         *audit_logs.AuditLogService
	fieldEncryptor          encryption.FieldEncryptor
	storageDatabaseCounters []StorageDatabaseCounter
}

func (s *StorageService) AddStorageDatabaseCounter(
	storageDatabaseCounter StorageDatabaseCounter,
) {
	s.storageDatabaseCounters = append(s.storageDatabaseCounters, storageDatabaseCounter)
}

func (s *StorageService) GetStorageAttachedDatabasesIDs(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{})
	merged := make([]uuid.UUID, 0)

	for _, counter := range s.storageDatabaseCounters {
		ids, err := counter.GetStorageAttachedDatabasesIDs(storageID)
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}

	return merged, nil
}

func (s *StorageService) IsStorageUsing(storageID uuid.UUID) (bool, error) {
	ids, err := s.GetStorageAttachedDatabasesIDs(storageID)
	if err != nil {
		return false, err
	}

	return len(ids) > 0, nil
}

func (s *StorageService) CountDatabasesForStorage(storageID uuid.UUID) (int, error) {
	ids, err := s.GetStorageAttachedDatabasesIDs(storageID)
	if err != nil {
		return 0, err
	}

	return len(ids), nil
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
		return ErrInsufficientPermissionsToManageStorage
	}

	if storage.Type == StorageTypeRclone && user.Role != users_enums.UserRoleAdmin {
		return ErrRcloneStorageRequiresAdmin
	}

	isUpdate := storage.ID != uuid.Nil

	if isUpdate {
		existingStorage, err := s.storageRepository.FindByID(storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != workspaceID {
			return ErrStorageDoesNotBelongToWorkspace
		}

		existingStorage.Update(storage)

		oldName := existingStorage.Name

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

		if oldName != existingStorage.Name {
			s.auditLogService.WriteAuditLog(
				fmt.Sprintf("Storage renamed from '%s' to '%s'", oldName, existingStorage.Name),
				&user.ID,
				&workspaceID,
			)
		} else {
			s.auditLogService.WriteAuditLog(
				fmt.Sprintf("Storage updated: %s", existingStorage.Name),
				&user.ID,
				&workspaceID,
			)
		}
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
		return ErrInsufficientPermissionsToManageStorage
	}

	attachedDatabasesIDs, err := s.GetStorageAttachedDatabasesIDs(storage.ID)
	if err != nil {
		return err
	}
	if len(attachedDatabasesIDs) > 0 {
		return ErrStorageHasAttachedDatabases
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
		return nil, ErrInsufficientPermissionsToViewStorage
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
		return nil, ErrInsufficientPermissionsToViewStorages
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
		return ErrInsufficientPermissionsToTestStorage
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
	user *users_models.User,
	storage *Storage,
) error {
	if storage.Type == StorageTypeRclone && user.Role != users_enums.UserRoleAdmin {
		return ErrRcloneStorageRequiresAdmin
	}

	var usingStorage *Storage

	if storage.ID != uuid.Nil {
		existingStorage, err := s.storageRepository.FindByID(storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != storage.WorkspaceID {
			return ErrStorageDoesNotBelongToWorkspace
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

func (s *StorageService) GetAllStorages() ([]*Storage, error) {
	return s.storageRepository.GetAllStorages()
}

func (s *StorageService) TransferStorageToWorkspace(
	user *users_models.User,
	storageID uuid.UUID,
	targetWorkspaceID uuid.UUID,
	transferingWithDbID *uuid.UUID,
) error {
	existingStorage, err := s.storageRepository.FindByID(storageID)
	if err != nil {
		return err
	}

	canManageSource, err := s.workspaceService.CanUserManageDBs(existingStorage.WorkspaceID, user)
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

	attachedDatabasesIDs, err := s.GetStorageAttachedDatabasesIDs(existingStorage.ID)
	if err != nil {
		return err
	}

	if transferingWithDbID != nil {
		for _, dbID := range attachedDatabasesIDs {
			if dbID != *transferingWithDbID {
				return ErrStorageHasOtherAttachedDatabasesCannotTransfer
			}
		}
	} else if len(attachedDatabasesIDs) > 0 {
		return ErrStorageHasAttachedDatabasesCannotTransfer
	}

	sourceWorkspaceID := existingStorage.WorkspaceID
	existingStorage.WorkspaceID = targetWorkspaceID

	_, err = s.storageRepository.Save(existingStorage)
	if err != nil {
		return err
	}

	sourceWorkspace, err := s.workspaceService.GetWorkspaceByID(sourceWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	targetWorkspace, err := s.workspaceService.GetWorkspaceByID(targetWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get target workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Storage transferred out: %s to workspace '%s'",
			existingStorage.Name, targetWorkspace.Name),
		&user.ID,
		&sourceWorkspaceID,
	)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Storage transferred in: %s from workspace '%s'",
			existingStorage.Name, sourceWorkspace.Name),
		&user.ID,
		&targetWorkspaceID,
	)

	return nil
}
