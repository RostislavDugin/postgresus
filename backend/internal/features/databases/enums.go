package databases

type DatabaseType string

const (
	DatabaseTypePostgresLogical  DatabaseType = "POSTGRES_LOGICAL"
	DatabaseTypePostgresPhysical DatabaseType = "POSTGRES_PHYSICAL"
	DatabaseTypeMysql            DatabaseType = "MYSQL"
	DatabaseTypeMariadb          DatabaseType = "MARIADB"
	DatabaseTypeMongodb          DatabaseType = "MONGODB"
)

type HealthStatus string

const (
	HealthStatusAvailable   HealthStatus = "AVAILABLE"
	HealthStatusUnavailable HealthStatus = "UNAVAILABLE"
)
