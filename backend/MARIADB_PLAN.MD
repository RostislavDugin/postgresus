# MariaDB Implementation Plan

## Overview

This document outlines the implementation plan for adding MariaDB backup and restore support to Postgresus. The implementation will follow the existing PostgreSQL and MySQL architecture patterns, with MariaDB as a separate database type.

## Supported MariaDB Versions

| Version           | Status | EOL Date      | Support |
| ----------------- | ------ | ------------- | ------- |
| **MariaDB 5.5**   | EOL    | April 2020    | ✅ Yes  |
| **MariaDB 10.1**  | EOL    | October 2020  | ✅ Yes  |
| **MariaDB 10.2**  | EOL    | May 2022      | ✅ Yes  |
| **MariaDB 10.3**  | EOL    | May 2023      | ✅ Yes  |
| **MariaDB 10.4**  | EOL    | June 2024     | ✅ Yes  |
| **MariaDB 10.5**  | EOL    | June 2025     | ✅ Yes  |
| **MariaDB 10.6**  | LTS    | July 2026     | ✅ Yes  |
| **MariaDB 10.11** | LTS    | February 2028 | ✅ Yes  |
| **MariaDB 11.4**  | LTS    | May 2029      | ✅ Yes  |
| **MariaDB 11.8**  | LTS    | June 2028     | ✅ Yes  |
| **MariaDB 12.0**  | Stable | TBD (~2028)   | ✅ Yes  |

**Notes:**

- MariaDB versions 8 and 9 don't exist. MariaDB versioning went from 5.5 → 10.0 to differentiate from MySQL.
- MariaDB 5.5 and 10.1-10.5 reached EOL but are still supported for legacy systems that cannot be upgraded.
- MariaDB 12.0 was declared stable in March 2025. LTS version 12.3 is expected in 2026.

## Key Decisions

| Aspect             | Decision                                                                   |
| ------------------ | -------------------------------------------------------------------------- |
| **Architecture**   | Separate `DatabaseTypeMariadb` (not reusing MySQL)                         |
| **Client tools**   | Single latest client (12.0) - backward compatible with all server versions |
| **Backup tool**    | Native `mariadb-dump`                                                      |
| **Restore tool**   | Native `mariadb` client                                                    |
| **Auto-detection** | Parse "MariaDB" from `SELECT VERSION()`                                    |
| **Cross-restore**  | No (MariaDB → MariaDB only)                                                |
| **ARM64**          | Full support (all versions)                                                |

---

## File Structure

### New Files to Create

```
backend/internal/features/databases/databases/mariadb/
├── model.go                    # MariadbDatabase struct and methods
├── readonly_user.go            # Read-only user creation (MariaDB-specific)

backend/internal/features/backups/backups/usecases/mariadb/
├── create_backup_uc.go         # MariaDB backup use case
├── di.go                       # Dependency injection

backend/internal/features/restores/usecases/mariadb/
├── restore_backup_uc.go        # MariaDB restore use case
├── di.go                       # Dependency injection

backend/internal/util/tools/
├── mariadb.go                  # MariaDB executable helpers and version enums

backend/internal/features/tests/
├── mariadb_backup_restore_test.go  # Integration tests

backend/migrations/
├── XXXXXX_add_mariadb_databases_table.up.sql
├── XXXXXX_add_mariadb_databases_table.down.sql
```

### Files to Modify

```
backend/internal/features/databases/
├── enums.go                    # Add DatabaseTypeMariadb
├── model.go                    # Add Mariadb field, update methods
├── service.go                  # Handle MariaDB in CopyDatabase, IsUserReadOnly, CreateReadOnlyUser
├── repository.go               # Preload MariaDB relation

backend/internal/features/backups/backups/usecases/
├── create_backup_uc.go         # Add MariaDB case
├── di.go                       # Wire MariaDB use case

backend/internal/features/restores/usecases/
├── restore_backup_uc.go        # Add MariaDB case
├── di.go                       # Wire MariaDB use case

backend/internal/features/restores/
├── service.go                  # Add MariaDB validation, version compatibility
├── dto.go                      # Add MariadbDatabase field to RestoreBackupRequest

backend/internal/config/
├── config.go                   # Add MariaDB test ports, verify MariaDB installation

backend/tools/
├── download_linux.sh           # Add MariaDB client download
├── download_macos.sh           # Add MariaDB client download
├── download_windows.bat        # Add MariaDB client download
├── readme.md                   # Update with MariaDB instructions

Dockerfile                      # Add MariaDB client installation
docker-compose.yml.example      # Add MariaDB test containers
.github/workflows/ci-release.yml # Add MariaDB test setup
```

---

## Database Type Enum

### `backend/internal/features/databases/enums.go`

```go
const (
    DatabaseTypePostgres DatabaseType = "POSTGRES"
    DatabaseTypeMysql    DatabaseType = "MYSQL"
    DatabaseTypeMariadb  DatabaseType = "MARIADB"
)
```

---

## MariaDB Version Enums

### `backend/internal/util/tools/mariadb.go`

```go
package tools

import (
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "runtime"

    env_utils "postgresus-backend/internal/util/env"
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

type MariadbExecutable string

const (
    MariadbExecutableMariadbDump MariadbExecutable = "mariadb-dump"
    MariadbExecutableMariadb     MariadbExecutable = "mariadb"
)

// GetMariadbExecutable returns the full path to a MariaDB executable.
// MariaDB uses a single client version (latest) that is backward compatible
// with all server versions.
func GetMariadbExecutable(
    executable MariadbExecutable,
    envMode env_utils.EnvMode,
    mariadbInstallDir string,
) string {
    basePath := getMariadbBasePath(envMode, mariadbInstallDir)
    executableName := string(executable)

    if runtime.GOOS == "windows" {
        executableName += ".exe"
    }

    return filepath.Join(basePath, executableName)
}

// VerifyMariadbInstallation verifies that MariaDB client tools are installed.
// Unlike MySQL/PostgreSQL, MariaDB uses a single client version that supports
// all server versions (backward compatible).
func VerifyMariadbInstallation(
    logger *slog.Logger,
    envMode env_utils.EnvMode,
    mariadbInstallDir string,
) {
    binDir := getMariadbBasePath(envMode, mariadbInstallDir)

    logger.Info(
        "Verifying MariaDB installation",
        "path", binDir,
    )

    if _, err := os.Stat(binDir); os.IsNotExist(err) {
        if envMode == env_utils.EnvModeDevelopment {
            logger.Warn(
                "MariaDB bin directory not found. MariaDB support will be disabled. Read ./tools/readme.md for details",
                "path", binDir,
            )
        } else {
            logger.Warn(
                "MariaDB bin directory not found. MariaDB support will be disabled.",
                "path", binDir,
            )
        }
        return
    }

    requiredCommands := []MariadbExecutable{
        MariadbExecutableMariadbDump,
        MariadbExecutableMariadb,
    }

    for _, cmd := range requiredCommands {
        cmdPath := GetMariadbExecutable(cmd, envMode, mariadbInstallDir)

        logger.Info(
            "Checking for MariaDB command",
            "command", cmd,
            "path", cmdPath,
        )

        if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
            if envMode == env_utils.EnvModeDevelopment {
                logger.Warn(
                    "MariaDB command not found. MariaDB support will be disabled. Read ./tools/readme.md for details",
                    "command", cmd,
                    "path", cmdPath,
                )
            } else {
                logger.Warn(
                    "MariaDB command not found. MariaDB support will be disabled.",
                    "command", cmd,
                    "path", cmdPath,
                )
            }
            continue
        }

        logger.Info("MariaDB command found", "command", cmd)
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

func getMariadbBasePath(
    envMode env_utils.EnvMode,
    mariadbInstallDir string,
) string {
    if envMode == env_utils.EnvModeDevelopment {
        return filepath.Join(mariadbInstallDir, "bin")
    }
    // Production: single client version in /usr/local/mariadb/bin
    return "/usr/local/mariadb/bin"
}
```

