package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	env_utils "databasus-backend/internal/util/env"
)

type ClickhouseVersion string

const (
	ClickhouseVersion238 ClickhouseVersion = "23.8"
	ClickhouseVersion244 ClickhouseVersion = "24.4"
	ClickhouseVersion248 ClickhouseVersion = "24.8"
	ClickhouseVersion254 ClickhouseVersion = "25.4"
)

type ClickhouseExecutable string

const (
	ClickhouseExecutableClient ClickhouseExecutable = "clickhouse-client"
)

// GetClickhouseExecutable returns the full path to a ClickHouse executable.
// ClickHouse client uses a single binary that is backward and forward compatible
// across server versions (within reasonable major-version drift).
func GetClickhouseExecutable(
	executable ClickhouseExecutable,
	envMode env_utils.EnvMode,
	clickhouseInstallDir string,
) string {
	basePath := getClickhouseBasePath(envMode, clickhouseInstallDir)
	executableName := string(executable)

	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	return filepath.Join(basePath, executableName)
}

// VerifyClickhouseInstallation verifies that the bundled clickhouse-client binary
// is present. Like MongoDB tools, ClickHouse uses a single client binary that is
// compatible with multiple server versions.
func VerifyClickhouseInstallation(
	logger *slog.Logger,
	envMode env_utils.EnvMode,
	clickhouseInstallDir string,
	isShowLogs bool,
) {
	binDir := getClickhouseBasePath(envMode, clickhouseInstallDir)

	if isShowLogs {
		logger.Info(
			"Verifying ClickHouse client installation",
			"path", binDir,
		)
	}

	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		if envMode == env_utils.EnvModeDevelopment {
			logger.Warn(
				"ClickHouse bin directory not found. ClickHouse support will be disabled. Read ./tools/readme.md for details",
				"path",
				binDir,
			)
		} else {
			logger.Warn(
				"ClickHouse bin directory not found. ClickHouse support will be disabled.",
				"path", binDir,
			)
		}

		return
	}

	requiredCommands := []ClickhouseExecutable{
		ClickhouseExecutableClient,
	}

	for _, cmd := range requiredCommands {
		cmdPath := GetClickhouseExecutable(cmd, envMode, clickhouseInstallDir)

		if isShowLogs {
			logger.Info(
				"Checking for ClickHouse command",
				"command", cmd,
				"path", cmdPath,
			)
		}

		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			if envMode == env_utils.EnvModeDevelopment {
				logger.Warn(
					"ClickHouse command not found. ClickHouse support will be disabled. Read ./tools/readme.md for details",
					"command",
					cmd,
					"path",
					cmdPath,
				)
			} else {
				logger.Warn(
					"ClickHouse command not found. ClickHouse support will be disabled.",
					"command", cmd,
					"path", cmdPath,
				)
			}

			continue
		}

		if isShowLogs {
			logger.Info("ClickHouse command found", "command", cmd)
		}
	}

	if isShowLogs {
		logger.Info("ClickHouse client verification completed!")
	}
}

// IsClickhouseBackupVersionHigherThanRestoreVersion checks if backup was made with
// a newer ClickHouse version than the restore target. Compares the version enum's
// chronological ordering: 23.8 < 24.4 < 24.8 < 25.4.
func IsClickhouseBackupVersionHigherThanRestoreVersion(
	backupVersion, restoreVersion ClickhouseVersion,
) bool {
	versionOrder := map[ClickhouseVersion]int{
		ClickhouseVersion238: 1,
		ClickhouseVersion244: 2,
		ClickhouseVersion248: 3,
		ClickhouseVersion254: 4,
	}
	return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// GetClickhouseVersionEnum converts a version string to ClickhouseVersion enum.
// Accepts full version strings (e.g. "25.4.1.123", "23.8.10.5-stable") and
// extracts the major.minor pair.
func GetClickhouseVersionEnum(version string) ClickhouseVersion {
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 3 {
		panic(fmt.Sprintf("invalid clickhouse version format: %s", version))
	}

	majorMinor := matches[1] + "." + matches[2]
	switch majorMinor {
	case "23.8":
		return ClickhouseVersion238
	case "24.4":
		return ClickhouseVersion244
	case "24.8":
		return ClickhouseVersion248
	case "25.4":
		return ClickhouseVersion254
	default:
		// Unknown major.minor — fall back to the closest supported LTS bucket.
		// 23.x → 23.8, 24.x → 24.8, 25.x+ → 25.4.
		switch matches[1] {
		case "23":
			return ClickhouseVersion238
		case "24":
			return ClickhouseVersion248
		default:
			return ClickhouseVersion254
		}
	}
}

func getClickhouseBasePath(
	envMode env_utils.EnvMode,
	clickhouseInstallDir string,
) string {
	if envMode == env_utils.EnvModeDevelopment {
		return filepath.Join(clickhouseInstallDir, "bin")
	}
	// Production: single client binary in /usr/local/clickhouse/bin
	return "/usr/local/clickhouse/bin"
}
