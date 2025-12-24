package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	env_utils "postgresus-backend/internal/util/env"
)

type MongodbVersion string

const (
	MongodbVersion4 MongodbVersion = "4"
	MongodbVersion5 MongodbVersion = "5"
	MongodbVersion6 MongodbVersion = "6"
	MongodbVersion7 MongodbVersion = "7"
	MongodbVersion8 MongodbVersion = "8"
)

type MongodbExecutable string

const (
	MongodbExecutableMongodump    MongodbExecutable = "mongodump"
	MongodbExecutableMongorestore MongodbExecutable = "mongorestore"
)

// GetMongodbExecutable returns the full path to a MongoDB executable.
// MongoDB Database Tools use a single client version that is backward compatible
// with all server versions.
func GetMongodbExecutable(
	executable MongodbExecutable,
	envMode env_utils.EnvMode,
	mongodbInstallDir string,
) string {
	basePath := getMongodbBasePath(envMode, mongodbInstallDir)
	executableName := string(executable)

	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	return filepath.Join(basePath, executableName)
}

// VerifyMongodbInstallation verifies that MongoDB Database Tools are installed.
// Unlike PostgreSQL (version-specific), MongoDB tools use a single version that
// supports all server versions (backward compatible).
func VerifyMongodbInstallation(
	logger *slog.Logger,
	envMode env_utils.EnvMode,
	mongodbInstallDir string,
) {
	binDir := getMongodbBasePath(envMode, mongodbInstallDir)

	logger.Info(
		"Verifying MongoDB Database Tools installation",
		"path", binDir,
	)

	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		if envMode == env_utils.EnvModeDevelopment {
			logger.Warn(
				"MongoDB bin directory not found. MongoDB support will be disabled. Read ./tools/readme.md for details",
				"path",
				binDir,
			)
		} else {
			logger.Warn(
				"MongoDB bin directory not found. MongoDB support will be disabled.",
				"path", binDir,
			)
		}
		return
	}

	requiredCommands := []MongodbExecutable{
		MongodbExecutableMongodump,
		MongodbExecutableMongorestore,
	}

	for _, cmd := range requiredCommands {
		cmdPath := GetMongodbExecutable(cmd, envMode, mongodbInstallDir)

		logger.Info(
			"Checking for MongoDB command",
			"command", cmd,
			"path", cmdPath,
		)

		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			if envMode == env_utils.EnvModeDevelopment {
				logger.Warn(
					"MongoDB command not found. MongoDB support will be disabled. Read ./tools/readme.md for details",
					"command",
					cmd,
					"path",
					cmdPath,
				)
			} else {
				logger.Warn(
					"MongoDB command not found. MongoDB support will be disabled.",
					"command", cmd,
					"path", cmdPath,
				)
			}
			continue
		}

		logger.Info("MongoDB command found", "command", cmd)
	}

	logger.Info("MongoDB Database Tools verification completed!")
}

// IsMongodbBackupVersionHigherThanRestoreVersion checks if backup was made with
// a newer MongoDB version than the restore target
func IsMongodbBackupVersionHigherThanRestoreVersion(
	backupVersion, restoreVersion MongodbVersion,
) bool {
	versionOrder := map[MongodbVersion]int{
		MongodbVersion4: 4,
		MongodbVersion5: 5,
		MongodbVersion6: 6,
		MongodbVersion7: 7,
		MongodbVersion8: 8,
	}
	return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// GetMongodbVersionEnum converts a version string to MongodbVersion enum.
// Accepts full version strings (e.g., "8.2", "5.0.1") and extracts the major version.
func GetMongodbVersionEnum(version string) MongodbVersion {
	re := regexp.MustCompile(`^(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 2 {
		panic(fmt.Sprintf("invalid mongodb version format: %s", version))
	}

	major := matches[1]
	switch major {
	case "4":
		return MongodbVersion4
	case "5":
		return MongodbVersion5
	case "6":
		return MongodbVersion6
	case "7":
		return MongodbVersion7
	case "8":
		return MongodbVersion8
	default:
		panic(fmt.Sprintf("unsupported mongodb major version: %s", major))
	}
}

func getMongodbBasePath(
	envMode env_utils.EnvMode,
	mongodbInstallDir string,
) string {
	if envMode == env_utils.EnvModeDevelopment {
		return filepath.Join(mongodbInstallDir, "bin")
	}
	// Production: single client version in /usr/local/mongodb-database-tools/bin
	return "/usr/local/mongodb-database-tools/bin"
}
