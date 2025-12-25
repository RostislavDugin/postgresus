package usecases_mysql

import (
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var createMysqlBackupUsecase = &CreateMysqlBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateMysqlBackupUsecase() *CreateMysqlBackupUsecase {
	return createMysqlBackupUsecase
}
