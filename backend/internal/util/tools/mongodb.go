package tools

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var mongodbRequired = []string{
	string(MongodbExecutableMongodump),
	string(MongodbExecutableMongorestore),
}

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

// GetMongodbExecutable returns the absolute path to a MongoDB Database Tools
// binary. The tools are version-agnostic — a single client supports all
// supported server versions (4.2 – 8.x).
func GetMongodbExecutable(executable MongodbExecutable) string {
	return filepath.Join(getMongodbBinDir(), string(executable))
}

func getMongodbBinDir() string {
	return filepath.Join(AssetsToolsDir(), "mongodb", "bin")
}

// checkMongodb verifies the unified MongoDB Database Tools bundle. Non-fatal
// — a missing bundle disables MongoDB support.
func checkMongodb() []ToolCheckResult {
	binDir := getMongodbBinDir()

	return []ToolCheckResult{{
		Db:      "mongodb",
		Version: "tools",
		BinDir:  binDir,
		Errors:  checkBinDir(binDir, mongodbRequired),
		IsFatal: false,
	}}
}

// IsMongodbBackupVersionHigherThanRestoreVersion reports whether a backup
// produced on backupVersion would be downgrade-restoring onto restoreVersion.
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

// GetMongodbVersionEnum extracts the major version from a full version
// string (e.g. "8.2", "5.0.1") and returns the matching enum value.
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
