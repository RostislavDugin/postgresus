package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	env_utils "databasus-backend/internal/util/env"
)

type MariadbVersion string

const (
	MariadbVersion55   MariadbVersion = "5.5"
	MariadbVersion101  MariadbVersion = "10.1"
	MariadbVersion102  MariadbVersion = "10.2"
	MariadbVersion103  MariadbVersion = "10.3"
	MariadbVersion104  MariadbVersion = "10.4"
	MariadbVersion105  MariadbVersion = "10.5"
	MariadbVersion106  MariadbVersion = "10.6"
	MariadbVersion1011 MariadbVersion = "10.11"
	MariadbVersion114  MariadbVersion = "11.4"
	MariadbVersion118  MariadbVersion = "11.8"
	MariadbVersion120  MariadbVersion = "12.0"
)

// MariadbClientVersion represents the client tool version to use
type MariadbClientVersion string

const (
	// MariadbClientLegacy is used for older MariaDB servers (5.5, 10.1) that don't support
	// the generation_expression column in information_schema.columns
	MariadbClientLegacy MariadbClientVersion = "10.6"
	// MariadbClientModern is used for newer MariaDB servers (10.2+)
	MariadbClientModern MariadbClientVersion = "12.1"
)

type MariadbExecutable string

const (
	MariadbExecutableMariadbDump MariadbExecutable = "mariadb-dump"
	MariadbExecutableMariadb     MariadbExecutable = "mariadb"
)

// GetMariadbClientVersionForServer returns the appropriate client version to use
// for a given server version. MariaDB 12.1 client uses SQL queries that reference
// the generation_expression column which was added in MariaDB 10.2, so older
// servers (5.5, 10.1) need the legacy 10.6 client.
func GetMariadbClientVersionForServer(serverVersion MariadbVersion) MariadbClientVersion {
	switch serverVersion {
	case MariadbVersion55, MariadbVersion101:
		return MariadbClientLegacy
	default:
		return MariadbClientModern
	}
}

// GetMariadbExecutable returns the full path to a MariaDB executable.
// The serverVersion parameter determines which client tools to use:
// - For MariaDB 5.5 and 10.1: uses legacy 10.6 client (compatible with older servers)
// - For MariaDB 10.2+: uses modern 12.1 client
func GetMariadbExecutable(
	executable MariadbExecutable,
	serverVersion MariadbVersion,
	envMode env_utils.EnvMode,
	mariadbInstallDir string,
) string {
	clientVersion := GetMariadbClientVersionForServer(serverVersion)
	basePath := getMariadbBasePath(clientVersion, envMode, mariadbInstallDir)
	executableName := string(executable)

	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	return filepath.Join(basePath, executableName)
}

// VerifyMariadbInstallation verifies that MariaDB client tools are installed.
// MariaDB uses two client versions:
// - Legacy (10.6) for older servers (5.5, 10.1)
// - Modern (12.1) for newer servers (10.2+)
func VerifyMariadbInstallation(
	logger *slog.Logger,
	envMode env_utils.EnvMode,
	mariadbInstallDir string,
) {
	clientVersions := []MariadbClientVersion{MariadbClientLegacy, MariadbClientModern}

	for _, clientVersion := range clientVersions {
		binDir := getMariadbBasePath(clientVersion, envMode, mariadbInstallDir)

		logger.Info(
			"Verifying MariaDB installation",
			"clientVersion", clientVersion,
			"path", binDir,
		)

		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			if envMode == env_utils.EnvModeDevelopment {
				logger.Warn(
					"MariaDB bin directory not found. Some MariaDB versions may not be supported. Read ./tools/readme.md for details",
					"clientVersion",
					clientVersion,
					"path",
					binDir,
				)
			} else {
				logger.Warn(
					"MariaDB bin directory not found. Some MariaDB versions may not be supported.",
					"clientVersion", clientVersion,
					"path", binDir,
				)
			}
			continue
		}

		requiredCommands := []MariadbExecutable{
			MariadbExecutableMariadbDump,
			MariadbExecutableMariadb,
		}

		for _, cmd := range requiredCommands {
			// Use a dummy server version that maps to this client version
			var dummyServerVersion MariadbVersion
			if clientVersion == MariadbClientLegacy {
				dummyServerVersion = MariadbVersion55
			} else {
				dummyServerVersion = MariadbVersion102
			}
			cmdPath := GetMariadbExecutable(cmd, dummyServerVersion, envMode, mariadbInstallDir)

			logger.Info(
				"Checking for MariaDB command",
				"clientVersion", clientVersion,
				"command", cmd,
				"path", cmdPath,
			)

			if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
				if envMode == env_utils.EnvModeDevelopment {
					logger.Warn(
						"MariaDB command not found. Some MariaDB versions may not be supported. Read ./tools/readme.md for details",
						"clientVersion",
						clientVersion,
						"command",
						cmd,
						"path",
						cmdPath,
					)
				} else {
					logger.Warn(
						"MariaDB command not found. Some MariaDB versions may not be supported.",
						"clientVersion", clientVersion,
						"command", cmd,
						"path", cmdPath,
					)
				}
				continue
			}

			logger.Info("MariaDB command found", "clientVersion", clientVersion, "command", cmd)
		}
	}

	logger.Info("MariaDB client tools verification completed!")
}

// IsMariadbBackupVersionHigherThanRestoreVersion checks if backup was made with
// a newer MariaDB version than the restore target
func IsMariadbBackupVersionHigherThanRestoreVersion(
	backupVersion, restoreVersion MariadbVersion,
) bool {
	versionOrder := map[MariadbVersion]int{
		MariadbVersion55:   1,
		MariadbVersion101:  2,
		MariadbVersion102:  3,
		MariadbVersion103:  4,
		MariadbVersion104:  5,
		MariadbVersion105:  6,
		MariadbVersion106:  7,
		MariadbVersion1011: 8,
		MariadbVersion114:  9,
		MariadbVersion118:  10,
		MariadbVersion120:  11,
	}
	return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// GetMariadbVersionEnum converts a version string to MariadbVersion enum
func GetMariadbVersionEnum(version string) MariadbVersion {
	switch version {
	case "5.5":
		return MariadbVersion55
	case "10.1":
		return MariadbVersion101
	case "10.2":
		return MariadbVersion102
	case "10.3":
		return MariadbVersion103
	case "10.4":
		return MariadbVersion104
	case "10.5":
		return MariadbVersion105
	case "10.6":
		return MariadbVersion106
	case "10.11":
		return MariadbVersion1011
	case "11.4":
		return MariadbVersion114
	case "11.8":
		return MariadbVersion118
	case "12.0":
		return MariadbVersion120
	default:
		panic(fmt.Sprintf("invalid mariadb version: %s", version))
	}
}

// EscapeMariadbPassword escapes special characters for MariaDB .my.cnf file format.
func EscapeMariadbPassword(password string) string {
	password = strings.ReplaceAll(password, "\\", "\\\\")
	password = strings.ReplaceAll(password, "\"", "\\\"")
	return password
}

func getMariadbBasePath(
	clientVersion MariadbClientVersion,
	envMode env_utils.EnvMode,
	mariadbInstallDir string,
) string {
	if envMode == env_utils.EnvModeDevelopment {
		// Development: tools/mariadb/mariadb-{version}/bin
		return filepath.Join(mariadbInstallDir, fmt.Sprintf("mariadb-%s", clientVersion), "bin")
	}
	// Production: /usr/local/mariadb-{version}/bin
	return fmt.Sprintf("/usr/local/mariadb-%s/bin", clientVersion)
}
