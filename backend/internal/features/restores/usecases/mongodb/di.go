package usecases_mongodb

import (
	encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/util/logger"
)

var restoreMongodbBackupUsecase = &RestoreMongodbBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
}

func GetRestoreMongodbBackupUsecase() *RestoreMongodbBackupUsecase {
	return restoreMongodbBackupUsecase
}
