package backups_config

import (
	"errors"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/period"

	"github.com/google/uuid"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	storageService         *storages.StorageService
	notifierService        *notifiers.NotifierService
	workspaceService       *workspaces_services.WorkspaceService

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

func (s *BackupConfigService) SaveBackupConfigWithAuth(
	user *users_models.User,
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabase(user, backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot save backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to modify backup configuration")
	}

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	return s.SaveBackupConfig(backupConfig)
}

func (s *BackupConfigService) SaveBackupConfig(
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	// Check if there's an existing backup config for this database
	existingConfig, err := s.GetBackupConfigByDbId(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil {
		// If storage is changing, notify the listener
		if s.dbStorageChangeListener != nil &&
			backupConfig.Storage != nil &&
			!storageIDsEqual(existingConfig.StorageID, &backupConfig.Storage.ID) {
			if err := s.dbStorageChangeListener.OnBeforeBackupsStorageChange(
				backupConfig.DatabaseID,
			); err != nil {
				return nil, err
			}
		}
	}

	return s.backupConfigRepository.Save(backupConfig)
}

func (s *BackupConfigService) GetBackupConfigByDbIdWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) (*BackupConfig, error) {
	_, err := s.databaseService.GetDatabase(user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.GetBackupConfigByDbId(databaseID)
}

func (s *BackupConfigService) GetBackupConfigByDbId(
	databaseID uuid.UUID,
) (*BackupConfig, error) {
	config, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		err = s.initializeDefaultConfig(databaseID)
		if err != nil {
			return nil, err
		}

		return s.backupConfigRepository.FindByDatabaseID(databaseID)
	}

	return config, nil
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
	originalConfig, err := s.GetBackupConfigByDbId(originalDatabaseID)
	if err != nil {
		return
	}

	newConfig := originalConfig.Copy(newDatabaseID)

	_, err = s.SaveBackupConfig(newConfig)
	if err != nil {
		return
	}
}

func (s *BackupConfigService) CreateDisabledBackupConfig(databaseID uuid.UUID) error {
	return s.initializeDefaultConfig(databaseID)
}

func (s *BackupConfigService) initializeDefaultConfig(
	databaseID uuid.UUID,
) error {
	timeOfDay := "04:00"

	_, err := s.backupConfigRepository.Save(&BackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: false,
		StorePeriod:      period.PeriodWeek,
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

	backupConfig, err := s.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return err
	}

	if request.IsTransferWithNotifiers {
		s.transferNotifiers(user, database, request.TargetWorkspaceID)
	}

	if request.IsTransferWithStorage {
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
	} else if request.TargetStorageID != nil {
		targetStorage, err := s.storageService.GetStorageByID(*request.TargetStorageID)
		if err != nil {
			return err
		}

		if targetStorage.WorkspaceID != request.TargetWorkspaceID {
			return ErrTargetStorageNotInTargetWorkspace
		}

		backupConfig.StorageID = request.TargetStorageID
		backupConfig.Storage = targetStorage

		_, err = s.backupConfigRepository.Save(backupConfig)
		if err != nil {
			return err
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
