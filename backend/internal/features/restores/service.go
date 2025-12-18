package restores

import (
	"errors"
	"fmt"
	"log/slog"
	audit_logs "postgresus-backend/internal/features/audit_logs"
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/restores/enums"
	"postgresus-backend/internal/features/restores/models"
	"postgresus-backend/internal/features/restores/usecases"
	"postgresus-backend/internal/features/storages"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/tools"
	"time"

	"github.com/google/uuid"
)

type RestoreService struct {
	backupService        *backups.BackupService
	restoreRepository    *RestoreRepository
	storageService       *storages.StorageService
	backupConfigService  *backups_config.BackupConfigService
	restoreBackupUsecase *usecases.RestoreBackupUsecase
	databaseService      *databases.DatabaseService
	logger               *slog.Logger
	workspaceService     *workspaces_services.WorkspaceService
	auditLogService      *audit_logs.AuditLogService
	fieldEncryptor       encryption.FieldEncryptor
}

func (s *RestoreService) OnBeforeBackupRemove(backup *backups.Backup) error {
	restores, err := s.restoreRepository.FindByBackupID(backup.ID)
	if err != nil {
		return err
	}

	for _, restore := range restores {
		if restore.Status == enums.RestoreStatusInProgress {
			return errors.New("restore is in progress, backup cannot be removed")
		}
	}

	for _, restore := range restores {
		if err := s.restoreRepository.DeleteByID(restore.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *RestoreService) GetRestores(
	user *users_models.User,
	backupID uuid.UUID,
) ([]*models.Restore, error) {
	backup, err := s.backupService.GetBackup(backupID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot get restores for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		*database.WorkspaceID,
		user,
	)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access restores for this backup")
	}

	return s.restoreRepository.FindByBackupID(backupID)
}

func (s *RestoreService) RestoreBackupWithAuth(
	user *users_models.User,
	backupID uuid.UUID,
	requestDTO RestoreBackupRequest,
) error {
	backup, err := s.backupService.GetBackup(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot restore backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		*database.WorkspaceID,
		user,
	)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to restore this backup")
	}

	backupDatabase, err := s.databaseService.GetDatabase(user, backup.DatabaseID)
	if err != nil {
		return err
	}

	if tools.IsBackupDbVersionHigherThanRestoreDbVersion(
		backupDatabase.Postgresql.Version,
		requestDTO.PostgresqlDatabase.Version,
	) {
		return errors.New(`backup database version is higher than restore database version. ` +
			`Should be restored to the same version as the backup database or higher. ` +
			`For example, you can restore PG 15 backup to PG 15, 16 or higher. But cannot restore to 14 and lower`)
	}

	go func() {
		if err := s.RestoreBackup(backup, requestDTO); err != nil {
			s.logger.Error("Failed to restore backup", "error", err)
		}
	}()

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf(
			"Database restored from backup %s for database: %s",
			backupID.String(),
			database.Name,
		),
		&user.ID,
		database.WorkspaceID,
	)

	return nil
}

func (s *RestoreService) RestoreBackup(
	backup *backups.Backup,
	requestDTO RestoreBackupRequest,
) error {
	if backup.Status != backups.BackupStatusCompleted {
		return errors.New("backup is not completed")
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.Type == databases.DatabaseTypePostgres {
		if requestDTO.PostgresqlDatabase == nil {
			return errors.New("postgresql database is required")
		}
	}

	restore := models.Restore{
		ID:     uuid.New(),
		Status: enums.RestoreStatusInProgress,

		BackupID: backup.ID,
		Backup:   backup,

		CreatedAt:         time.Now().UTC(),
		RestoreDurationMs: 0,

		FailMessage: nil,
	}

	// Save the restore first
	if err := s.restoreRepository.Save(&restore); err != nil {
		return err
	}

	// Save the restore again to include the postgresql database
	if err := s.restoreRepository.Save(&restore); err != nil {
		return err
	}

	storage, err := s.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return err
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(
		database.ID,
	)
	if err != nil {
		return err
	}

	start := time.Now().UTC()

	restoringToDB := &databases.Database{
		Postgresql: requestDTO.PostgresqlDatabase,
	}

	if err := restoringToDB.PopulateVersionIfEmpty(s.logger, s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to auto-detect database version: %w", err)
	}

	isExcludeExtensions := false
	if requestDTO.PostgresqlDatabase != nil {
		isExcludeExtensions = requestDTO.PostgresqlDatabase.IsExcludeExtensions
	}

	err = s.restoreBackupUsecase.Execute(
		backupConfig,
		restore,
		database,
		restoringToDB,
		backup,
		storage,
		isExcludeExtensions,
	)
	if err != nil {
		errMsg := err.Error()
		restore.FailMessage = &errMsg
		restore.Status = enums.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := s.restoreRepository.Save(&restore); err != nil {
			return err
		}

		return err
	}

	restore.Status = enums.RestoreStatusCompleted
	restore.RestoreDurationMs = time.Since(start).Milliseconds()

	if err := s.restoreRepository.Save(&restore); err != nil {
		return err
	}

	return nil
}
