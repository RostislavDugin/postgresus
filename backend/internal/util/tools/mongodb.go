package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	env_utils "postgresus-backend/internal/util/env"
)

type MongodbVersion string

const (
	MongodbVersion40 MongodbVersion = "4.0"
	MongodbVersion42 MongodbVersion = "4.2"
	MongodbVersion44 MongodbVersion = "4.4"
	MongodbVersion50 MongodbVersion = "5.0"
	MongodbVersion60 MongodbVersion = "6.0"
	MongodbVersion70 MongodbVersion = "7.0"
	MongodbVersion80 MongodbVersion = "8.0"
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
		MongodbVersion40: 1,
		MongodbVersion42: 2,
		MongodbVersion44: 3,
		MongodbVersion50: 4,
		MongodbVersion60: 5,
		MongodbVersion70: 6,
		MongodbVersion80: 7,
	}
	return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// GetMongodbVersionEnum converts a version string to MongodbVersion enum
func GetMongodbVersionEnum(version string) MongodbVersion {
	switch version {
	case "4.0":
		return MongodbVersion40
	case "4.2":
		return MongodbVersion42
	case "4.4":
		return MongodbVersion44
	case "5.0":
		return MongodbVersion50
	case "6.0":
		return MongodbVersion60
	case "7.0":
		return MongodbVersion70
	case "8.0":
		return MongodbVersion80
	default:
		panic(fmt.Sprintf("invalid mongodb version: %s", version))
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
