package tools

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var clickhouseRequired = []string{
	string(ClickhouseExecutableClient),
}

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

// GetClickhouseExecutable returns the absolute path to a ClickHouse client
// binary. ClickHouse client uses a single multicall binary that is forward
// and backward compatible across server versions (within reasonable major-
// version drift), so the path is version-independent.
func GetClickhouseExecutable(executable ClickhouseExecutable) string {
	return filepath.Join(getClickhouseBinDir(), withExeOnWindows(string(executable)))
}

func getClickhouseBinDir() string {
	return filepath.Join(AssetsToolsDir(), "clickhouse", "bin")
}

// checkClickhouse verifies the bundled clickhouse-client binary. Non-fatal —
// a missing bundle disables ClickHouse support.
func checkClickhouse() []ToolCheckResult {
	binDir := getClickhouseBinDir()

	return []ToolCheckResult{{
		Db:      "clickhouse",
		Version: "client",
		BinDir:  binDir,
		Errors:  checkBinDir(binDir, clickhouseRequired),
		IsFatal: false,
	}}
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
