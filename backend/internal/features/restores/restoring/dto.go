package restoring

import (
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
)

type RestoreDatabaseCache struct {
	PostgresqlLogicalDatabase *postgresql_logical.PostgresqlLogicalDatabase `json:"postgresqlDatabase,omitzero"`
	MysqlDatabase             *mysql.MysqlDatabase                          `json:"mysqlDatabase,omitzero"`
	MariadbDatabase           *mariadb.MariadbDatabase                      `json:"mariadbDatabase,omitzero"`
	MongodbDatabase           *mongodb.MongodbDatabase                      `json:"mongodbDatabase,omitzero"`
}
