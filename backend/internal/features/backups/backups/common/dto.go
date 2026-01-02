package common

import backups_config "databasus-backend/internal/features/backups/config"

type BackupType string

const (
	BackupTypeDefault   BackupType = "DEFAULT"   // For MySQL, MongoDB, PostgreSQL legacy (-Fc)
	BackupTypeDirectory BackupType = "DIRECTORY" // PostgreSQL directory type (-Fd)
)

type BackupMetadata struct {
	EncryptionSalt *string
	EncryptionIV   *string
	Encryption     backups_config.BackupEncryption
	Type           BackupType
}
