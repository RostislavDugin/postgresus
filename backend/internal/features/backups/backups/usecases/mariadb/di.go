package usecases_mariadb

import (
	"postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/logger"
)

var createMariadbBackupUsecase = &CreateMariadbBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
	return createMariadbBackupUsecase
}
