package tools

import (
	"fmt"
	"slices"
	"strconv"
)

type PostgresqlExtension string

const (
	// needed for queries monitoring
	PostgresqlExtensionPgStatMonitor PostgresqlExtension = "pg_stat_statements"
)

type PostgresqlVersion string

const (
	PostgresqlVersion12 PostgresqlVersion = "12"
	PostgresqlVersion13 PostgresqlVersion = "13"
	PostgresqlVersion14 PostgresqlVersion = "14"
	PostgresqlVersion15 PostgresqlVersion = "15"
	PostgresqlVersion16 PostgresqlVersion = "16"
	PostgresqlVersion17 PostgresqlVersion = "17"
	PostgresqlVersion18 PostgresqlVersion = "18"
	PostgresqlVersion19 PostgresqlVersion = "19"
)

type PostgresqlExecutable string

const (
	PostgresqlExecutablePgDump          PostgresqlExecutable = "pg_dump"
	PostgresqlExecutablePsql            PostgresqlExecutable = "psql"
	PostgresqlExecutablePgBasebackup    PostgresqlExecutable = "pg_basebackup"
	PostgresqlExecutablePgReceivewal    PostgresqlExecutable = "pg_receivewal"
	PostgresqlExecutablePgCombinebackup PostgresqlExecutable = "pg_combinebackup"
)

func GetPostgresqlVersionEnum(version string) PostgresqlVersion {
	parsedVersion, err := ParsePostgresqlVersion(version)
	if err != nil {
		panic(err)
	}

	return parsedVersion
}

func ParsePostgresqlVersion(version string) (PostgresqlVersion, error) {
	postgresqlVersion := PostgresqlVersion(version)
	if slices.Contains(postgresqlVersions, postgresqlVersion) {
		return postgresqlVersion, nil
	}

	return "", fmt.Errorf("invalid postgresql version: %s", version)
}

func IsBackupDbVersionHigherThanRestoreDbVersion(
	backupDbVersion, restoreDbVersion PostgresqlVersion,
) bool {
	backupDbVersionInt, err := strconv.Atoi(string(backupDbVersion))
	if err != nil {
		return false
	}

	restoreDbVersionInt, err := strconv.Atoi(string(restoreDbVersion))
	if err != nil {
		return false
	}

	return backupDbVersionInt > restoreDbVersionInt
}
