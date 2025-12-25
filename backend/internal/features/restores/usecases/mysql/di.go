package usecases_mysql

import (
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var restoreMysqlBackupUsecase = &RestoreMysqlBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
}

func GetRestoreMysqlBackupUsecase() *RestoreMysqlBackupUsecase {
	return restoreMysqlBackupUsecase
}
