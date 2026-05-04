package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

var mysqlVersions = []MysqlVersion{
	MysqlVersion57,
	MysqlVersion80,
	MysqlVersion84,
	MysqlVersion9,
}

var mysqlRequired = []string{
	string(MysqlExecutableMysqldump),
	string(MysqlExecutableMysql),
}

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

// GetMysqlExecutable returns the absolute path to a MySQL client binary for
// the given version (mysqldump or mysql).
func GetMysqlExecutable(version MysqlVersion, executable MysqlExecutable) string {
	return filepath.Join(getMysqlBinDir(version), withExeOnWindows(string(executable)))
}

func getMysqlBinDir(version MysqlVersion) string {
	return filepath.Join(
		AssetsToolsDir(),
		"mysql",
		fmt.Sprintf("mysql-%s", version),
		"bin",
	)
}

// checkMysql verifies every supported MySQL version's bin directory. MySQL
// is non-fatal — a missing bundle disables that version's support.
func checkMysql() []ToolCheckResult {
	results := make([]ToolCheckResult, 0, len(mysqlVersions))

	for _, v := range mysqlVersions {
		binDir := getMysqlBinDir(v)

		results = append(results, ToolCheckResult{
			Db:      "mysql",
			Version: string(v),
			BinDir:  binDir,
			Errors:  checkBinDir(binDir, mysqlRequired),
			IsFatal: false,
		})
	}

	return results
}

// IsMysqlBackupVersionHigherThanRestoreVersion reports whether a backup
// produced on backupVersion would be downgrade-restoring onto restoreVersion.
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

// EscapeMysqlPassword escapes special characters for the MySQL .my.cnf file
// format (passwords with special chars are double-quoted).
func EscapeMysqlPassword(password string) string {
	password = strings.ReplaceAll(password, "\\", "\\\\")
	password = strings.ReplaceAll(password, "\"", "\\\"")
	return password
}

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