---

## Model Definition

### `backend/internal/features/databases/databases/mariadb/model.go`

```go
package mariadb

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log/slog"
    "regexp"
    "strings"
    "time"

    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"

    _ "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"
)

type MariadbDatabase struct {
    ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

    Version tools.MariadbVersion `json:"version" gorm:"type:text;not null"`

    Host     string  `json:"host"     gorm:"type:text;not null"`
    Port     int     `json:"port"     gorm:"type:int;not null"`
    Username string  `json:"username" gorm:"type:text;not null"`
    Password string  `json:"password" gorm:"type:text;not null"`
    Database *string `json:"database" gorm:"type:text"`
    IsHttps  bool    `json:"isHttps"  gorm:"type:boolean;default:false"`
}

func (m *MariadbDatabase) TableName() string {
    return "mariadb_databases"
}

func (m *MariadbDatabase) Validate() error {
    if m.Host == "" {
        return errors.New("host is required")
    }
    if m.Port == 0 {
        return errors.New("port is required")
    }
    if m.Username == "" {
        return errors.New("username is required")
    }
    if m.Password == "" {
        return errors.New("password is required")
    }
    return nil
}

func (m *MariadbDatabase) TestConnection(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if m.Database == nil || *m.Database == "" {
        return errors.New("database name is required for MariaDB backup")
    }

    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return fmt.Errorf("failed to decrypt password: %w", err)
    }

    dsn := m.buildDSN(password, *m.Database)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return fmt.Errorf("failed to connect to MariaDB database '%s': %w", *m.Database, err)
    }
    defer func() {
        if closeErr := db.Close(); closeErr != nil {
            logger.Error("Failed to close MariaDB connection", "error", closeErr)
        }
    }()

    db.SetConnMaxLifetime(15 * time.Second)
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)

    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping MariaDB database '%s': %w", *m.Database, err)
    }

    detectedVersion, err := detectMariadbVersion(ctx, db)
    if err != nil {
        return err
    }
    m.Version = detectedVersion

    return nil
}

func (m *MariadbDatabase) HideSensitiveData() {
    if m == nil {
        return
    }
    m.Password = ""
}

func (m *MariadbDatabase) Update(incoming *MariadbDatabase) {
    m.Version = incoming.Version
    m.Host = incoming.Host
    m.Port = incoming.Port
    m.Username = incoming.Username
    m.Database = incoming.Database
    m.IsHttps = incoming.IsHttps

    if incoming.Password != "" {
        m.Password = incoming.Password
    }
}

func (m *MariadbDatabase) EncryptSensitiveFields(
    databaseID uuid.UUID,
    encryptor encryption.FieldEncryptor,
) error {
    if m.Password != "" {
        encrypted, err := encryptor.Encrypt(databaseID, m.Password)
        if err != nil {
            return err
        }
        m.Password = encrypted
    }
    return nil
}

func (m *MariadbDatabase) PopulateVersionIfEmpty(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    if m.Version != "" {
        return nil
    }

    if m.Database == nil || *m.Database == "" {
        return nil
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return fmt.Errorf("failed to decrypt password: %w", err)
    }

    dsn := m.buildDSN(password, *m.Database)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if closeErr := db.Close(); closeErr != nil {
            logger.Error("Failed to close connection", "error", closeErr)
        }
    }()

    detectedVersion, err := detectMariadbVersion(ctx, db)
    if err != nil {
        return err
    }

    m.Version = detectedVersion
    return nil
}

func (m *MariadbDatabase) IsUserReadOnly(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (bool, error) {
    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return false, fmt.Errorf("failed to decrypt password: %w", err)
    }

    dsn := m.buildDSN(password, *m.Database)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return false, fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if closeErr := db.Close(); closeErr != nil {
            logger.Error("Failed to close connection", "error", closeErr)
        }
    }()

    rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
    if err != nil {
        return false, fmt.Errorf("failed to check grants: %w", err)
    }
    defer func() { _ = rows.Close() }()

    writePrivileges := []string{
        "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
        "INDEX", "GRANT OPTION", "ALL PRIVILEGES", "SUPER",
    }

    for rows.Next() {
        var grant string
        if err := rows.Scan(&grant); err != nil {
            return false, fmt.Errorf("failed to scan grant: %w", err)
        }

        for _, priv := range writePrivileges {
            if regexp.MustCompile(`(?i)\b` + priv + `\b`).MatchString(grant) {
                return false, nil
            }
        }
    }

    if err := rows.Err(); err != nil {
        return false, fmt.Errorf("error iterating grants: %w", err)
    }

    return true, nil
}

func (m *MariadbDatabase) CreateReadOnlyUser(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (string, string, error) {
    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return "", "", fmt.Errorf("failed to decrypt password: %w", err)
    }

    dsn := m.buildDSN(password, *m.Database)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return "", "", fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if closeErr := db.Close(); closeErr != nil {
            logger.Error("Failed to close connection", "error", closeErr)
        }
    }()

    maxRetries := 3
    for attempt := range maxRetries {
        newUsername := fmt.Sprintf("postgresus-%s", uuid.New().String()[:8])
        newPassword := uuid.New().String()

        tx, err := db.BeginTx(ctx, nil)
        if err != nil {
            return "", "", fmt.Errorf("failed to begin transaction: %w", err)
        }

        success := false
        defer func() {
            if !success {
                if rollbackErr := tx.Rollback(); rollbackErr != nil {
                    logger.Error("Failed to rollback transaction", "error", rollbackErr)
                }
            }
        }()

        // MariaDB uses same syntax as MySQL for user creation
        _, err = tx.ExecContext(ctx, fmt.Sprintf(
            "CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
            newUsername,
            newPassword,
        ))
        if err != nil {
            if attempt < maxRetries-1 {
                continue
            }
            return "", "", fmt.Errorf("failed to create user: %w", err)
        }

        // Grant SELECT, SHOW VIEW, LOCK TABLES for backup operations
        // Also grant TRIGGER and EVENT for complete backups
        _, err = tx.ExecContext(ctx, fmt.Sprintf(
            "GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO '%s'@'%%'",
            *m.Database,
            newUsername,
        ))
        if err != nil {
            return "", "", fmt.Errorf("failed to grant database privileges: %w", err)
        }

        // PROCESS privilege needed for --single-transaction
        _, err = tx.ExecContext(ctx, fmt.Sprintf(
            "GRANT PROCESS ON *.* TO '%s'@'%%'",
            newUsername,
        ))
        if err != nil {
            return "", "", fmt.Errorf("failed to grant PROCESS privilege: %w", err)
        }

        _, err = tx.ExecContext(ctx, "FLUSH PRIVILEGES")
        if err != nil {
            return "", "", fmt.Errorf("failed to flush privileges: %w", err)
        }

        if err := tx.Commit(); err != nil {
            return "", "", fmt.Errorf("failed to commit transaction: %w", err)
        }

        success = true
        logger.Info(
            "Read-only MariaDB user created successfully",
            "username", newUsername,
        )
        return newUsername, newPassword, nil
    }

    return "", "", errors.New("failed to generate unique username after 3 attempts")
}

func (m *MariadbDatabase) buildDSN(password string, database string) string {
    tlsConfig := "false"
    if m.IsHttps {
        tlsConfig = "true"
    }

    return fmt.Sprintf(
        "%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=15s&tls=%s&charset=utf8mb4",
        m.Username,
        password,
        m.Host,
        m.Port,
        database,
        tlsConfig,
    )
}

// detectMariadbVersion parses VERSION() output to detect MariaDB version
// MariaDB returns strings like "10.11.6-MariaDB" or "11.4.2-MariaDB-1:11.4.2+maria~ubu2204"
func detectMariadbVersion(ctx context.Context, db *sql.DB) (tools.MariadbVersion, error) {
    var versionStr string
    err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionStr)
    if err != nil {
        return "", fmt.Errorf("failed to query MariaDB version: %w", err)
    }

    // Check if this is actually MariaDB (not MySQL)
    if !strings.Contains(strings.ToLower(versionStr), "mariadb") {
        return "", fmt.Errorf("not a MariaDB server (version: %s). Use MySQL database type instead", versionStr)
    }

    // Parse version number (e.g., "10.11.6-MariaDB" -> "10.11")
    re := regexp.MustCompile(`^(\d+)\.(\d+)`)
    matches := re.FindStringSubmatch(versionStr)
    if len(matches) < 3 {
        return "", fmt.Errorf("could not parse MariaDB version: %s", versionStr)
    }

    major := matches[1]
    minor := matches[2]
    versionKey := fmt.Sprintf("%s.%s", major, minor)

    switch versionKey {
    case "5.5":
        return tools.MariadbVersion55, nil
    case "10.1":
        return tools.MariadbVersion101, nil
    case "10.2":
        return tools.MariadbVersion102, nil
    case "10.3":
        return tools.MariadbVersion103, nil
    case "10.4":
        return tools.MariadbVersion104, nil
    case "10.5":
        return tools.MariadbVersion105, nil
    case "10.6":
        return tools.MariadbVersion106, nil
    case "10.11":
        return tools.MariadbVersion1011, nil
    case "11.4":
        return tools.MariadbVersion114, nil
    case "11.8":
        return tools.MariadbVersion118, nil
    case "12.0":
        return tools.MariadbVersion120, nil
    default:
        return "", fmt.Errorf("unsupported MariaDB version: %s (supported: 5.5, 10.1-10.6, 10.11, 11.4, 11.8, 12.0)", versionKey)
    }
}

func decryptPasswordIfNeeded(
    password string,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (string, error) {
    if encryptor == nil {
        return password, nil
    }
    return encryptor.Decrypt(databaseID, password)
}
```

