package common

import backups_config "databasus-backend/internal/features/backups/config"

type BackupMetadata struct {
	EncryptionSalt *string
	EncryptionIV   *string
	Encryption     backups_config.BackupEncryption
}
