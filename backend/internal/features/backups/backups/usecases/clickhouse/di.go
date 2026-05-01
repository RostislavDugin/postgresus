package usecases_clickhouse

import (
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var createClickhouseBackupUsecase = &CreateClickhouseBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateClickhouseBackupUsecase() *CreateClickhouseBackupUsecase {
	return createClickhouseBackupUsecase
}
