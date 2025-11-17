package usecases_postgresql

import (
	users_repositories "postgresus-backend/internal/features/users/repositories"
	"postgresus-backend/internal/util/logger"
)

var restorePostgresqlBackupUsecase = &RestorePostgresqlBackupUsecase{
	logger.GetLogger(),
	users_repositories.GetSecretKeyRepository(),
}

func GetRestorePostgresqlBackupUsecase() *RestorePostgresqlBackupUsecase {
	return restorePostgresqlBackupUsecase
}
