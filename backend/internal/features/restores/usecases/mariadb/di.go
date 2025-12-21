package usecases_mariadb

import (
	"postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/util/logger"
)

var restoreMariadbBackupUsecase = &RestoreMariadbBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
}

func GetRestoreMariadbBackupUsecase() *RestoreMariadbBackupUsecase {
	return restoreMariadbBackupUsecase
}
