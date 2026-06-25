package backups_core_logical

var backupRepository = &BackupRepository{}

func GetBackupRepository() *BackupRepository {
	return backupRepository
}
