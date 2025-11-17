package usecases_postgresql

import backups_config "postgresus-backend/internal/features/backups/config"

type EncryptionMetadata struct {
	Salt       string
	IV         string
	Encryption backups_config.BackupEncryption
}

type BackupMetadata struct {
	EncryptionSalt *string
	EncryptionIV   *string
	Encryption     backups_config.BackupEncryption
}
