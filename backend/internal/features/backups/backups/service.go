package backups

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"

	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/backups/backups/backuping"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"

	"github.com/google/uuid"
)

type BackupService struct {
	databaseService     *databases.DatabaseService
	storageService      *storages.StorageService
	backupRepository    *backups_core.BackupRepository
	notifierService     *notifiers.NotifierService
	notificationSender  backups_core.NotificationSender
	backupConfigService *backups_config.BackupConfigService
	secretKeyService    *encryption_secrets.SecretKeyService
	fieldEncryptor      util_encryption.FieldEncryptor

	createBackupUseCase backups_core.CreateBackupUsecase

	logger *slog.Logger

	backupRemoveListeners []backups_core.BackupRemoveListener

	workspaceService       *workspaces_services.WorkspaceService
	auditLogService        *audit_logs.AuditLogService
	taskCancelManager      *task_cancellation.TaskCancelManager
	downloadTokenService   *backups_download.DownloadTokenService
	backupSchedulerService *backuping.BackupsScheduler
	backupCleaner          *backuping.BackupCleaner
}

func (s *BackupService) AddBackupRemoveListener(listener backups_core.BackupRemoveListener) {
	s.backupRemoveListeners = append(s.backupRemoveListeners, listener)
}

func (s *BackupService) OnBeforeBackupsStorageChange(databaseID uuid.UUID) error {
	err := s.deleteDbBackups(databaseID)
	if err != nil {
		return err
	}

	return nil
}

func (s *BackupService) OnBeforeDatabaseRemove(databaseID uuid.UUID) error {
	err := s.deleteDbBackups(databaseID)
	if err != nil {
		return err
	}

	return nil
}

func (s *BackupService) MakeBackupWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot create backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to create backup for this database")
	}

	configs, err := s.backupConfigService.GetBackupConfigsByDatabaseID(database.ID)
	if err != nil {
		return err
	}
	for _, config := range configs {
		s.backupSchedulerService.StartBackup(database, config, true)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup manually initiated for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return nil
}

