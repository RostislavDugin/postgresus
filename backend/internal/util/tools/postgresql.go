package tools

import (
	"fmt"
	"path/filepath"
	"runtime"
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

// GetPostgresqlExecutable returns the absolute path to a PostgreSQL client
// binary for the given version (e.g. pg_dump, pg_restore, psql).
func GetPostgresqlExecutable(
	version PostgresqlVersion,
	executable PostgresqlExecutable,
) string {
	return filepath.Join(getPostgresqlBinDir(version), withExeOnWindows(string(executable)))
}

func getPostgresqlBinDir(version PostgresqlVersion) string {
	// Windows pg 12/13 have a piping bug on restore — fall through to the v14
	// client which speaks the older wire formats fine.
	if runtime.GOOS == "windows" {
		if version == PostgresqlVersion12 || version == PostgresqlVersion13 {
			version = PostgresqlVersion14
		}
	}

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
			Errors:  checkBinDir(binDir, postgresqlRequired),
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
