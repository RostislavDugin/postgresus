package usecases_clickhouse

import (
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var restoreClickhouseBackupUsecase = &RestoreClickhouseBackupUsecase{
	logger.GetLogger(),
	encryption_secrets.GetSecretKeyService(),
}

func GetRestoreClickhouseBackupUsecase() *RestoreClickhouseBackupUsecase {
	return restoreClickhouseBackupUsecase
}
