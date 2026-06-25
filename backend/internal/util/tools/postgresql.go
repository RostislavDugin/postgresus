package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

var postgresqlVersions = []PostgresqlVersion{
	PostgresqlVersion12,
	PostgresqlVersion13,
	PostgresqlVersion14,
	PostgresqlVersion15,
	PostgresqlVersion16,
	PostgresqlVersion17,
	PostgresqlVersion18,
}

var postgresqlRequired = []string{
	string(PostgresqlExecutablePgDump),
	string(PostgresqlExecutablePsql),
}

// postgresqlRequiredV17Plus extends the base set with the physical-backup
// binaries. pg_basebackup --incremental and pg_combinebackup are PG 17+ only,
// so pre-17 versions stay on the smaller list.
var postgresqlRequiredV17Plus = []string{
	string(PostgresqlExecutablePgDump),
	string(PostgresqlExecutablePsql),
	string(PostgresqlExecutablePgBasebackup),
	string(PostgresqlExecutablePgReceivewal),
	string(PostgresqlExecutablePgCombinebackup),
}

func getPostgresqlRequiredForVersion(version PostgresqlVersion) []string {
	if version == PostgresqlVersion17 || version == PostgresqlVersion18 {
		return postgresqlRequiredV17Plus
	}

	return postgresqlRequired
}

// GetPostgresqlExecutable returns the absolute path to a PostgreSQL client
// binary for the given version (e.g. pg_dump, pg_restore, psql).
func GetPostgresqlExecutable(
	version PostgresqlVersion,
	executable PostgresqlExecutable,
) string {
	return filepath.Join(getPostgresqlBinDir(version), string(executable))
}

func getPostgresqlBinDir(version PostgresqlVersion) string {
	return filepath.Join(
		AssetsToolsDir(),
		"postgresql",
		fmt.Sprintf("postgresql-%s", version),
		"bin",
	)
}

// checkPostgresql verifies every supported PG version's bin directory. PG is
// fatal-tier — the app reads the version from each managed database and must
// be able to invoke the matching client.
func checkPostgresql() []ToolCheckResult {
	results := make([]ToolCheckResult, 0, len(postgresqlVersions))

	for _, v := range postgresqlVersions {
		binDir := getPostgresqlBinDir(v)

		results = append(results, ToolCheckResult{
			Db:      "postgresql",
			Version: string(v),
			BinDir:  binDir,
			Errors:  checkBinDir(binDir, getPostgresqlRequiredForVersion(v)),
			IsFatal: true,
		})
	}

	return results
}

// EscapePgpassField escapes special characters for the .pgpass file format.
// PostgreSQL requires backslash → \\ and colon → \:; newlines and carriage
// returns are stripped to prevent format corruption.
func EscapePgpassField(field string) string {
	field = strings.ReplaceAll(field, "\r", "")
	field = strings.ReplaceAll(field, "\n", "")
	field = strings.ReplaceAll(field, "\\", "\\\\")
	field = strings.ReplaceAll(field, ":", "\\:")

	return field
}
