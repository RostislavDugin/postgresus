package usecases

import (
	usecases_mysql "postgresus-backend/internal/features/backups/backups/usecases/mysql"
	usecases_postgresql "postgresus-backend/internal/features/backups/backups/usecases/postgresql"
)

var createBackupUsecase = &CreateBackupUsecase{
	usecases_postgresql.GetCreatePostgresqlBackupUsecase(),
	usecases_mysql.GetCreateMysqlBackupUsecase(),
}

func GetCreateBackupUsecase() *CreateBackupUsecase {
	return createBackupUsecase
}
