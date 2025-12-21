package usecases

import (
	usecases_mariadb "postgresus-backend/internal/features/backups/backups/usecases/mariadb"
	usecases_mysql "postgresus-backend/internal/features/backups/backups/usecases/mysql"
	usecases_postgresql "postgresus-backend/internal/features/backups/backups/usecases/postgresql"
)

var createBackupUsecase = &CreateBackupUsecase{
	usecases_postgresql.GetCreatePostgresqlBackupUsecase(),
	usecases_mysql.GetCreateMysqlBackupUsecase(),
	usecases_mariadb.GetCreateMariadbBackupUsecase(),
}

func GetCreateBackupUsecase() *CreateBackupUsecase {
	return createBackupUsecase
}
