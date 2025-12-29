package usecases

import (
	usecases_mariadb "databasus-backend/internal/features/backups/backups/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/backups/backups/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/backups/backups/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/backups/backups/usecases/postgresql"
)

var createBackupUsecase = &CreateBackupUsecase{
	usecases_postgresql.GetCreatePostgresqlBackupUsecase(),
	usecases_mysql.GetCreateMysqlBackupUsecase(),
	usecases_mariadb.GetCreateMariadbBackupUsecase(),
	usecases_mongodb.GetCreateMongodbBackupUsecase(),
}

func GetCreateBackupUsecase() *CreateBackupUsecase {
	return createBackupUsecase
}
