package databases

type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "POSTGRES"
	DatabaseTypeMysql    DatabaseType = "MYSQL"
	DatabaseTypeMariadb  DatabaseType = "MARIADB"
)

type HealthStatus string

const (
	HealthStatusAvailable   HealthStatus = "AVAILABLE"
	HealthStatusUnavailable HealthStatus = "UNAVAILABLE"
)
