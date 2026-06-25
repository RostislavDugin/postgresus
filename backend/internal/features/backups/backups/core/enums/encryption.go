package backups_core_enums

type BackupEncryption string

const (
	BackupEncryptionNone      BackupEncryption = "NONE"
	BackupEncryptionEncrypted BackupEncryption = "ENCRYPTED"
)
