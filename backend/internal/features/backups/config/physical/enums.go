package backups_config_physical

type BackupNotificationType string

const (
	NotificationBackupSuccess BackupNotificationType = "BACKUP_SUCCESS"
	NotificationBackupFailed  BackupNotificationType = "BACKUP_FAILED"
	NotificationChainBroken   BackupNotificationType = "CHAIN_BROKEN"
	NotificationWalGap        BackupNotificationType = "WAL_GAP"
)
