package usecases_mongodb

import (
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var restoreMongodbBackupUsecase = &RestoreMongodbBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
}

func GetRestoreMongodbBackupUsecase() *RestoreMongodbBackupUsecase {
	return restoreMongodbBackupUsecase
}