---

## Backup Implementation

### `backend/internal/features/backups/backups/usecases/mariadb/create_backup_uc.go`

```go
package usecases_mariadb

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/klauspost/compress/zstd"

    "postgresus-backend/internal/config"
    backup_encryption "postgresus-backend/internal/features/backups/backups/encryption"
    usecases_common "postgresus-backend/internal/features/backups/backups/usecases/common"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mariadbtypes "postgresus-backend/internal/features/databases/databases/mariadb"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/storages"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"
)

const (
    backupTimeout               = 23 * time.Hour
    shutdownCheckInterval       = 1 * time.Second
    copyBufferSize              = 8 * 1024 * 1024
    progressReportIntervalMB    = 1.0
    zstdStorageCompressionLevel = 3
    exitCodeGenericError        = 1
    exitCodeConnectionError     = 2
)

type CreateMariadbBackupUsecase struct {
    logger           *slog.Logger
    secretKeyService *encryption_secrets.SecretKeyService
    fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
    bytesWritten int
    writeErr     error
}

func (uc *CreateMariadbBackupUsecase) Execute(
    ctx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    db *databases.Database,
    storage *storages.Storage,
    backupProgressListener func(completedMBs float64),
) (*usecases_common.BackupMetadata, error) {
    uc.logger.Info(
        "Creating MariaDB backup via mariadb-dump",
        "databaseId", db.ID,
        "storageId", storage.ID,
    )

    if !backupConfig.IsBackupsEnabled {
        return nil, fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
    }

    mdb := db.Mariadb
    if mdb == nil {
        return nil, fmt.Errorf("mariadb database configuration is required")
    }

    if mdb.Database == nil || *mdb.Database == "" {
        return nil, fmt.Errorf("database name is required for mariadb-dump backups")
    }

    decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, mdb.Password)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt database password: %w", err)
    }

    args := uc.buildMariadbDumpArgs(mdb)

    return uc.streamToStorage(
        ctx,
        backupID,
        backupConfig,
        tools.GetMariadbExecutable(
            tools.MariadbExecutableMariadbDump,
            config.GetEnv().EnvMode,
            config.GetEnv().MariadbInstallDir,
        ),
        args,
        decryptedPassword,
        storage,
        backupProgressListener,
        mdb,
    )
}

func (uc *CreateMariadbBackupUsecase) buildMariadbDumpArgs(mdb *mariadbtypes.MariadbDatabase) []string {
    args := []string{
        "--host=" + mdb.Host,
        "--port=" + strconv.Itoa(mdb.Port),
        "--user=" + mdb.Username,
        "--single-transaction",
        "--routines",
        "--triggers",
        "--events",
        "--quick",
        "--verbose",
    }

    // MariaDB supports zstd compression for network transfer (10.4.3+)
    // Use it for all supported versions
    args = append(args, "--compress")

    if mdb.IsHttps {
        args = append(args, "--ssl")
    }

    if mdb.Database != nil && *mdb.Database != "" {
        args = append(args, *mdb.Database)
    }

    return args
}

func (uc *CreateMariadbBackupUsecase) streamToStorage(
    parentCtx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    mariadbBin string,
    args []string,
    password string,
    storage *storages.Storage,
    backupProgressListener func(completedMBs float64),
    mdbConfig *mariadbtypes.MariadbDatabase,
) (*usecases_common.BackupMetadata, error) {
    uc.logger.Info("Streaming MariaDB backup to storage", "mariadbBin", mariadbBin)

    ctx, cancel := uc.createBackupContext(parentCtx)
    defer cancel()

    myCnfFile, err := uc.createTempMyCnfFile(mdbConfig, password)
    if err != nil {
        return nil, fmt.Errorf("failed to create .my.cnf: %w", err)
    }
    defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

    fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

    cmd := exec.CommandContext(ctx, mariadbBin, fullArgs...)
    uc.logger.Info("Executing MariaDB backup command", "command", cmd.String())

    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env,
        "MYSQL_PWD=",
        "LC_ALL=C.UTF-8",
        "LANG=C.UTF-8",
    )

    pgStdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("stdout pipe: %w", err)
    }

    pgStderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, fmt.Errorf("stderr pipe: %w", err)
    }

    stderrCh := make(chan []byte, 1)
    go func() {
        stderrOutput, _ := io.ReadAll(pgStderr)
        stderrCh <- stderrOutput
    }()

    storageReader, storageWriter := io.Pipe()

    finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
        backupID,
        backupConfig,
        storageWriter,
    )
    if err != nil {
        return nil, err
    }

    zstdWriter, err := zstd.NewWriter(finalWriter,
        zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(zstdStorageCompressionLevel)))
    if err != nil {
        return nil, fmt.Errorf("failed to create zstd writer: %w", err)
    }
    countingWriter := usecases_common.NewCountingWriter(zstdWriter)

    saveErrCh := make(chan error, 1)
    go func() {
        saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
        saveErrCh <- saveErr
    }()

    if err = cmd.Start(); err != nil {
        return nil, fmt.Errorf("start %s: %w", filepath.Base(mariadbBin), err)
    }

    copyResultCh := make(chan error, 1)
    bytesWrittenCh := make(chan int64, 1)
    go func() {
        bytesWritten, err := uc.copyWithShutdownCheck(
            ctx,
            countingWriter,
            pgStdout,
            backupProgressListener,
        )
        bytesWrittenCh <- bytesWritten
        copyResultCh <- err
    }()

    copyErr := <-copyResultCh
    bytesWritten := <-bytesWrittenCh
    waitErr := cmd.Wait()

    select {
    case <-ctx.Done():
        uc.cleanupOnCancellation(zstdWriter, encryptionWriter, storageWriter, saveErrCh)
        return nil, uc.checkCancellationReason()
    default:
    }

    if err := zstdWriter.Close(); err != nil {
        uc.logger.Error("Failed to close zstd writer", "error", err)
    }
    if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
        <-saveErrCh
        return nil, err
    }

    saveErr := <-saveErrCh
    stderrOutput := <-stderrCh

    if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
        sizeMB := float64(bytesWritten) / (1024 * 1024)
        backupProgressListener(sizeMB)
    }

    switch {
    case waitErr != nil:
        return nil, uc.buildMariadbDumpErrorMessage(waitErr, stderrOutput, mariadbBin)
    case copyErr != nil:
        return nil, fmt.Errorf("copy to storage: %w", copyErr)
    case saveErr != nil:
        return nil, fmt.Errorf("save to storage: %w", saveErr)
    }

    return &backupMetadata, nil
}

func (uc *CreateMariadbBackupUsecase) createTempMyCnfFile(
    mdbConfig *mariadbtypes.MariadbDatabase,
    password string,
) (string, error) {
    tempDir, err := os.MkdirTemp("", "mycnf")
    if err != nil {
        return "", fmt.Errorf("failed to create temp directory: %w", err)
    }

    myCnfFile := filepath.Join(tempDir, ".my.cnf")

    // Escape password for .my.cnf format
    escapedPassword := strings.ReplaceAll(password, "\\", "\\\\")
    escapedPassword = strings.ReplaceAll(escapedPassword, "\"", "\\\"")

    content := fmt.Sprintf(`[client]
