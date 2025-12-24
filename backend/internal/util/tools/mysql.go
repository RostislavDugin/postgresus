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

type MysqlVersion string

const (
	MysqlVersion57 MysqlVersion = "5.7"
	MysqlVersion80 MysqlVersion = "8.0"
	MysqlVersion84 MysqlVersion = "8.4"
	MysqlVersion9  MysqlVersion = "9"
)

type MysqlExecutable string

const (
	MysqlExecutableMysqldump MysqlExecutable = "mysqldump"
	MysqlExecutableMysql     MysqlExecutable = "mysql"
)

// GetMysqlExecutable returns the full path to a specific MySQL executable
// for the given version. Common executables include: mysqldump, mysql.
// On Windows, automatically appends .exe extension.
func GetMysqlExecutable(
	version MysqlVersion,
	executable MysqlExecutable,
	envMode env_utils.EnvMode,
	mysqlInstallDir string,
) string {
	basePath := getMysqlBasePath(version, envMode, mysqlInstallDir)
	executableName := string(executable)

	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	return filepath.Join(basePath, executableName)
}

// VerifyMysqlInstallation verifies that MySQL versions 5.7, 8.0, 8.4, 9 are installed
// in the current environment. Each version should be installed with the required
// client tools (mysqldump, mysql) available.
// In development: ./tools/mysql/mysql-{VERSION}/bin
// In production: /usr/local/mysql-{VERSION}/bin
func VerifyMysqlInstallation(
	logger *slog.Logger,
	envMode env_utils.EnvMode,
	mysqlInstallDir string,
) {
	versions := []MysqlVersion{
		MysqlVersion57,
		MysqlVersion80,
		MysqlVersion84,
		MysqlVersion9,
	}

	requiredCommands := []MysqlExecutable{
		MysqlExecutableMysqldump,
		MysqlExecutableMysql,
	}

	for _, version := range versions {
		binDir := getMysqlBasePath(version, envMode, mysqlInstallDir)

		logger.Info(
			"Verifying MySQL installation",
			"version",
			string(version),
			"path",
			binDir,
		)

		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			if envMode == env_utils.EnvModeDevelopment {
				logger.Warn(
					"MySQL bin directory not found. MySQL support will be disabled. Read ./tools/readme.md for details",
					"version",
					string(version),
					"path",
					binDir,
				)
			} else {
				logger.Warn(
					"MySQL bin directory not found. MySQL support will be disabled.",
					"version",
					string(version),
					"path",
					binDir,
				)
			}
			continue
		}

		for _, cmd := range requiredCommands {
			cmdPath := GetMysqlExecutable(
				version,
				cmd,
				envMode,
				mysqlInstallDir,
			)

			logger.Info(
				"Checking for MySQL command",
				"command",
				cmd,
				"version",
				string(version),
				"path",
				cmdPath,
			)

			if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
				if envMode == env_utils.EnvModeDevelopment {
					logger.Warn(
						"MySQL command not found. MySQL support for this version will be disabled. Read ./tools/readme.md for details",
						"command",
						cmd,
						"version",
						string(version),
						"path",
						cmdPath,
					)
				} else {
					logger.Warn(
						"MySQL command not found. MySQL support for this version will be disabled.",
						"command",
						cmd,
						"version",
						string(version),
						"path",
						cmdPath,
					)
				}
				continue
			}

			logger.Info(
				"MySQL command found",
				"command",
				cmd,
				"version",
				string(version),
			)
		}

		logger.Info(
			"Installation of MySQL verified",
			"version",
			string(version),
			"path",
			binDir,
		)
	}

	logger.Info("MySQL version-specific client tools verification completed!")
}

// IsMysqlBackupVersionHigherThanRestoreVersion checks if backup was made with
// a newer MySQL version than the restore target
func IsMysqlBackupVersionHigherThanRestoreVersion(
	backupVersion, restoreVersion MysqlVersion,
) bool {
	versionOrder := map[MysqlVersion]int{
		MysqlVersion57: 1,
		MysqlVersion80: 2,
		MysqlVersion84: 3,
		MysqlVersion9:  4,
	}
	return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// EscapeMysqlPassword escapes special characters for MySQL .my.cnf file format.
// In .my.cnf, passwords with special chars should be quoted.
// Escape backslash and quote characters.
func EscapeMysqlPassword(password string) string {
	password = strings.ReplaceAll(password, "\\", "\\\\")
	password = strings.ReplaceAll(password, "\"", "\\\"")
	return password
}

// GetMysqlVersionEnum converts a version string to MysqlVersion enum
func GetMysqlVersionEnum(version string) MysqlVersion {
	switch version {
	case "5.7":
		return MysqlVersion57
	case "8.0":
		return MysqlVersion80
	case "8.4":
		return MysqlVersion84
	case "9":
		return MysqlVersion9
	default:
		panic(fmt.Sprintf("invalid mysql version: %s", version))
	}
}

func getMysqlBasePath(
	version MysqlVersion,
	envMode env_utils.EnvMode,
	mysqlInstallDir string,
) string {
	if envMode == env_utils.EnvModeDevelopment {
		return filepath.Join(
			mysqlInstallDir,
			fmt.Sprintf("mysql-%s", string(version)),
			"bin",
		)
	}
	return fmt.Sprintf("/usr/local/mysql-%s/bin", string(version))
}
