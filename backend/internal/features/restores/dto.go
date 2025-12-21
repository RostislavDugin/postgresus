package restores

import (
	"postgresus-backend/internal/features/databases/databases/mariadb"
	"postgresus-backend/internal/features/databases/databases/mysql"
	"postgresus-backend/internal/features/databases/databases/postgresql"
)

type RestoreBackupRequest struct {
	PostgresqlDatabase *postgresql.PostgresqlDatabase `json:"postgresqlDatabase"`
	MysqlDatabase      *mysql.MysqlDatabase           `json:"mysqlDatabase"`
	MariadbDatabase    *mariadb.MariadbDatabase       `json:"mariadbDatabase"`
}
