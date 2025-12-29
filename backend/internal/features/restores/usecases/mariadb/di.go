package usecases_mariadb

import (
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var restoreMariadbBackupUsecase = &RestoreMariadbBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
}

func GetRestoreMariadbBackupUsecase() *RestoreMariadbBackupUsecase {
	return restoreMariadbBackupUsecase
}