user=%s
password="%s"
host=%s
port=%d
`, mdbConfig.Username, escapedPassword, mdbConfig.Host, mdbConfig.Port)

    if mdbConfig.IsHttps {
        content += "ssl=true\n"
    }

    err = os.WriteFile(myCnfFile, []byte(content), 0600)
    if err != nil {
        return "", fmt.Errorf("failed to write .my.cnf: %w", err)
    }

    return myCnfFile, nil
}

// copyWithShutdownCheck, createBackupContext, setupBackupEncryption,
// cleanupOnCancellation, closeWriters, checkCancellationReason,
// buildMariadbDumpErrorMessage - same implementation as MySQL use case
// (copy from usecases_mysql package)
```

### `backend/internal/features/backups/backups/usecases/mariadb/di.go`

```go
package usecases_mariadb

import (
    "log/slog"

    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/logger"
)

var createMariadbBackupUsecase *CreateMariadbBackupUsecase

func init() {
    createMariadbBackupUsecase = &CreateMariadbBackupUsecase{
        logger:           logger.GetLogger(),
        secretKeyService: encryption_secrets.GetSecretKeyService(),
        fieldEncryptor:   encryption.GetFieldEncryptor(),
    }
}

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
    return createMariadbBackupUsecase
}
```

---

## Restore Implementation

### `backend/internal/features/restores/usecases/mariadb/restore_backup_uc.go`

```go
package usecases_mariadb

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/klauspost/compress/zstd"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    "postgresus-backend/internal/features/backups/backups/encryption"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mariadbtypes "postgresus-backend/internal/features/databases/databases/mariadb"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/restores/models"
    "postgresus-backend/internal/features/storages"
    util_encryption "postgresus-backend/internal/util/encryption"
    files_utils "postgresus-backend/internal/util/files"
    "postgresus-backend/internal/util/tools"
)

type RestoreMariadbBackupUsecase struct {
    logger           *slog.Logger
    secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMariadbBackupUsecase) Execute(
    originalDB *databases.Database,
    restoringToDB *databases.Database,
    backupConfig *backups_config.BackupConfig,
    restore models.Restore,
    backup *backups.Backup,
    storage *storages.Storage,
) error {
    if originalDB.Type != databases.DatabaseTypeMariadb {
        return errors.New("database type not supported")
    }

    uc.logger.Info(
        "Restoring MariaDB backup via mariadb client",
        "restoreId", restore.ID,
        "backupId", backup.ID,
    )

    mdb := restoringToDB.Mariadb
    if mdb == nil {
        return fmt.Errorf("mariadb configuration is required for restore")
    }

    if mdb.Database == nil || *mdb.Database == "" {
        return fmt.Errorf("target database name is required for mariadb restore")
    }

    args := []string{
        "--host=" + mdb.Host,
        "--port=" + strconv.Itoa(mdb.Port),
        "--user=" + mdb.Username,
        "--verbose",
    }

    if mdb.IsHttps {
        args = append(args, "--ssl")
    }

    if mdb.Database != nil && *mdb.Database != "" {
        args = append(args, *mdb.Database)
    }

    return uc.restoreFromStorage(
        originalDB,
        tools.GetMariadbExecutable(
            tools.MariadbExecutableMariadb,
            config.GetEnv().EnvMode,
            config.GetEnv().MariadbInstallDir,
        ),
        args,
        mdb.Password,
        backup,
        storage,
        mdb,
    )
}

// restoreFromStorage, executeMysqlRestore, downloadBackupToTempFile,
// setupDecryption, createTempMyCnfFile, copyWithShutdownCheck,
// handleMariadbRestoreError - same pattern as MySQL restore use case
```

