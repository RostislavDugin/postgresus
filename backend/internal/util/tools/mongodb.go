package tools

import (
	"fmt"
	"path/filepath"
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
	MongodbVersion9 MongodbVersion = "9"
)

type MongodbExecutable string

const (
	MongodbExecutableMongodump    MongodbExecutable = "mongodump"
	MongodbExecutableMongorestore MongodbExecutable = "mongorestore"
)

// GetMongodbExecutable returns the absolute path to a MongoDB Database Tools
// binary. The tools are version-agnostic — a single client supports all
// supported server versions (4.2 – 9.x).
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
		// The MongoDB tools bundle is not named after a release line — one
		// client serves every supported server — so there is no version to
		// hold it to, only the requirement that it starts.
		Errors:  runBinDirChecks(binDir, mongodbRequired, "", nil),
		IsFatal: false,
	}}
}

// IsMongodbBackupVersionHigherThanRestoreVersion reports whether a backup
// produced on backupVersion would be downgrade-restoring onto restoreVersion.
func IsMongodbBackupVersionHigherThanRestoreVersion(
	backupVersion, restoreVersion MongodbVersion,
) (bool, error) {
	order, err := compareReleaseLines(string(backupVersion), string(restoreVersion))
	if err != nil {
		return false, fmt.Errorf("cannot order MongoDB versions: %w", err)
	}

	return order > 0, nil
}
