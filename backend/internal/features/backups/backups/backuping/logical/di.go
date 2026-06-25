package backuping_logical

import (
	"sync/atomic"
	"time"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	usecases_logical "databasus-backend/internal/features/backups/backups/usecases/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var backupRepository = &backups_core_logical.BackupRepository{}

var taskCancelManager = tasks_cancellation.GetTaskCancelManager()

var backupCleaner = &BackupCleaner{
	backupRepository,
	storages.GetStorageService(),
	backups_config_logical.GetBackupConfigService(),
	encryption.GetFieldEncryptor(),
	logger.GetLogger(),
	[]backups_core_logical.BackupRemoveListener{},
	atomic.Bool{},
}

var backuper = &Backuper{
	databases.GetDatabaseService(),
	encryption.GetFieldEncryptor(),
	workspaces_services.GetWorkspaceService(),
	backupRepository,
	backups_config_logical.GetBackupConfigService(),
	storages.GetStorageService(),
	notifiers.GetNotifierService(),
	taskCancelManager,
	logger.GetLogger(),
	usecases_logical.GetCreateBackupUsecase(),
}

var backupsScheduler = &BackupsScheduler{
	backupRepository,
	backups_config_logical.GetBackupConfigService(),
	taskCancelManager,
	databases.GetDatabaseService(),
	time.Now().UTC(),
	logger.GetLogger(),
	backuper,
	[]backups_core_logical.BackupCompletionListener{},
	atomic.Bool{},
	atomic.Bool{},
}

func GetBackupsScheduler() *BackupsScheduler {
	return backupsScheduler
}

func GetBackuper() *Backuper {
	return backuper
}

func GetBackupCleaner() *BackupCleaner {
	return backupCleaner
}