### `backend/internal/features/restores/usecases/mariadb/di.go`

```go
package usecases_mariadb

import (
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/util/logger"
)

var restoreMariadbBackupUsecase *RestoreMariadbBackupUsecase

func init() {
    restoreMariadbBackupUsecase = &RestoreMariadbBackupUsecase{
        logger:           logger.GetLogger(),
        secretKeyService: encryption_secrets.GetSecretKeyService(),
    }
}

func GetRestoreMariadbBackupUsecase() *RestoreMariadbBackupUsecase {
    return restoreMariadbBackupUsecase
}
```

---

## Migration Script

### `backend/migrations/XXXXXX_add_mariadb_databases_table.up.sql`

```sql
CREATE TABLE mariadb_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    version     TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INT NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    database    TEXT,
    is_https    BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_mariadb_databases_database_id ON mariadb_databases(database_id);
```

### `backend/migrations/XXXXXX_add_mariadb_databases_table.down.sql`

```sql
DROP INDEX IF EXISTS idx_mariadb_databases_database_id;
DROP TABLE IF EXISTS mariadb_databases;
```

---

## Model Updates

### `backend/internal/features/databases/model.go` changes

Add to Database struct:

```go
type Database struct {
    // ... existing fields ...

    Postgresql *postgresql.PostgresqlDatabase `json:"postgresql,omitempty" gorm:"foreignKey:DatabaseID"`
    Mysql      *mysql.MysqlDatabase           `json:"mysql,omitempty"      gorm:"foreignKey:DatabaseID"`
    Mariadb    *mariadb.MariadbDatabase       `json:"mariadb,omitempty"    gorm:"foreignKey:DatabaseID"`

    // ... rest of fields ...
}
```

Update methods to handle MariaDB case:

```go
func (d *Database) Validate() error {
    // ... existing code ...
    case DatabaseTypeMariadb:
        if d.Mariadb == nil {
            return errors.New("mariadb database is required")
        }
        return d.Mariadb.Validate()
    // ...
}

func (d *Database) TestConnection(...) error {
    // ... add Mariadb case
}

func (d *Database) EncryptSensitiveFields(...) error {
    // ... add Mariadb case
}

func (d *Database) PopulateVersionIfEmpty(...) error {
    // ... add Mariadb case
}

func (d *Database) Update(incoming *Database) {
    // ... add Mariadb case
}

func (d *Database) HideSensitiveData() {
    // ... add Mariadb case
}
```

---

## Service Updates

### `backend/internal/features/databases/service.go` - CopyDatabase

Add MariaDB case:

```go
case DatabaseTypeMariadb:
    if existingDatabase.Mariadb != nil {
        newDatabase.Mariadb = &mariadb.MariadbDatabase{
            ID:         uuid.Nil,
            DatabaseID: nil,
            Version:    existingDatabase.Mariadb.Version,
            Host:       existingDatabase.Mariadb.Host,
            Port:       existingDatabase.Mariadb.Port,
            Username:   existingDatabase.Mariadb.Username,
            Password:   existingDatabase.Mariadb.Password,
            Database:   existingDatabase.Mariadb.Database,
            IsHttps:    existingDatabase.Mariadb.IsHttps,
        }
    }
```

### `backend/internal/features/databases/service.go` - IsUserReadOnly

Add MariaDB case:

```go
case DatabaseTypeMariadb:
    return usingDatabase.Mariadb.IsUserReadOnly(
        ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
    )
```

### `backend/internal/features/databases/service.go` - CreateReadOnlyUser

Add MariaDB case:

```go
case DatabaseTypeMariadb:
    username, password, err = usingDatabase.Mariadb.CreateReadOnlyUser(
        ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
    )
```

---

## Repository Updates

### `backend/internal/features/databases/repository.go`

Add MariaDB preload and handling:

```go
func (r *DatabaseRepository) FindByID(id uuid.UUID) (*Database, error) {
    // ... existing code ...
    Preload("Mariadb").
    // ...
}

func (r *DatabaseRepository) Save(database *Database) (*Database, error) {
    // ... add case for DatabaseTypeMariadb
    case DatabaseTypeMariadb:
        if database.Mariadb == nil {
            return errors.New("mariadb configuration is required for MariaDB database")
        }
        database.Mariadb.DatabaseID = &database.ID
    // ...
}

func (r *DatabaseRepository) Delete(id uuid.UUID) error {
    // ... add case for DatabaseTypeMariadb
    case DatabaseTypeMariadb:
        if err := tx.
            Where("database_id = ?", id).
            Delete(&mariadb.MariadbDatabase{}).Error; err != nil {
            return err
        }
    // ...
}
```

---

## Restore Service Updates

### `backend/internal/features/restores/dto.go`

```go
type RestoreBackupRequest struct {
    PostgresqlDatabase *postgresql.PostgresqlDatabase `json:"postgresql,omitempty"`
    MysqlDatabase      *mysql.MysqlDatabase           `json:"mysql,omitempty"`
    MariadbDatabase    *mariadb.MariadbDatabase       `json:"mariadb,omitempty"`
}
```

### `backend/internal/features/restores/service.go`

Add version validation for MariaDB:

```go
func (s *RestoreService) validateVersionCompatibility(...) error {
    // ... existing cases ...
    case databases.DatabaseTypeMariadb:
        if requestDTO.MariadbDatabase == nil {
            return errors.New("mariadb database configuration is required for restore")
        }
        if tools.IsMariadbBackupVersionHigherThanRestoreVersion(
            backupDatabase.Mariadb.Version,
            requestDTO.MariadbDatabase.Version,
        ) {
            return errors.New(`backup database version is higher than restore database version. ` +
                `Should be restored to the same version as the backup database or higher. ` +
                `For example, you can restore MariaDB 10.11 backup to MariaDB 10.11, 11.4 or higher. But cannot restore to 10.6`)
        }
    // ...
}
```

Add MariaDB restore handling:

```go
func (s *RestoreService) RestoreBackup(...) error {
    // ... existing code ...
    case databases.DatabaseTypeMariadb:
        if requestDTO.MariadbDatabase == nil {
            return errors.New("mariadb database is required")
        }
    // ...
}
```

---

## Config Updates

### `backend/internal/config/config.go`

