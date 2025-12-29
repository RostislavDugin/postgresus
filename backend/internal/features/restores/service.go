package restores

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/backups/backups"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/restores/enums"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
	"errors"
	"fmt"
	"log/slog"
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

	if err := s.validateVersionCompatibility(backupDatabase, requestDTO); err != nil {
		return err
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

	switch database.Type {
	case databases.DatabaseTypePostgres:
		if requestDTO.PostgresqlDatabase == nil {
			return errors.New("postgresql database is required")
		}
	case databases.DatabaseTypeMysql:
		if requestDTO.MysqlDatabase == nil {
			return errors.New("mysql database is required")
		}
	case databases.DatabaseTypeMariadb:
		if requestDTO.MariadbDatabase == nil {
			return errors.New("mariadb database is required")
		}
	case databases.DatabaseTypeMongodb:
		if requestDTO.MongodbDatabase == nil {
			return errors.New("mongodb database is required")
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
		Type:       database.Type,
		Postgresql: requestDTO.PostgresqlDatabase,
		Mysql:      requestDTO.MysqlDatabase,
		Mariadb:    requestDTO.MariadbDatabase,
		Mongodb:    requestDTO.MongodbDatabase,
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

func (s *RestoreService) validateVersionCompatibility(
	backupDatabase *databases.Database,
	requestDTO RestoreBackupRequest,
) error {
	// populate version
	if requestDTO.MariadbDatabase != nil {
		err := requestDTO.MariadbDatabase.PopulateVersion(
			s.logger,
			s.fieldEncryptor,
			backupDatabase.ID,
		)
		if err != nil {
			return err
		}
	}
	if requestDTO.MysqlDatabase != nil {
		err := requestDTO.MysqlDatabase.PopulateVersion(
			s.logger,
			s.fieldEncryptor,
			backupDatabase.ID,
		)
		if err != nil {
			return err
		}
	}
	if requestDTO.PostgresqlDatabase != nil {
		err := requestDTO.PostgresqlDatabase.PopulateVersion(
			s.logger,
			s.fieldEncryptor,
			backupDatabase.ID,
		)
		if err != nil {
			return err
		}
	}
	if requestDTO.MongodbDatabase != nil {
		err := requestDTO.MongodbDatabase.PopulateVersion(
			s.logger,
			s.fieldEncryptor,
			backupDatabase.ID,
		)
		if err != nil {
			return err
		}
	}

	switch backupDatabase.Type {
	case databases.DatabaseTypePostgres:
		if requestDTO.PostgresqlDatabase == nil {
			return errors.New("postgresql database configuration is required for restore")
		}
		if tools.IsBackupDbVersionHigherThanRestoreDbVersion(
			backupDatabase.Postgresql.Version,
			requestDTO.PostgresqlDatabase.Version,
		) {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore PG 15 backup to PG 15, 16 or higher. But cannot restore to 14 and lower`)
		}
	case databases.DatabaseTypeMysql:
		if requestDTO.MysqlDatabase == nil {
			return errors.New("mysql database configuration is required for restore")
		}
		if tools.IsMysqlBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mysql.Version,
			requestDTO.MysqlDatabase.Version,
		) {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MySQL 8.0 backup to MySQL 8.0, 8.4 or higher. But cannot restore to 5.7`)
		}
	case databases.DatabaseTypeMariadb:
		if requestDTO.MariadbDatabase == nil {
			return errors.New("mariadb database configuration is required for restore")
		}
		if tools.IsMariadbBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mariadb.Version,
			requestDTO.MariadbDatabase.Version,
		) {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MariaDB 10.11 backup to MariaDB 10.11, 11.4 or higher. But cannot restore to 10.6`)
		}
	case databases.DatabaseTypeMongodb:
		if requestDTO.MongodbDatabase == nil {
			return errors.New("mongodb database configuration is required for restore")
		}
		if tools.IsMongodbBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mongodb.Version,
			requestDTO.MongodbDatabase.Version,
		) {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MongoDB 6.0 backup to MongoDB 6.0, 7.0 or higher. But cannot restore to 5.0`)
		}
	}
	return nil
}
