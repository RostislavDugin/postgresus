package usecases_mysql

import (
	"postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/util/logger"
)

var restoreMysqlBackupUsecase = &RestoreMysqlBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
}

func GetRestoreMysqlBackupUsecase() *RestoreMysqlBackupUsecase {
	return restoreMysqlBackupUsecase
}
