package backups_config_physical

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	storageService         *storages.StorageService
	notifierService        *notifiers.NotifierService
	workspaceService       *workspaces_services.WorkspaceService

	dbStorageChangeListener BackupConfigStorageChangeListener
	configChangeListener    BackupConfigChangeListener
}

func (s *BackupConfigService) SetDatabaseStorageChangeListener(
	dbStorageChangeListener BackupConfigStorageChangeListener,
) {
	s.dbStorageChangeListener = dbStorageChangeListener
}

func (s *BackupConfigService) SetBackupConfigChangeListener(
	configChangeListener BackupConfigChangeListener,
) {
	s.configChangeListener = configChangeListener
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
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
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

	if database.PostgresqlPhysical == nil {
		return nil, errors.New(
			"physical backup config requires the owning database to be of type POSTGRES_PHYSICAL",
		)
	}

	backupConfig.PostgresqlPhysical = database.PostgresqlPhysical

	if err := backupConfig.Validate(); err != nil {
		return nil, err
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
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	if backupConfig.PostgresqlPhysical == nil {
		database, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
		if err != nil {
			return nil, err
		}

		if database.PostgresqlPhysical == nil {
			return nil, errors.New(
				"physical backup config requires the owning database to be of type POSTGRES_PHYSICAL",
			)
		}

		backupConfig.PostgresqlPhysical = database.PostgresqlPhysical
	}

	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	existingConfig, err := s.GetBackupConfigByDbId(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil {
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

	savedConfig, err := s.backupConfigRepository.Save(backupConfig)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil && s.configChangeListener != nil {
		s.notifyConfigChangeIfNeeded(existingConfig, backupConfig)
	}

	return savedConfig, nil
}

func (s *BackupConfigService) GetBackupConfigByDbIdWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	_, err := s.databaseService.GetDatabase(user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.GetBackupConfigByDbId(databaseID)
}

func (s *BackupConfigService) GetBackupConfigByDbId(
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	config, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(databaseID); err != nil {
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

	return s.storageService.IsStorageUsing(storageID)
}

func (s *BackupConfigService) CountDatabasesForStorage(
	user *users_models.User,
	storageID uuid.UUID,
) (int, error) {
	_, err := s.storageService.GetStorage(user, storageID)
	if err != nil {
		return 0, err
	}

	return s.storageService.CountDatabasesForStorage(storageID)
}

func (s *BackupConfigService) GetBackupConfigsWithEnabledBackups() (
	[]*PhysicalBackupConfig,
	error,
) {
	return s.backupConfigRepository.GetWithEnabledBackups()
}

func (s *BackupConfigService) RequestFullBackupNow(databaseID uuid.UUID) error {
	return s.backupConfigRepository.RequestFullBackupNow(databaseID)
}

func (s *BackupConfigService) ClearFullBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	return s.backupConfigRepository.ClearFullBackupRequest(databaseID, requestedAt)
}

func (s *BackupConfigService) RequestIncrementalBackupNow(databaseID uuid.UUID) error {
	return s.backupConfigRepository.RequestIncrementalBackupNow(databaseID)
}

func (s *BackupConfigService) ClearIncrementalBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	return s.backupConfigRepository.ClearIncrementalBackupRequest(databaseID, requestedAt)
}

func (s *BackupConfigService) OnDatabaseCopied(originalDatabaseID, newDatabaseID uuid.UUID) {
	originalConfig, err := s.backupConfigRepository.FindByDatabaseID(originalDatabaseID)
	if err != nil || originalConfig == nil {
		return
	}

	newConfig := originalConfig.Copy(newDatabaseID)

	_, _ = s.SaveBackupConfig(newConfig)
}

func (s *BackupConfigService) CreateDisabledBackupConfig(databaseID uuid.UUID) error {
	return s.initializeDefaultConfig(databaseID)
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

	switch {
	case request.IsTransferWithStorage:
		if backupConfig.StorageID == nil {
			return ErrDatabaseHasNoStorage
		}

		attachedDatabasesIDs, err := s.storageService.GetStorageAttachedDatabasesIDs(
			*backupConfig.StorageID,
		)
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
	case request.TargetStorageID != nil:
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
	default:
		return ErrTargetStorageNotSpecified
	}

	err = s.databaseService.TransferDatabaseToWorkspace(databaseID, request.TargetWorkspaceID)
	if err != nil {
		return err
	}

	if len(request.TargetNotifierIDs) > 0 {
		if err := s.assignTargetNotifiers(databaseID, request.TargetNotifierIDs); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupConfigService) initializeDefaultConfig(databaseID uuid.UUID) error {
	timeOfDay := "04:00"

	_, err := s.backupConfigRepository.Save(&PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: false,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Retention: RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  7,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
			NotificationChainBroken,
			NotificationWalGap,
		},
		Encryption: "NONE",
	})

	return err
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

// notifyConfigChangeIfNeeded fires the config-change listener only on the two
// transitions that must stand work down: backups disabled or BackupType
// demoted away from WAL_STREAM.
func (s *BackupConfigService) notifyConfigChangeIfNeeded(oldConfig, newConfig *PhysicalBackupConfig) {
	disabled := oldConfig.IsBackupsEnabled && !newConfig.IsBackupsEnabled
	demotedFromWalStream := isWalStream(oldConfig) && !isWalStream(newConfig)

	if disabled || demotedFromWalStream {
		s.configChangeListener.OnBackupConfigChanged(oldConfig, newConfig)
	}
}

func isWalStream(backupConfig *PhysicalBackupConfig) bool {
	return backupConfig.PostgresqlPhysical != nil &&
		backupConfig.PostgresqlPhysical.BackupType == postgresql_physical.BackupTypeFullIncrementalAndWalStream
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
