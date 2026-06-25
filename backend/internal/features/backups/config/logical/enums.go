package backups_config_logical

type BackupNotificationType string

const (
	NotificationBackupFailed  BackupNotificationType = "BACKUP_FAILED"
	NotificationBackupSuccess BackupNotificationType = "BACKUP_SUCCESS"
)

type RetentionPolicyType string

const (
	RetentionPolicyTypeTimePeriod RetentionPolicyType = "TIME_PERIOD"
	RetentionPolicyTypeCount      RetentionPolicyType = "COUNT"
	RetentionPolicyTypeGFS        RetentionPolicyType = "GFS"
)
