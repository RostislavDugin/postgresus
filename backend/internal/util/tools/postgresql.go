package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	env_utils "postgresus-backend/internal/util/env"
)

// GetPostgresqlExecutable returns the full path to a specific PostgreSQL executable
// for the given version. Common executables include: pg_dump, psql, etc.
// On Windows, automatically appends .exe extension.
func GetPostgresqlExecutable(
	version PostgresqlVersion,
	executable PostgresqlExecutable,
	envMode env_utils.EnvMode,
	postgresesInstallDir string,
) string {
	basePath := getPostgresqlBasePath(version, envMode, postgresesInstallDir)
	executableName := string(executable)

	// Add .exe extension on Windows
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	return filepath.Join(basePath, executableName)
}

// VerifyPostgresesInstallation verifies that PostgreSQL versions 12-18 are installed
// in the current environment. Each version should be installed with the required
// client tools (pg_dump, psql) available.
// In development: ./tools/postgresql/postgresql-{VERSION}/bin
// In production: /usr/pgsql-{VERSION}/bin
func VerifyPostgresesInstallation(
	logger *slog.Logger,
	envMode env_utils.EnvMode,
	postgresesInstallDir string,
) {
	versions := []PostgresqlVersion{
		PostgresqlVersion12,
		PostgresqlVersion13,
		PostgresqlVersion14,
		PostgresqlVersion15,
		PostgresqlVersion16,
		PostgresqlVersion17,
		PostgresqlVersion18,
	}

	requiredCommands := []PostgresqlExecutable{
		PostgresqlExecutablePgDump,
		PostgresqlExecutablePsql,
	}

	for _, version := range versions {
		binDir := getPostgresqlBasePath(version, envMode, postgresesInstallDir)

		logger.Info(
			"Verifying PostgreSQL installation",
			"version",
			string(version),
			"path",
			binDir,
		)

		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			if envMode == env_utils.EnvModeDevelopment {
				logger.Error(
					"PostgreSQL bin directory not found. Make sure PostgreSQL is installed. Read ./tools/readme.md for details",
					"version",
					string(version),
					"path",
					binDir,
				)
			} else {
				logger.Error(
					"PostgreSQL bin directory not found. Please ensure PostgreSQL client tools are installed.",
					"version",
					string(version),
					"path",
					binDir,
				)
			}
			os.Exit(1)
		}

		for _, cmd := range requiredCommands {
			cmdPath := GetPostgresqlExecutable(
				version,
				cmd,
				envMode,
				postgresesInstallDir,
			)

			logger.Info(
				"Checking for PostgreSQL command",
				"command",
				cmd,
				"version",
				string(version),
				"path",
				cmdPath,
			)

			if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
				if envMode == env_utils.EnvModeDevelopment {
					logger.Error(
						"PostgreSQL command not found. Make sure PostgreSQL is installed. Read ./tools/readme.md for details",
						"command",
						cmd,
						"version",
						string(version),
						"path",
						cmdPath,
					)
				} else {
					logger.Error(
						"PostgreSQL command not found. Please ensure PostgreSQL client tools are properly installed.",
						"command",
						cmd,
						"version",
						string(version),
						"path",
						cmdPath,
					)
				}
				os.Exit(1)
			}

			logger.Info(
				"PostgreSQL command found",
				"command",
				cmd,
				"version",
				string(version),
			)
		}

		logger.Info(
			"Installation of PostgreSQL verified",
			"version",
			string(version),
			"path",
			binDir,
		)
	}

	logger.Info("All PostgreSQL version-specific client tools verification completed successfully!")
}

// EscapePgpassField escapes special characters in a field value for .pgpass file format.
// According to PostgreSQL documentation, the .pgpass file format requires:
// - Backslash (\) must be escaped as \\
// - Colon (:) must be escaped as \:
// Additionally, newlines and carriage returns are removed to prevent format corruption.
func EscapePgpassField(field string) string {
	// Remove newlines and carriage returns that would break .pgpass format
	field = strings.ReplaceAll(field, "\r", "")
	field = strings.ReplaceAll(field, "\n", "")

	// Escape backslashes first (order matters!)
	// Then escape colons
	field = strings.ReplaceAll(field, "\\", "\\\\")
	field = strings.ReplaceAll(field, ":", "\\:")

	return field
}

func getPostgresqlBasePath(
	version PostgresqlVersion,
	envMode env_utils.EnvMode,
	postgresesInstallDir string,
) string {
	if envMode == env_utils.EnvModeDevelopment {
		return filepath.Join(
			postgresesInstallDir,
			fmt.Sprintf("postgresql-%s", string(version)),
			"bin",
		)
	} else {
		return fmt.Sprintf("/usr/lib/postgresql/%s/bin", string(version))
	}
}
