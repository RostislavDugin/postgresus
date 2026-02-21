package backups_config

import (
	"errors"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	plans "databasus-backend/internal/features/plan"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"

	"github.com/google/uuid"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	storageService         *storages.StorageService
	notifierService        *notifiers.NotifierService
	workspaceService       *workspaces_services.WorkspaceService
	databasePlanService    *plans.DatabasePlanService

	dbStorageChangeListener BackupConfigStorageChangeListener
}

func (s *BackupConfigService) SetDatabaseStorageChangeListener(
	dbStorageChangeListener BackupConfigStorageChangeListener,
) {
	s.dbStorageChangeListener = dbStorageChangeListener
}

func (s *BackupConfigService) GetStorageAttachedDatabasesIDs(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	databasesIDs, err := s.backupConfigRepository.GetDatabasesIDsByStorageID(storageID)
	if err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

func (s *BackupConfigService) CreateBackupConfig(
	user *users_models.User,
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	plan, err := s.databasePlanService.GetDatabasePlan(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if err := backupConfig.Validate(plan); err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabase(user, backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot create backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create backup configuration")
	}

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID && !storage.IsSystem {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	return s.backupConfigRepository.Save(backupConfig)
}

func (s *BackupConfigService) UpdateBackupConfig(
	user *users_models.User,
	id uuid.UUID,
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	existingConfig, err := s.backupConfigRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	if existingConfig == nil {
		return nil, errors.New("backup config not found")
	}

	plan, err := s.databasePlanService.GetDatabasePlan(existingConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if err := backupConfig.Validate(plan); err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabase(user, existingConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot update backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update backup configuration")
	}

	// Preserve the ID and DatabaseID
	backupConfig.ID = id
	backupConfig.DatabaseID = existingConfig.DatabaseID

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID && !storage.IsSystem {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	// Check if storage is changing
	if s.dbStorageChangeListener != nil &&
		backupConfig.Storage != nil &&
		!storageIDsEqual(existingConfig.StorageID, &backupConfig.Storage.ID) {
		if err := s.dbStorageChangeListener.OnBeforeBackupsStorageChange(
			backupConfig.DatabaseID,
		); err != nil {
			return nil, err
		}
	}

	return s.backupConfigRepository.Save(backupConfig)
}

func (s *BackupConfigService) DeleteBackupConfig(
	user *users_models.User,
	id uuid.UUID,
) error {
	existingConfig, err := s.backupConfigRepository.FindByID(id)
	if err != nil {
		return err
	}

	if existingConfig == nil {
		return errors.New("backup config not found")
	}

	database, err := s.databaseService.GetDatabase(user, existingConfig.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot delete backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete backup configuration")
	}

	return s.backupConfigRepository.Delete(id)
}

func (s *BackupConfigService) GetBackupConfigByID(id uuid.UUID) (*BackupConfig, error) {
	return s.backupConfigRepository.FindByID(id)
}

func (s *BackupConfigService) GetBackupConfigsByDatabaseIDWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) ([]*BackupConfig, error) {
	_, err := s.databaseService.GetDatabase(user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.GetBackupConfigsByDatabaseID(databaseID)
}

func (s *BackupConfigService) GetDatabasePlan(
	user *users_models.User,
	databaseID uuid.UUID,
) (*plans.DatabasePlan, error) {
	_, err := s.databaseService.GetDatabase(user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.databasePlanService.GetDatabasePlan(databaseID)
}

func (s *BackupConfigService) GetBackupConfigsByDatabaseID(
	databaseID uuid.UUID,
) ([]*BackupConfig, error) {
	configs, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if len(configs) == 0 {
		err = s.initializeDefaultConfig(databaseID)
		if err != nil {
			return nil, err
		}

		return s.backupConfigRepository.FindByDatabaseID(databaseID)
	}

	return configs, nil
}

func (s *BackupConfigService) IsStorageUsing(
	user *users_models.User,
	storageID uuid.UUID,
) (bool, error) {
	_, err := s.storageService.GetStorage(user, storageID)
	if err != nil {
		return false, err
	}

	return s.backupConfigRepository.IsStorageUsing(storageID)
}

func (s *BackupConfigService) CountDatabasesForStorage(
	user *users_models.User,
	storageID uuid.UUID,
) (int, error) {
	_, err := s.storageService.GetStorage(user, storageID)
	if err != nil {
		return 0, err
	}

	databaseIDs, err := s.backupConfigRepository.GetDatabasesIDsByStorageID(storageID)
	if err != nil {
		return 0, err
	}

	return len(databaseIDs), nil
}

func (s *BackupConfigService) GetBackupConfigsWithEnabledBackups() ([]*BackupConfig, error) {
	return s.backupConfigRepository.GetWithEnabledBackups()
}

func (s *BackupConfigService) OnDatabaseCopied(originalDatabaseID, newDatabaseID uuid.UUID) {
	originalConfigs, err := s.GetBackupConfigsByDatabaseID(originalDatabaseID)
	if err != nil {
		return
	}

	for _, originalConfig := range originalConfigs {
		newConfig := originalConfig.Copy(newDatabaseID)
		_, _ = s.backupConfigRepository.Save(newConfig)
	}
}

func (s *BackupConfigService) CreateDisabledBackupConfig(databaseID uuid.UUID) error {
	return s.initializeDefaultConfig(databaseID)
}

func (s *BackupConfigService) initializeDefaultConfig(
	databaseID uuid.UUID,
) error {
	plan, err := s.databasePlanService.GetDatabasePlan(databaseID)
	if err != nil {
		return err
	}

	timeOfDay := "04:00"

	_, err = s.backupConfigRepository.Save(&BackupConfig{
		DatabaseID:            databaseID,
		Name:                  "Default",
		IsBackupsEnabled:      false,
		RetentionPolicyType:   RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   plan.MaxStoragePeriod,
		MaxBackupSizeMB:       plan.MaxBackupSizeMB,
		MaxBackupsTotalSizeMB: plan.MaxBackupsTotalSizeMB,
		BackupInterval: &intervals.Interval{
			Interval:  intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          BackupEncryptionNone,
	})

	return err
}

func (s *BackupConfigService) TransferDatabaseToWorkspace(
	user *users_models.User,
	databaseID uuid.UUID,
	request *TransferDatabaseRequest,
) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return ErrDatabaseHasNoWorkspace
	}

	canManageSource, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	canManageTarget, err := s.workspaceService.CanUserManageDBs(request.TargetWorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	if err := s.validateTargetNotifiers(request); err != nil {
		return err
	}

	backupConfigs, err := s.GetBackupConfigsByDatabaseID(databaseID)
	if err != nil {
		return err
	}

	if request.IsTransferWithNotifiers {
		s.transferNotifiers(user, database, request.TargetWorkspaceID)
	}

	if request.IsTransferWithStorage {
		for _, backupConfig := range backupConfigs {
			if backupConfig.StorageID == nil {
				return ErrDatabaseHasNoStorage
			}

			attachedDatabasesIDs, err := s.GetStorageAttachedDatabasesIDs(*backupConfig.StorageID)
			if err != nil {
				return err
			}

			for _, dbID := range attachedDatabasesIDs {
				if dbID != databaseID {
					return ErrStorageHasOtherAttachedDatabases
				}
			}

			err = s.storageService.TransferStorageToWorkspace(
				user,
				*backupConfig.StorageID,
				request.TargetWorkspaceID,
				&databaseID,
			)
			if err != nil {
				return err
			}
		}
	} else if request.TargetStorageID != nil {
		targetStorage, err := s.storageService.GetStorageByID(*request.TargetStorageID)
		if err != nil {
			return err
		}

		if targetStorage.WorkspaceID != request.TargetWorkspaceID {
			return ErrTargetStorageNotInTargetWorkspace
		}

		for _, backupConfig := range backupConfigs {
			backupConfig.StorageID = request.TargetStorageID
			backupConfig.Storage = targetStorage

			_, err = s.backupConfigRepository.Save(backupConfig)
			if err != nil {
				return err
			}
		}
	} else {
		return ErrTargetStorageNotSpecified
	}

	err = s.databaseService.TransferDatabaseToWorkspace(databaseID, request.TargetWorkspaceID)
	if err != nil {
		return err
	}

	if len(request.TargetNotifierIDs) > 0 {
		err = s.assignTargetNotifiers(databaseID, request.TargetNotifierIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupConfigService) transferNotifiers(
	user *users_models.User,
	database *databases.Database,
	targetWorkspaceID uuid.UUID,
) {
	for _, notifier := range database.Notifiers {
		_ = s.notifierService.TransferNotifierToWorkspace(
			user,
			notifier.ID,
			targetWorkspaceID,
			&database.ID,
		)
	}
}

func (s *BackupConfigService) validateTargetNotifiers(request *TransferDatabaseRequest) error {
	for _, notifierID := range request.TargetNotifierIDs {
		notifier, err := s.notifierService.GetNotifierByID(notifierID)
		if err != nil {
			return err
		}

		if notifier.WorkspaceID != request.TargetWorkspaceID {
			return ErrTargetNotifierNotInTargetWorkspace
		}
	}
	return nil
}

func (s *BackupConfigService) assignTargetNotifiers(
	databaseID uuid.UUID,
	notifierIDs []uuid.UUID,
) error {
	targetNotifiers := make([]notifiers.Notifier, 0, len(notifierIDs))

	for _, notifierID := range notifierIDs {
		notifier, err := s.notifierService.GetNotifierByID(notifierID)
		if err != nil {
			return err
		}

		targetNotifiers = append(targetNotifiers, *notifier)
	}

	return s.databaseService.UpdateDatabaseNotifiers(databaseID, targetNotifiers)
}

func storageIDsEqual(id1, id2 *uuid.UUID) bool {
	if id1 == nil && id2 == nil {
		return true
	}
	if id1 == nil || id2 == nil {
		return false
	}
	return *id1 == *id2
}