```go
type EnvVariables struct {
    // ... existing fields ...

    MariadbInstallDir string `env:"MARIADB_INSTALL_DIR"`

    // Testing MariaDB
    TestMariadb55Port   string `env:"TEST_MARIADB_55_PORT"`
    TestMariadb101Port  string `env:"TEST_MARIADB_101_PORT"`
    TestMariadb102Port  string `env:"TEST_MARIADB_102_PORT"`
    TestMariadb103Port  string `env:"TEST_MARIADB_103_PORT"`
    TestMariadb104Port  string `env:"TEST_MARIADB_104_PORT"`
    TestMariadb105Port  string `env:"TEST_MARIADB_105_PORT"`
    TestMariadb106Port  string `env:"TEST_MARIADB_106_PORT"`
    TestMariadb1011Port string `env:"TEST_MARIADB_1011_PORT"`
    TestMariadb114Port  string `env:"TEST_MARIADB_114_PORT"`
    TestMariadb118Port  string `env:"TEST_MARIADB_118_PORT"`
    TestMariadb120Port  string `env:"TEST_MARIADB_120_PORT"`
}

// In loadEnvVariables():
env.MariadbInstallDir = filepath.Join(backendRoot, "tools", "mariadb")
tools.VerifyMariadbInstallation(log, env.EnvMode, env.MariadbInstallDir)

if env.IsTesting {
    // ... existing checks ...
    if env.TestMariadb55Port == "" {
        log.Error("TEST_MARIADB_55_PORT is empty")
        os.Exit(1)
    }
    // ... similar for other MariaDB ports (10.1-10.6, 10.11, 11.4, 11.8, 12.0)
}
```

---

## Docker Compose for Testing

### `backend/docker-compose.yml.example` additions

```yaml
services:
  # ... existing services ...

  test-mariadb-55:
    image: mariadb:5.5
    container_name: test-mariadb-55
    ports:
      - "${TEST_MARIADB_55_PORT:-33055}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8 --collation-server=utf8_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-prootpassword"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-101:
    image: mariadb:10.1
    container_name: test-mariadb-101
    ports:
      - "${TEST_MARIADB_101_PORT:-33101}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-prootpassword"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-102:
    image: mariadb:10.2
    container_name: test-mariadb-102
    ports:
      - "${TEST_MARIADB_102_PORT:-33102}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-prootpassword"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-103:
    image: mariadb:10.3
    container_name: test-mariadb-103
    ports:
      - "${TEST_MARIADB_103_PORT:-33103}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-prootpassword"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-104:
    image: mariadb:10.4
    container_name: test-mariadb-104
    ports:
      - "${TEST_MARIADB_104_PORT:-33104}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-105:
    image: mariadb:10.5
    container_name: test-mariadb-105
    ports:
      - "${TEST_MARIADB_105_PORT:-33105}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-106:
    image: mariadb:10.6
    container_name: test-mariadb-106
    ports:
      - "${TEST_MARIADB_106_PORT:-33106}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-1011:
    image: mariadb:10.11
    container_name: test-mariadb-1011
    ports:
      - "${TEST_MARIADB_1011_PORT:-33111}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-114:
    image: mariadb:11.4
    container_name: test-mariadb-114
    ports:
      - "${TEST_MARIADB_114_PORT:-33114}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-118:
    image: mariadb:11.8
    container_name: test-mariadb-118
    ports:
      - "${TEST_MARIADB_118_PORT:-33118}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mariadb-120:
    image: mariadb:12.0
    container_name: test-mariadb-120
    ports:
      - "${TEST_MARIADB_120_PORT:-33120}:3306"
    environment:
      MARIADB_ROOT_PASSWORD: rootpassword
      MARIADB_DATABASE: testdb
      MARIADB_USER: testuser
      MARIADB_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10
```

---

## Dockerfile Updates

Add to the runtime stage:

```dockerfile
# Install MariaDB client tools (single latest version - backward compatible)
# MariaDB client 12.0 supports all server versions (5.5, 10.6, 10.11, 11.4, 11.8, 12.0)
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-transport-https curl && \
    # Add MariaDB repository
    curl -fsSL https://mariadb.org/mariadb_release_signing_key.asc | apt-key add - && \
    echo "deb https://mirror.mariadb.org/repo/12.0/debian $(lsb_release -cs) main" > /etc/apt/sources.list.d/mariadb.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends mariadb-client && \
    # Create symlinks to standard location
    mkdir -p /usr/local/mariadb/bin && \
    ln -sf /usr/bin/mariadb /usr/local/mariadb/bin/mariadb && \
    ln -sf /usr/bin/mariadb-dump /usr/local/mariadb/bin/mariadb-dump && \
    rm -rf /var/lib/apt/lists/*
```

---

## CI/CD Updates

### `.github/workflows/ci-release.yml` additions

```yaml
# In .env file creation:
# testing MariaDB
TEST_MARIADB_55_PORT=33055
TEST_MARIADB_101_PORT=33101
TEST_MARIADB_102_PORT=33102
TEST_MARIADB_103_PORT=33103
TEST_MARIADB_104_PORT=33104
TEST_MARIADB_105_PORT=33105
TEST_MARIADB_106_PORT=33106
TEST_MARIADB_1011_PORT=33111
TEST_MARIADB_114_PORT=33114
TEST_MARIADB_118_PORT=33118
TEST_MARIADB_120_PORT=33120

# In "Wait for containers to be ready":
# Wait for MariaDB containers (legacy versions use mysqladmin, newer use healthcheck.sh)
echo "Waiting for MariaDB 5.5..."
timeout 120 bash -c 'until docker exec test-mariadb-55 mysqladmin ping -h localhost -prootpassword 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.1..."
timeout 120 bash -c 'until docker exec test-mariadb-101 mysqladmin ping -h localhost -prootpassword 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.2..."
timeout 120 bash -c 'until docker exec test-mariadb-102 mysqladmin ping -h localhost -prootpassword 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.3..."
timeout 120 bash -c 'until docker exec test-mariadb-103 mysqladmin ping -h localhost -prootpassword 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.4..."
timeout 120 bash -c 'until docker exec test-mariadb-104 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.5..."
timeout 120 bash -c 'until docker exec test-mariadb-105 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.6..."
timeout 120 bash -c 'until docker exec test-mariadb-106 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 10.11..."
timeout 120 bash -c 'until docker exec test-mariadb-1011 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 11.4..."
timeout 120 bash -c 'until docker exec test-mariadb-114 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 11.8..."
timeout 120 bash -c 'until docker exec test-mariadb-118 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'
echo "Waiting for MariaDB 12.0..."
timeout 120 bash -c 'until docker exec test-mariadb-120 healthcheck.sh --connect --innodb_initialized 2>/dev/null; do sleep 2; done'

# Cache MariaDB client tools
- name: Cache MariaDB client tools
  id: cache-mariadb
  uses: actions/cache@v4
  with:
    path: backend/tools/mariadb
    key: mariadb-client-120-v1
```

---

## Download Scripts Updates

### `backend/tools/download_linux.sh` additions

