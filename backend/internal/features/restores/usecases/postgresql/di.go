package usecases_postgresql

import (
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var restorePostgresqlBackupUsecase = &RestorePostgresqlBackupUsecase{
	logger.GetLogger(),
	secrets.GetSecretKeyService(),
}

func GetRestorePostgresqlBackupUsecase() *RestorePostgresqlBackupUsecase {
	return restorePostgresqlBackupUsecase
}
