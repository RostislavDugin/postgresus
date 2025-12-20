package common

import backups_config "postgresus-backend/internal/features/backups/config"

type BackupMetadata struct {
	EncryptionSalt *string
	EncryptionIV   *string
	Encryption     backups_config.BackupEncryption
}