```bash
# ========== MariaDB Installation ==========
echo "========================================"
echo "Installing MariaDB client tools (single latest version 12.0)..."
echo "========================================"

MARIADB_DIR="$(pwd)/mariadb"
mkdir -p "$MARIADB_DIR/bin"

# Add MariaDB repository and install client
$SUDO apt-get install -y -qq apt-transport-https curl
curl -fsSL https://mariadb.org/mariadb_release_signing_key.asc | $SUDO apt-key add - 2>/dev/null
echo "deb https://mirror.mariadb.org/repo/12.0/debian $(lsb_release -cs) main" | $SUDO tee /etc/apt/sources.list.d/mariadb.list >/dev/null
$SUDO apt-get update -qq -y
$SUDO apt-get install -y -qq mariadb-client

# Create symlinks
ln -sf /usr/bin/mariadb "$MARIADB_DIR/bin/mariadb"
ln -sf /usr/bin/mariadb-dump "$MARIADB_DIR/bin/mariadb-dump"

echo "MariaDB client tools installed successfully"
echo "  mariadb: $MARIADB_DIR/bin/mariadb"
echo "  mariadb-dump: $MARIADB_DIR/bin/mariadb-dump"
```

---

## Test Implementation

### `backend/internal/features/tests/mariadb_backup_restore_test.go`

```go
package tests

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    _ "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/stretchr/testify/assert"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mariadbtypes "postgresus-backend/internal/features/databases/databases/mariadb"
    "postgresus-backend/internal/features/restores"
    restores_enums "postgresus-backend/internal/features/restores/enums"
    restores_models "postgresus-backend/internal/features/restores/models"
    "postgresus-backend/internal/features/storages"
    users_enums "postgresus-backend/internal/features/users/enums"
    users_testing "postgresus-backend/internal/features/users/testing"
    workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
    test_utils "postgresus-backend/internal/util/testing"
    "postgresus-backend/internal/util/tools"
)

const dropMariadbTestTableQuery = `DROP TABLE IF EXISTS test_data`

const createMariadbTestTableQuery = `
CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

const insertMariadbTestDataQuery = `
INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300)`

type MariadbContainer struct {
    Host     string
    Port     int
    Username string
    Password string
    Database string
    Version  tools.MariadbVersion
    DB       *sqlx.DB
}

type MariadbTestDataItem struct {
    ID        int       `db:"id"`
    Name      string    `db:"name"`
    Value     int       `db:"value"`
    CreatedAt time.Time `db:"created_at"`
}

func Test_BackupAndRestoreMariadb_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MariadbVersion
        port    string
    }{
        {"MariaDB 5.5", tools.MariadbVersion55, env.TestMariadb55Port},
        {"MariaDB 10.1", tools.MariadbVersion101, env.TestMariadb101Port},
        {"MariaDB 10.2", tools.MariadbVersion102, env.TestMariadb102Port},
        {"MariaDB 10.3", tools.MariadbVersion103, env.TestMariadb103Port},
        {"MariaDB 10.4", tools.MariadbVersion104, env.TestMariadb104Port},
        {"MariaDB 10.5", tools.MariadbVersion105, env.TestMariadb105Port},
        {"MariaDB 10.6", tools.MariadbVersion106, env.TestMariadb106Port},
        {"MariaDB 10.11", tools.MariadbVersion1011, env.TestMariadb1011Port},
        {"MariaDB 11.4", tools.MariadbVersion114, env.TestMariadb114Port},
        {"MariaDB 11.8", tools.MariadbVersion118, env.TestMariadb118Port},
        {"MariaDB 12.0", tools.MariadbVersion120, env.TestMariadb120Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMariadbBackupRestoreForVersion(t, tc.version, tc.port)
        })
    }
}

func Test_BackupAndRestoreMariadbWithEncryption_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MariadbVersion
        port    string
    }{
        {"MariaDB 5.5", tools.MariadbVersion55, env.TestMariadb55Port},
        {"MariaDB 10.1", tools.MariadbVersion101, env.TestMariadb101Port},
        {"MariaDB 10.2", tools.MariadbVersion102, env.TestMariadb102Port},
        {"MariaDB 10.3", tools.MariadbVersion103, env.TestMariadb103Port},
        {"MariaDB 10.4", tools.MariadbVersion104, env.TestMariadb104Port},
        {"MariaDB 10.5", tools.MariadbVersion105, env.TestMariadb105Port},
        {"MariaDB 10.6", tools.MariadbVersion106, env.TestMariadb106Port},
        {"MariaDB 10.11", tools.MariadbVersion1011, env.TestMariadb1011Port},
        {"MariaDB 11.4", tools.MariadbVersion114, env.TestMariadb114Port},
        {"MariaDB 11.8", tools.MariadbVersion118, env.TestMariadb118Port},
        {"MariaDB 12.0", tools.MariadbVersion120, env.TestMariadb120Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMariadbBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
        })
    }
}

func Test_BackupAndRestoreMariadb_WithReadOnlyUser_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MariadbVersion
        port    string
    }{
        {"MariaDB 5.5", tools.MariadbVersion55, env.TestMariadb55Port},
        {"MariaDB 10.1", tools.MariadbVersion101, env.TestMariadb101Port},
        {"MariaDB 10.2", tools.MariadbVersion102, env.TestMariadb102Port},
        {"MariaDB 10.3", tools.MariadbVersion103, env.TestMariadb103Port},
        {"MariaDB 10.4", tools.MariadbVersion104, env.TestMariadb104Port},
        {"MariaDB 10.5", tools.MariadbVersion105, env.TestMariadb105Port},
        {"MariaDB 10.6", tools.MariadbVersion106, env.TestMariadb106Port},
        {"MariaDB 10.11", tools.MariadbVersion1011, env.TestMariadb1011Port},
        {"MariaDB 11.4", tools.MariadbVersion114, env.TestMariadb114Port},
        {"MariaDB 11.8", tools.MariadbVersion118, env.TestMariadb118Port},
        {"MariaDB 12.0", tools.MariadbVersion120, env.TestMariadb120Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMariadbBackupRestoreWithReadOnlyUserForVersion(t, tc.version, tc.port)
        })
    }
}

// Test implementations follow the same pattern as MySQL tests
// See mysql_backup_restore_test.go for reference

func connectToMariadbContainer(version tools.MariadbVersion, port string) (*MariadbContainer, error) {
    if port == "" {
        return nil, fmt.Errorf("MariaDB %s port not configured", version)
    }

    dbName := "testdb"
    password := "rootpassword"
    username := "root"
    host := "127.0.0.1"

    portInt, err := strconv.Atoi(port)
    if err != nil {
        return nil, fmt.Errorf("failed to parse port: %w", err)
    }

    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
        username, password, host, portInt, dbName)

    db, err := sqlx.Connect("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MariaDB database: %w", err)
    }

    return &MariadbContainer{
        Host:     host,
        Port:     portInt,
        Username: username,
        Password: password,
        Database: dbName,
        Version:  version,
        DB:       db,
    }, nil
}

