package usecases_mongodb

import (
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var createMongodbBackupUsecase = &CreateMongodbBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
	return createMongodbBackupUsecase
}