func (s *BackupService) GetBackups(
	user *users_models.User,
	databaseID uuid.UUID,
	limit, offset int,
) (*GetBackupsResponse, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot get backups for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access backups for this database")
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	backups, err := s.backupRepository.FindByDatabaseIDWithPagination(databaseID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.backupRepository.CountByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	return &GetBackupsResponse{
		Backups: backups,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *BackupService) DeleteBackup(
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot delete backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete backup for this database")
	}

	if backup.Status == backups_core.BackupStatusInProgress {
		return errors.New("backup is in progress")
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup deleted for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return s.backupCleaner.DeleteBackup(backup)
}

func (s *BackupService) GetBackup(backupID uuid.UUID) (*backups_core.Backup, error) {
	return s.backupRepository.FindByID(backupID)
}

func (s *BackupService) CancelBackup(
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot cancel backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to cancel backup for this database")
	}

	if backup.Status != backups_core.BackupStatusInProgress {
		return errors.New("backup is not in progress")
	}

	if err := s.taskCancelManager.CancelTask(backupID); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup cancelled for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return nil
}

func (s *BackupService) GetBackupFile(
	user *users_models.User,
	backupID uuid.UUID,
) (io.ReadCloser, *backups_core.Backup, *databases.Database, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, nil, nil, err
	}

	if database.WorkspaceID == nil {
		return nil, nil, nil, errors.New("cannot download backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		*database.WorkspaceID,
		user,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if !canAccess {
		return nil, nil, nil, errors.New(
			"insufficient permissions to download backup for this database",
		)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup file downloaded for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	reader, err := s.getBackupReader(backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	return reader, backup, database, nil
}

func (s *BackupService) deleteDbBackups(databaseID uuid.UUID) error {
	dbBackupsInProgress, err := s.backupRepository.FindByDatabaseIdAndStatus(
		databaseID,
		backups_core.BackupStatusInProgress,
	)
	if err != nil {
		return err
	}

	if len(dbBackupsInProgress) > 0 {
		return errors.New("backup is in progress, storage cannot be removed")
	}

	dbBackups, err := s.backupRepository.FindByDatabaseID(
		databaseID,
	)
	if err != nil {
		return err
	}

	for _, dbBackup := range dbBackups {
		err := s.backupCleaner.DeleteBackup(dbBackup)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetBackupReader returns a reader for the backup file
// If encrypted, wraps with DecryptionReader
func (s *BackupService) getBackupReader(backupID uuid.UUID) (io.ReadCloser, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup: %w", err)
	}

	storage, err := s.storageService.GetStorageByID(backup.StorageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileReader, err := storage.GetFile(s.fieldEncryptor, backup.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file: %w", err)
	}

	// If not encrypted, return raw reader
	if backup.Encryption == backups_config.BackupEncryptionNone {
		s.logger.Info("Returning non-encrypted backup", "backupId", backupID)
		return fileReader, nil
	}

	// Decrypt on-the-fly for encrypted backups
	if backup.Encryption != backups_config.BackupEncryptionEncrypted {
		if err := fileReader.Close(); err != nil {
			s.logger.Error("Failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("unsupported encryption type: %s", backup.Encryption)
	}

	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		if err := fileReader.Close(); err != nil {
			s.logger.Error("Failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("backup marked as encrypted but missing encryption metadata")
	}

	// Get master key
	masterKey, err := s.secretKeyService.GetSecretKey()
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	// Decode salt and IV
	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	// Wrap with decrypting reader
	decryptionReader, err := encryption.NewDecryptionReader(
		fileReader,
		masterKey,
		backup.ID,
		salt,
		iv,
	)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			s.logger.Error("Failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create decrypting reader: %w", err)
	}

	s.logger.Info("Returning encrypted backup with decryption", "backupId", backupID)

	return &DecryptionReaderCloser{
		DecryptionReader: decryptionReader,
		BaseReader:       fileReader,
	}, nil
}

func (s *BackupService) GenerateDownloadToken(
	user *users_models.User,
	backupID uuid.UUID,
) (*backups_download.GenerateDownloadTokenResponse, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot download backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to download backup for this database")
	}

	token, err := s.downloadTokenService.Generate(backupID, user.ID)
	if err != nil {
		return nil, err
	}

	filename := s.generateBackupFilename(backup, database)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Download token generated for backup of database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return &backups_download.GenerateDownloadTokenResponse{
		Token:    token,
		Filename: filename,
		BackupID: backupID,
	}, nil
}

func (s *BackupService) ValidateDownloadToken(
	token string,
) (*backups_download.DownloadToken, *backups_download.RateLimiter, error) {
	return s.downloadTokenService.ValidateAndConsume(token)
}

func (s *BackupService) GetBackupFileWithoutAuth(
	backupID uuid.UUID,
) (io.ReadCloser, *backups_core.Backup, *databases.Database, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, nil, nil, err
	}

	reader, err := s.getBackupReader(backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	return reader, backup, database, nil
}

func (s *BackupService) WriteAuditLogForDownload(
	userID uuid.UUID,
	backup *backups_core.Backup,
	database *databases.Database,
) {
	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup file downloaded for database: %s", database.Name),
		&userID,
		database.WorkspaceID,
	)
}

func (s *BackupService) RefreshDownloadLock(userID uuid.UUID) {
	s.downloadTokenService.RefreshDownloadLock(userID)
}

func (s *BackupService) ReleaseDownloadLock(userID uuid.UUID) {
	s.downloadTokenService.ReleaseDownloadLock(userID)
}

func (s *BackupService) IsDownloadInProgress(userID uuid.UUID) bool {
	return s.downloadTokenService.IsDownloadInProgress(userID)
}

func (s *BackupService) UnregisterDownload(userID uuid.UUID) {
	s.downloadTokenService.UnregisterDownload(userID)
}

func (s *BackupService) generateBackupFilename(
	backup *backups_core.Backup,
	database *databases.Database,
) string {
	timestamp := backup.CreatedAt.Format("2006-01-02_15-04-05")
	safeName := files_utils.SanitizeFilename(database.Name)
	extension := s.getBackupExtension(database.Type)
	return fmt.Sprintf("%s_backup_%s%s", safeName, timestamp, extension)
}

func (s *BackupService) getBackupExtension(dbType databases.DatabaseType) string {
	switch dbType {
	case databases.DatabaseTypeMysql, databases.DatabaseTypeMariadb:
		return ".sql.zst"
	case databases.DatabaseTypePostgres:
		return ".dump"
	case databases.DatabaseTypeMongodb:
		return ".archive"
	default:
		return ".backup"
	}
}