func createMariadbDatabaseViaAPI(
    t *testing.T,
    router *gin.Engine,
    name string,
    workspaceID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    version tools.MariadbVersion,
    token string,
) *databases.Database {
    request := databases.Database{
        Name:        name,
        WorkspaceID: &workspaceID,
        Type:        databases.DatabaseTypeMariadb,
        Mariadb: &mariadbtypes.MariadbDatabase{
            Host:     host,
            Port:     port,
            Username: username,
            Password: password,
            Database: &database,
            Version:  version,
        },
    }

    w := workspaces_testing.MakeAPIRequest(
        router,
        "POST",
        "/api/v1/databases/create",
        "Bearer "+token,
        request,
    )

    if w.Code != http.StatusCreated {
        t.Fatalf("Failed to create MariaDB database. Status: %d, Body: %s", w.Code, w.Body.String())
    }

    var createdDatabase databases.Database
    if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
        t.Fatalf("Failed to unmarshal database response: %v", err)
    }

    return &createdDatabase
}

func createMariadbRestoreViaAPI(
    t *testing.T,
    router *gin.Engine,
    backupID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    version tools.MariadbVersion,
    token string,
) {
    request := restores.RestoreBackupRequest{
        MariadbDatabase: &mariadbtypes.MariadbDatabase{
            Host:     host,
            Port:     port,
            Username: username,
            Password: password,
            Database: &database,
            Version:  version,
        },
    }

    test_utils.MakePostRequest(
        t,
        router,
        fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
        "Bearer "+token,
        request,
        http.StatusOK,
    )
}
```

---

## Implementation Order

1. **Phase 1: Core Infrastructure**
   - [ ] Create `MariadbVersion` enums and `mariadb.go` tools
   - [ ] Create `mariadb/model.go` with `MariadbDatabase` struct
   - [ ] Add migration for `mariadb_databases` table
   - [ ] Update `databases/model.go` to include MariaDB
   - [ ] Update `databases/enums.go` with `DatabaseTypeMariadb`
   - [ ] Update `databases/repository.go` to preload MariaDB

2. **Phase 2: Backup Implementation**
   - [ ] Create `usecases/mariadb/create_backup_uc.go`
   - [ ] Create `usecases/mariadb/di.go`
   - [ ] Update `usecases/create_backup_uc.go` to route to MariaDB
   - [ ] Update `usecases/di.go` to wire MariaDB use case

3. **Phase 3: Restore Implementation**
   - [ ] Create `restores/usecases/mariadb/restore_backup_uc.go`
   - [ ] Create `restores/usecases/mariadb/di.go`
   - [ ] Update `restores/usecases/restore_backup_uc.go` to route to MariaDB
   - [ ] Update `restores/dto.go` for MariaDB restore DTO
   - [ ] Update `restores/service.go` for MariaDB version validation

4. **Phase 4: Service Layer Updates**
   - [ ] Update `databases/service.go` - CopyDatabase, IsUserReadOnly, CreateReadOnlyUser
   - [ ] Update `config/config.go` with MariaDB configuration
   - [ ] Add MariaDB installation verification

5. **Phase 5: Infrastructure**
   - [ ] Update Dockerfile with MariaDB client installation
   - [ ] Update docker-compose.yml.example with MariaDB test containers
   - [ ] Update download_linux.sh with MariaDB client
   - [ ] Update download_macos.sh with MariaDB client
   - [ ] Update download_windows.bat with MariaDB client
   - [ ] Update CI/CD workflow

6. **Phase 6: Testing**
   - [ ] Create `mariadb_backup_restore_test.go`
   - [ ] Test all MariaDB versions (10.6, 10.11, 11.4, 11.8)
   - [ ] Test encryption with MariaDB backups
   - [ ] Test read-only user creation

---

## Notes and Considerations

### MariaDB vs MySQL Differences

| Feature              | MySQL                          | MariaDB                        |
| -------------------- | ------------------------------ | ------------------------------ |
| Client tools         | `mysqldump`, `mysql`           | `mariadb-dump`, `mariadb`      |
| Client compatibility | Version-specific               | Single client for all versions |
| VERSION() output     | `8.0.35`                       | `10.11.6-MariaDB`              |
| Default auth plugin  | `caching_sha2_password` (8.0+) | `mysql_native_password`        |
| Storage engines      | InnoDB, MyISAM                 | InnoDB, Aria, ColumnStore      |

### Version Detection

MariaDB's `SELECT VERSION()` returns strings containing "MariaDB":

- `10.6.18-MariaDB`
- `10.11.6-MariaDB-1:10.11.6+maria~ubu2204`
- `11.4.2-MariaDB`
- `11.8.0-MariaDB`

The detection logic checks for "MariaDB" substring to distinguish from MySQL.

### Client Backward Compatibility

Unlike MySQL/PostgreSQL where version-specific clients are needed, MariaDB's client tools are backward compatible:

- `mariadb-dump` from 12.0 can dump databases from 5.5, 10.1-10.6, 10.11, 11.4, 11.8
- `mariadb` client from 12.0 can restore to any supported version

This simplifies the implementation - only one client version is needed.

### ARM64 Support

All MariaDB versions (10.6, 10.11, 11.4, 11.8, 12.0) have full ARM64 support, unlike MySQL 5.7 which is x86_64 only.

**Note:** MariaDB 5.5 Docker images are available for both amd64 and arm64, but arm64 support may be limited on some platforms.

### Security Considerations

1. **Password handling**: Uses `.my.cnf` temp file (same as MySQL)
2. **File permissions**: `.my.cnf` must be 0600
3. **TLS**: Support SSL/TLS connections via `--ssl` flag

### Cloud Database Compatibility

- **AWS RDS MariaDB**: Works with standard mariadb-dump
- **Google Cloud SQL MariaDB**: Works with standard mariadb-dump
- **Azure Database for MariaDB**: Works with standard mariadb-dump
- **SkySQL (MariaDB Cloud)**: Works with standard mariadb-dump

### Legacy Version Considerations (5.5, 10.1-10.3)

These EOL versions are supported for legacy systems that cannot be upgraded:

| Version | Environment Variables | Healthcheck    | Character Set |
| ------- | --------------------- | -------------- | ------------- |
| 5.5     | `MYSQL_*`             | mysqladmin     | utf8          |
| 10.1    | `MYSQL_*`             | mysqladmin     | utf8mb4       |
| 10.2    | `MYSQL_*`             | mysqladmin     | utf8mb4       |
| 10.3    | `MYSQL_*`             | mysqladmin     | utf8mb4       |
| 10.4+   | `MARIADB_*`           | healthcheck.sh | utf8mb4       |

**Key differences:**

1. **Environment variables**: Versions 5.5-10.3 use `MYSQL_*` env vars, 10.4+ use `MARIADB_*`
2. **Healthcheck**: Versions 5.5-10.3 use `mysqladmin ping`, 10.4+ use `healthcheck.sh`
3. **Character set**: 5.5 defaults to `utf8`, 10.1+ support full `utf8mb4`
4. **Client compatibility**: Modern mariadb-dump (12.0) can backup all legacy versions
5. **Compression**: `--compress` flag behavior varies across versions

### MariaDB 12.0 Specific Features

MariaDB 12.0 introduced several new features:

1. **Passphrase-protected keys**: Enhanced security for key management
2. **SET SESSION AUTHORIZATION**: Execute actions as another user
3. **SHA2 support**: For `file_key_management.so` plugin
4. **New optimizer hints**: Improved query optimization
