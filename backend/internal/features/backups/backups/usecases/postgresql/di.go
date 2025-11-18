package usecases_postgresql

import (
	users_repositories "postgresus-backend/internal/features/users/repositories"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/logger"
)

var createPostgresqlBackupUsecase = &CreatePostgresqlBackupUsecase{
	logger.GetLogger(),
	users_repositories.GetSecretKeyRepository(),
	encryption.GetFieldEncryptor(),
}

func GetCreatePostgresqlBackupUsecase() *CreatePostgresqlBackupUsecase {
	return createPostgresqlBackupUsecase
}
