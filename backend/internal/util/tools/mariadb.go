package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

var mariadbClientVersions = []MariadbClientVersion{
	MariadbClientLegacy,
	MariadbClientModern,
}

var mariadbRequired = []string{
	string(MariadbExecutableMariadbDump),
	string(MariadbExecutableMariadb),
}

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

// MariadbClientVersion is the client tool version installed in assets.
type MariadbClientVersion string

const (
	// MariadbClientLegacy is used for older MariaDB servers (5.5, 10.1) that
	// don't have the generation_expression column in information_schema.columns.
	MariadbClientLegacy MariadbClientVersion = "10.6"
	// MariadbClientModern is used for newer MariaDB servers (10.2+).
	MariadbClientModern MariadbClientVersion = "12.1"
)

type MariadbExecutable string

const (
	MariadbExecutableMariadbDump MariadbExecutable = "mariadb-dump"
	MariadbExecutableMariadb     MariadbExecutable = "mariadb"
)

// GetMariadbClientVersionForServer returns the client version that talks to
// the given server version. The 12.1 client uses queries referencing
// generation_expression (added in MariaDB 10.2), so older servers (5.5, 10.1)
// need the 10.6 legacy client.
func GetMariadbClientVersionForServer(serverVersion MariadbVersion) MariadbClientVersion {
	switch serverVersion {
	case MariadbVersion55, MariadbVersion101:
		return MariadbClientLegacy
	default:
		return MariadbClientModern
	}
}

// GetMariadbExecutable returns the absolute path to a MariaDB client binary
// appropriate for the given server version.
func GetMariadbExecutable(
	serverVersion MariadbVersion,
	executable MariadbExecutable,
) string {
	clientVersion := GetMariadbClientVersionForServer(serverVersion)
	return filepath.Join(getMariadbBinDir(clientVersion), string(executable))
}

func getMariadbBinDir(clientVersion MariadbClientVersion) string {
	return filepath.Join(
		AssetsToolsDir(),
		"mariadb",
		fmt.Sprintf("mariadb-%s", clientVersion),
		"bin",
	)
}

// checkMariadb verifies the legacy and modern MariaDB client bundles.
// Non-fatal — missing bundles disable that client tier.
func checkMariadb() []ToolCheckResult {
	results := make([]ToolCheckResult, 0, len(mariadbClientVersions))

	for _, cv := range mariadbClientVersions {
		binDir := getMariadbBinDir(cv)

		results = append(results, ToolCheckResult{
			Db:      "mariadb",
			Version: string(cv),
			BinDir:  binDir,
			Errors:  checkBinDir(binDir, mariadbRequired),
			IsFatal: false,
		})
	}

	return results
}

// IsMariadbBackupVersionHigherThanRestoreVersion reports whether a backup
// produced on backupVersion would be downgrade-restoring onto restoreVersion.
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

// EscapeMariadbPassword escapes special characters for the MariaDB .my.cnf
// file format.
func EscapeMariadbPassword(password string) string {
	password = strings.ReplaceAll(password, "\\", "\\\\")
	password = strings.ReplaceAll(password, "\"", "\\\"")
	return password
}
