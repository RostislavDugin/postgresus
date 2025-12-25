package usecases_mariadb

import (
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var createMariadbBackupUsecase = &CreateMariadbBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
}

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
	return createMariadbBackupUsecase
}
