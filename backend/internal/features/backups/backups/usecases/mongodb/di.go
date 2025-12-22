package usecases_mongodb

import (
	encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/logger"
)

var createMongodbBackupUsecase = &CreateMongodbBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
	return createMongodbBackupUsecase
}
