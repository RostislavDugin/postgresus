# MySQL Implementation Plan

## Overview

This document outlines the implementation plan for adding MySQL backup and restore support to Postgresus. The implementation will follow the existing PostgreSQL architecture patterns.

## Supported MySQL Versions

- MySQL 5.7
- MySQL 8.0
- MySQL 8.4

## Architecture

### Database Type Enum

Add new database type to `backend/internal/features/databases/enums.go`:

```go
const (
    DatabaseTypePostgres DatabaseType = "POSTGRES"
    DatabaseTypeMysql    DatabaseType = "MYSQL"
)
```

---

## File Structure

### New Files to Create

```
backend/internal/features/databases/databases/mysql/
├── model.go                    # MysqlDatabase struct and methods

backend/internal/features/backups/backups/usecases/mysql/
├── create_backup_uc.go         # MySQL backup use case
├── di.go                       # Dependency injection
├── dto.go                      # DTOs for backup metadata
├── interfaces.go               # Interfaces

backend/internal/features/restores/usecases/mysql/
├── restore_backup_uc.go        # MySQL restore use case
├── di.go                       # Dependency injection

backend/internal/util/tools/
├── mysql.go                    # MySQL executable helpers and version enums

backend/internal/features/tests/
├── mysql_backup_restore_test.go  # Integration tests

backend/migrations/
├── XXXXXX_add_mysql_databases_table.up.sql
├── XXXXXX_add_mysql_databases_table.down.sql
```

### Files to Modify

```
backend/internal/features/databases/
├── enums.go                    # Add DatabaseTypeMysql
├── model.go                    # Add Mysql field, update methods
├── service.go                  # Handle MySQL in CopyDatabase
├── repository.go               # Preload MySQL relation

backend/internal/features/backups/backups/usecases/
├── create_backup_uc.go         # Add MySQL case
├── di.go                       # Wire MySQL use case

backend/internal/features/restores/usecases/
├── restore_backup_uc.go        # Add MySQL case
├── di.go                       # Wire MySQL use case

backend/internal/config/
├── config.go                   # Add MySQL test ports, verify MySQL installation

backend/internal/util/tools/
├── enums.go                    # Add MySQL version enums (or create separate file)
```

---

## Model Definition

### `backend/internal/features/databases/databases/mysql/model.go`

```go
package mysql

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log/slog"
    "regexp"
    "time"

    "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"

    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"
)

type MysqlDatabase struct {
    ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

    Version tools.MysqlVersion `json:"version" gorm:"type:text;not null"`

    // Connection data
    Host     string  `json:"host"     gorm:"type:text;not null"`
    Port     int     `json:"port"     gorm:"type:int;not null"`
    Username string  `json:"username" gorm:"type:text;not null"`
    Password string  `json:"password" gorm:"type:text;not null"`
    Database *string `json:"database" gorm:"type:text"`  // Target database name (single DB per config)
    UseTLS   bool    `json:"useTls"   gorm:"type:boolean;default:false"`
}

func (m *MysqlDatabase) TableName() string {
    return "mysql_databases"
}

func (m *MysqlDatabase) Validate() error {
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

func (m *MysqlDatabase) TestConnection(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    // Implementation: connect via go-sql-driver/mysql
    // Decrypt password, build DSN, test connection
    // Auto-detect version via SELECT VERSION()
}

func (m *MysqlDatabase) HideSensitiveData() {
    if m == nil {
        return
    }
    m.Password = ""
}

func (m *MysqlDatabase) Update(incoming *MysqlDatabase) {
    m.Version = incoming.Version
    m.Host = incoming.Host
    m.Port = incoming.Port
    m.Username = incoming.Username
    m.Database = incoming.Database
    m.UseTLS = incoming.UseTLS

    if incoming.Password != "" {
        m.Password = incoming.Password
    }
}

func (m *MysqlDatabase) EncryptSensitiveFields(
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

func (m *MysqlDatabase) PopulateVersionIfEmpty(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    // Connect and run SELECT VERSION()
    // Parse version string like "8.0.35" or "5.7.44-log"
    // Map to MysqlVersion enum
}

func (m *MysqlDatabase) IsUserReadOnly(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (bool, error) {
    // Check MySQL privileges:
    // - SHOW GRANTS FOR CURRENT_USER()
    // - Look for SELECT only (no INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, etc.)
}

func (m *MysqlDatabase) CreateReadOnlyUser(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (string, string, error) {
    // CREATE USER 'postgresus-xxxxx'@'%' IDENTIFIED BY 'password';
    // GRANT SELECT ON database.* TO 'postgresus-xxxxx'@'%';
    // FLUSH PRIVILEGES;
}

// Helper: build MySQL DSN
func (m *MysqlDatabase) buildDSN(password string, database string) string {
    cfg := mysql.Config{
        User:                 m.Username,
        Passwd:               password,
        Net:                  "tcp",
        Addr:                 fmt.Sprintf("%s:%d", m.Host, m.Port),
        DBName:               database,
        AllowNativePasswords: true,
        ParseTime:            true,
        Timeout:              15 * time.Second,
    }

    if m.UseTLS {
        cfg.TLSConfig = "true"  // or custom TLS config name
    }

    return cfg.FormatDSN()
}

// detectMysqlVersion parses VERSION() output
func detectMysqlVersion(versionStr string) (tools.MysqlVersion, error) {
    // Parse "8.0.35", "5.7.44-log", "8.4.0" etc.
    re := regexp.MustCompile(`^(\d+)\.(\d+)`)
    matches := re.FindStringSubmatch(versionStr)
    if len(matches) < 3 {
        return "", fmt.Errorf("could not parse version: %s", versionStr)
    }

    major := matches[1]
    minor := matches[2]

    switch {
    case major == "5" && minor == "7":
        return tools.MysqlVersion57, nil
    case major == "8" && minor == "0":
        return tools.MysqlVersion80, nil
    case major == "8" && minor == "4":
        return tools.MysqlVersion84, nil
    default:
        return "", fmt.Errorf("unsupported MySQL version: %s.%s", major, minor)
    }
}
```

---

## MySQL Version Enums

### `backend/internal/util/tools/mysql.go`

```go
package tools

import (
    "fmt"
    "path/filepath"
    "runtime"
    "strings"

    env_utils "postgresus-backend/internal/util/env"
)

type MysqlVersion string

const (
    MysqlVersion57 MysqlVersion = "5.7"
    MysqlVersion80 MysqlVersion = "8.0"
    MysqlVersion84 MysqlVersion = "8.4"
)

type MysqlExecutable string

const (
    MysqlExecutableMysqldump  MysqlExecutable = "mysqldump"
    MysqlExecutableMysql      MysqlExecutable = "mysql"
)

func GetMysqlExecutable(
    version MysqlVersion,
    executable MysqlExecutable,
    envMode env_utils.EnvMode,
    mysqlInstallDir string,
) string {
    basePath := getMysqlBasePath(version, envMode, mysqlInstallDir)
    executableName := string(executable)

    if runtime.GOOS == "windows" {
        executableName += ".exe"
    }

    return filepath.Join(basePath, executableName)
}

func getMysqlBasePath(
    version MysqlVersion,
    envMode env_utils.EnvMode,
    mysqlInstallDir string,
) string {
    if envMode == env_utils.EnvModeDevelopment {
        return filepath.Join(
            mysqlInstallDir,
            fmt.Sprintf("mysql-%s", string(version)),
            "bin",
        )
    }
    // Production: /usr/bin or /usr/local/mysql-X.Y/bin
    return fmt.Sprintf("/usr/local/mysql-%s/bin", string(version))
}

func IsMysqlBackupVersionHigherThanRestoreVersion(
    backupVersion, restoreVersion MysqlVersion,
) bool {
    // Compare versions: 5.7 < 8.0 < 8.4
    versionOrder := map[MysqlVersion]int{
        MysqlVersion57: 1,
        MysqlVersion80: 2,
        MysqlVersion84: 3,
    }
    return versionOrder[backupVersion] > versionOrder[restoreVersion]
}

// EscapeMysqlPassword escapes special characters for MySQL command line
// Used when creating .my.cnf temporary file
func EscapeMysqlPassword(password string) string {
    // In .my.cnf, passwords with special chars should be quoted
    // Escape backslash and quote characters
    password = strings.ReplaceAll(password, "\\", "\\\\")
    password = strings.ReplaceAll(password, "\"", "\\\"")
    return password
}
```

---

## Backup Implementation

### `backend/internal/features/backups/backups/usecases/mysql/create_backup_uc.go`

Key implementation details:

```go
package usecases_mysql

import (
    "context"
    "encoding/base64"
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
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mysqltypes "postgresus-backend/internal/features/databases/databases/mysql"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/storages"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"
)

const (
    backupTimeout                = 23 * time.Hour
    shutdownCheckInterval        = 1 * time.Second
    copyBufferSize               = 8 * 1024 * 1024
    progressReportMB             = 1.0
    mysqlConnectTimeout          = 30
    zstdStorageCompressionLevel  = 3 // Medium level (1-19, 3 is default/balanced)
)

type CreateMysqlBackupUsecase struct {
    logger           *slog.Logger
    secretKeyService *encryption_secrets.SecretKeyService
    fieldEncryptor   encryption.FieldEncryptor
}

type BackupMetadata struct {
    Encryption     backups_config.BackupEncryption
    EncryptionSalt *string
    EncryptionIV   *string
}

func (uc *CreateMysqlBackupUsecase) Execute(
    ctx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    db *databases.Database,
    storage *storages.Storage,
    backupProgressListener func(completedMBs float64),
) (*BackupMetadata, error) {
    uc.logger.Info(
        "Creating MySQL backup via mysqldump",
        "databaseId", db.ID,
        "storageId", storage.ID,
    )

    if !backupConfig.IsBackupsEnabled {
        return nil, fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
    }

    my := db.Mysql
    if my == nil {
        return nil, fmt.Errorf("mysql database configuration is required")
    }

    // Decrypt password
    decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, my.Password)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt database password: %w", err)
    }

    args := uc.buildMysqldumpArgs(my)

    return uc.streamToStorage(
        ctx,
        backupID,
        backupConfig,
        tools.GetMysqlExecutable(
            my.Version,
            tools.MysqlExecutableMysqldump,
            config.GetEnv().EnvMode,
            config.GetEnv().MysqlInstallDir,
        ),
        args,
        decryptedPassword,
        storage,
        db,
        backupProgressListener,
        my,
    )
}

func (uc *CreateMysqlBackupUsecase) buildMysqldumpArgs(my *mysqltypes.MysqlDatabase) []string {
    args := []string{
        "--host=" + my.Host,
        "--port=" + strconv.Itoa(my.Port),
        "--user=" + my.Username,
        "--single-transaction",      // Consistent snapshot for InnoDB
        "--routines",                // Include stored procedures and functions
        "--triggers",                // Include triggers
        "--events",                  // Include scheduled events
        "--set-gtid-purged=OFF",     // Avoid GTID issues on restore
        "--quick",                   // Don't buffer result set in memory
        "--verbose",
    }

    // Network compression: use zstd for MySQL 8.0.18+, zlib for older versions
    // This compresses data during transfer between MySQL server and mysqldump client
    args = append(args, uc.getNetworkCompressionArgs(my.Version)...)

    // MySQL 8.0+ client connecting to 5.7 server needs this flag
    if my.Version == tools.MysqlVersion57 {
        args = append(args, "--column-statistics=0")
    }

    // SSL/TLS
    if my.UseTLS {
        args = append(args, "--ssl-mode=REQUIRED")
    }

    // Single database backup
    if my.Database != nil && *my.Database != "" {
        args = append(args, *my.Database)
    }

    return args
}

// getNetworkCompressionArgs returns compression arguments based on MySQL version
// MySQL 8.0.18+ supports zstd compression algorithm (faster, better ratio)
// MySQL 5.7 and early 8.0 only support zlib via --compress flag
func (uc *CreateMysqlBackupUsecase) getNetworkCompressionArgs(version tools.MysqlVersion) []string {
    const zstdCompressionLevel = 3 // Medium level (1-19, default 3)

    switch version {
    case tools.MysqlVersion80, tools.MysqlVersion84:
        // MySQL 8.0.18+ supports zstd with compression level
        return []string{
            "--compression-algorithms=zstd",
            fmt.Sprintf("--zstd-compression-level=%d", zstdCompressionLevel),
        }
    case tools.MysqlVersion57:
        // MySQL 5.7 only supports zlib via --compress (no level control)
        return []string{"--compress"}
    default:
        // Fallback to basic compression
        return []string{"--compress"}
    }
}

func (uc *CreateMysqlBackupUsecase) streamToStorage(
    parentCtx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    mysqlBin string,
    args []string,
    password string,
    storage *storages.Storage,
    db *databases.Database,
    backupProgressListener func(completedMBs float64),
    myConfig *mysqltypes.MysqlDatabase,
) (*BackupMetadata, error) {
    uc.logger.Info("Streaming MySQL backup to storage", "mysqlBin", mysqlBin)

    ctx, cancel := uc.createBackupContext(parentCtx)
    defer cancel()

    // Create temporary .my.cnf file for password authentication
    // This avoids password appearing in process list
    myCnfFile, err := uc.createTempMyCnfFile(myConfig, password)
    if err != nil {
        return nil, fmt.Errorf("failed to create .my.cnf: %w", err)
    }
    defer os.RemoveAll(filepath.Dir(myCnfFile))

    // Add defaults-file as FIRST argument (required by mysqldump)
    fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

    cmd := exec.CommandContext(ctx, mysqlBin, fullArgs...)
    uc.logger.Info("Executing MySQL backup command", "command", cmd.String())

    // Setup environment
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env,
        "MYSQL_PWD=",  // Clear any existing password env var
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

    // Capture stderr
    stderrCh := make(chan []byte, 1)
    go func() {
        stderrOutput, _ := io.ReadAll(pgStderr)
        stderrCh <- stderrOutput
    }()

    // Setup storage writer with optional encryption
    storageReader, storageWriter := io.Pipe()

    finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
        backupID,
        backupConfig,
        storageWriter,
    )
    if err != nil {
        return nil, err
    }

    // Wrap with zstd compression (mysqldump outputs plain SQL)
    // zstd level 3 (medium) - 10x faster than gzip with similar ratio
    zstdWriter, err := zstd.NewWriter(finalWriter,
        zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(zstdStorageCompressionLevel)))
    if err != nil {
        return nil, fmt.Errorf("failed to create zstd writer: %w", err)
    }
    countingWriter := &CountingWriter{writer: zstdWriter}

    // Start storage save goroutine
    saveErrCh := make(chan error, 1)
    go func() {
        saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
        saveErrCh <- saveErr
    }()

    // Start mysqldump
    if err = cmd.Start(); err != nil {
        return nil, fmt.Errorf("start %s: %w", filepath.Base(mysqlBin), err)
    }

    // Copy with shutdown check (same pattern as PostgreSQL)
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

    // Check for cancellation
    select {
    case <-ctx.Done():
        uc.cleanupOnCancellation(zstdWriter, encryptionWriter, storageWriter, saveErrCh)
        return nil, uc.checkCancellationReason()
    default:
    }

    // Close writers in order: zstd -> encryption -> pipe
    if err := zstdWriter.Close(); err != nil {
        uc.logger.Error("Failed to close zstd writer", "error", err)
    }
    if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
        <-saveErrCh
        return nil, err
    }

    saveErr := <-saveErrCh
    stderrOutput := <-stderrCh

    // Report final size
    if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
        sizeMB := float64(bytesWritten) / (1024 * 1024)
        backupProgressListener(sizeMB)
    }

    // Handle errors
    switch {
    case waitErr != nil:
        return nil, uc.buildMysqldumpErrorMessage(waitErr, stderrOutput, mysqlBin)
    case copyErr != nil:
        return nil, fmt.Errorf("copy to storage: %w", copyErr)
    case saveErr != nil:
        return nil, fmt.Errorf("save to storage: %w", saveErr)
    }

    return &backupMetadata, nil
}

// createTempMyCnfFile creates a temporary MySQL config file with credentials
// This is the secure way to pass password to mysqldump
func (uc *CreateMysqlBackupUsecase) createTempMyCnfFile(
    myConfig *mysqltypes.MysqlDatabase,
    password string,
) (string, error) {
    tempDir, err := os.MkdirTemp("", "mycnf")
    if err != nil {
        return "", fmt.Errorf("failed to create temp directory: %w", err)
    }

    myCnfFile := filepath.Join(tempDir, ".my.cnf")

    content := fmt.Sprintf(`[client]
user=%s
password="%s"
host=%s
port=%d
`, myConfig.Username, tools.EscapeMysqlPassword(password), myConfig.Host, myConfig.Port)

    if myConfig.UseTLS {
        content += "ssl-mode=REQUIRED\n"
    }

    err = os.WriteFile(myCnfFile, []byte(content), 0600)
    if err != nil {
        return "", fmt.Errorf("failed to write .my.cnf: %w", err)
    }

    return myCnfFile, nil
}

// copyWithShutdownCheck - same pattern as PostgreSQL implementation
// Supports cancellation and progress reporting
func (uc *CreateMysqlBackupUsecase) copyWithShutdownCheck(
    ctx context.Context,
    dst io.Writer,
    src io.Reader,
    backupProgressListener func(completedMBs float64),
) (int64, error) {
    // Same implementation as PostgreSQL
    // Uses copyBufferSize, checks ctx.Done() and config.IsShouldShutdown()
}

// setupBackupEncryption - same pattern as PostgreSQL
func (uc *CreateMysqlBackupUsecase) setupBackupEncryption(
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, BackupMetadata, error) {
    // Same implementation as PostgreSQL
    // AES-256-GCM encryption with generated salt and nonce
}
```

### Key Differences from PostgreSQL:

1. **Output Format**: mysqldump outputs plain SQL (not custom format like pg_dump -Fc)
2. **Compression**: We add gzip compression in the pipeline since mysqldump doesn't have built-in custom format
3. **Password File**: Uses `.my.cnf` instead of `.pgpass`
4. **Arguments**: `--defaults-file` must be the FIRST argument
5. **Parallelism**: mysqldump doesn't support parallel dump (mysqlpump does, but has issues)

### Backup Pipeline:

```
mysqldump stdout -> gzip -> [encryption] -> storage
```

---

## Restore Implementation

### `backend/internal/features/restores/usecases/mysql/restore_backup_uc.go`

```go
package usecases_mysql

import (
    "context"
    "fmt"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "time"

    "github.com/google/uuid"
    "github.com/klauspost/compress/zstd"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    backup_encryption "postgresus-backend/internal/features/backups/backups/encryption"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mysqltypes "postgresus-backend/internal/features/databases/databases/mysql"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/restores/models"
    "postgresus-backend/internal/features/storages"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"
)

type RestoreMysqlBackupUsecase struct {
    logger           *slog.Logger
    fieldEncryptor   encryption.FieldEncryptor
    secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMysqlBackupUsecase) Execute(
    originalDB *databases.Database,
    restoringToDB *databases.Database,
    backupConfig *backups_config.BackupConfig,
    restore models.Restore,
    backup *backups.Backup,
    storage *storages.Storage,
) error {
    if originalDB.Type != databases.DatabaseTypeMysql {
        return fmt.Errorf("database type not supported")
    }

    uc.logger.Info(
        "Restoring MySQL backup via mysql client",
        "restoreId", restore.ID,
        "backupId", backup.ID,
    )

    my := restoringToDB.Mysql
    if my == nil {
        return fmt.Errorf("mysql configuration is required for restore")
    }

    // Build mysql client args
    args := []string{
        "--host=" + my.Host,
        "--port=" + strconv.Itoa(my.Port),
        "--user=" + my.Username,
        "--verbose",
    }

    if my.UseTLS {
        args = append(args, "--ssl-mode=REQUIRED")
    }

    // Optionally specify target database
    if my.Database != nil && *my.Database != "" {
        args = append(args, *my.Database)
    }

    return uc.restoreFromStorage(
        originalDB,
        tools.GetMysqlExecutable(
            my.Version,
            tools.MysqlExecutableMysql,
            config.GetEnv().EnvMode,
            config.GetEnv().MysqlInstallDir,
        ),
        args,
        my.Password,
        backup,
        storage,
        my,
    )
}

func (uc *RestoreMysqlBackupUsecase) restoreFromStorage(
    database *databases.Database,
    mysqlBin string,
    args []string,
    password string,
    backup *backups.Backup,
    storage *storages.Storage,
    myConfig *mysqltypes.MysqlDatabase,
) error {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
    defer cancel()

    // Monitor for shutdown
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                if config.IsShouldShutdown() {
                    cancel()
                    return
                }
            }
        }
    }()

    // Create temp .my.cnf
    myCnfFile, err := uc.createTempMyCnfFile(myConfig, password)
    if err != nil {
        return fmt.Errorf("failed to create .my.cnf: %w", err)
    }
    defer os.RemoveAll(filepath.Dir(myCnfFile))

    // Download backup to temp file
    tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
    if err != nil {
        return fmt.Errorf("failed to download backup: %w", err)
    }
    defer cleanupFunc()

    // Decompress and pipe to mysql
    return uc.executeMysqlRestore(ctx, database, mysqlBin, args, myCnfFile, tempBackupFile, backup)
}

func (uc *RestoreMysqlBackupUsecase) executeMysqlRestore(
    ctx context.Context,
    database *databases.Database,
    mysqlBin string,
    args []string,
    myCnfFile string,
    backupFile string,
    backup *backups.Backup,
) error {
    fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

    cmd := exec.CommandContext(ctx, mysqlBin, fullArgs...)
    uc.logger.Info("Executing MySQL restore command", "command", cmd.String())

    // Setup stdin pipeline: file -> [decryption] -> gunzip -> mysql stdin
    backupFileHandle, err := os.Open(backupFile)
    if err != nil {
        return fmt.Errorf("failed to open backup file: %w", err)
    }
    defer backupFileHandle.Close()

    var inputReader io.Reader = backupFileHandle

    // Decrypt if needed
    if backup.Encryption == backups_config.BackupEncryptionEncrypted {
        // Setup decryption reader (same as PostgreSQL)
        decryptReader, err := uc.setupDecryption(backupFileHandle, backup)
        if err != nil {
            return fmt.Errorf("failed to setup decryption: %w", err)
        }
        inputReader = decryptReader
    }

    // Decompress (zstd)
    zstdReader, err := zstd.NewReader(inputReader)
    if err != nil {
        return fmt.Errorf("failed to create zstd reader: %w", err)
    }
    defer zstdReader.Close()

    cmd.Stdin = zstdReader

    // Setup environment
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env, "MYSQL_PWD=", "LC_ALL=C.UTF-8")

    // Get stderr for error messages
    stderrPipe, err := cmd.StderrPipe()
    if err != nil {
        return fmt.Errorf("stderr pipe: %w", err)
    }

    stderrCh := make(chan []byte, 1)
    go func() {
        output, _ := io.ReadAll(stderrPipe)
        stderrCh <- output
    }()

    if err = cmd.Start(); err != nil {
        return fmt.Errorf("start mysql: %w", err)
    }

    waitErr := cmd.Wait()
    stderrOutput := <-stderrCh

    if config.IsShouldShutdown() {
        return fmt.Errorf("restore cancelled due to shutdown")
    }

    if waitErr != nil {
        return uc.handleMysqlRestoreError(waitErr, stderrOutput, mysqlBin)
    }

    return nil
}

func (uc *RestoreMysqlBackupUsecase) downloadBackupToTempFile(
    ctx context.Context,
    backup *backups.Backup,
    storage *storages.Storage,
) (string, func(), error) {
    // Same implementation as PostgreSQL
    // Download from storage to temp file
}

func (uc *RestoreMysqlBackupUsecase) createTempMyCnfFile(
    myConfig *mysqltypes.MysqlDatabase,
    password string,
) (string, error) {
    // Same as backup implementation
}
```

### Restore Pipeline:

```
storage -> [decryption] -> gunzip -> mysql client stdin
```

---

## Migration Script

### `backend/migrations/XXXXXX_add_mysql_databases_table.up.sql`

```sql
CREATE TABLE mysql_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    version     TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INT NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    database    TEXT,
    use_tls     BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_mysql_databases_database_id ON mysql_databases(database_id);
```

### `backend/migrations/XXXXXX_add_mysql_databases_table.down.sql`

```sql
DROP INDEX IF EXISTS idx_mysql_databases_database_id;
DROP TABLE IF EXISTS mysql_databases;
```

---

## Model Updates

### `backend/internal/features/databases/model.go` changes

```go
type Database struct {
    // ... existing fields ...

    Postgresql *postgresql.PostgresqlDatabase `json:"postgresql,omitempty" gorm:"foreignKey:DatabaseID"`
    Mysql      *mysql.MysqlDatabase           `json:"mysql,omitempty"      gorm:"foreignKey:DatabaseID"`

    // ... rest of fields ...
}

func (d *Database) Validate() error {
    if d.Name == "" {
        return errors.New("name is required")
    }

    switch d.Type {
    case DatabaseTypePostgres:
        if d.Postgresql == nil {
            return errors.New("postgresql database is required")
        }
        return d.Postgresql.Validate()
    case DatabaseTypeMysql:
        if d.Mysql == nil {
            return errors.New("mysql database is required")
        }
        return d.Mysql.Validate()
    default:
        return errors.New("invalid database type: " + string(d.Type))
    }
}

func (d *Database) TestConnection(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
) error {
    return d.getSpecificDatabase().TestConnection(logger, encryptor, d.ID)
}

func (d *Database) EncryptSensitiveFields(encryptor encryption.FieldEncryptor) error {
    if d.Postgresql != nil {
        return d.Postgresql.EncryptSensitiveFields(d.ID, encryptor)
    }
    if d.Mysql != nil {
        return d.Mysql.EncryptSensitiveFields(d.ID, encryptor)
    }
    return nil
}

func (d *Database) PopulateVersionIfEmpty(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
) error {
    if d.Postgresql != nil {
        return d.Postgresql.PopulateVersionIfEmpty(logger, encryptor, d.ID)
    }
    if d.Mysql != nil {
        return d.Mysql.PopulateVersionIfEmpty(logger, encryptor, d.ID)
    }
    return nil
}

func (d *Database) Update(incoming *Database) {
    d.Name = incoming.Name
    d.Type = incoming.Type
    d.Notifiers = incoming.Notifiers

    switch d.Type {
    case DatabaseTypePostgres:
        if d.Postgresql != nil && incoming.Postgresql != nil {
            d.Postgresql.Update(incoming.Postgresql)
        }
    case DatabaseTypeMysql:
        if d.Mysql != nil && incoming.Mysql != nil {
            d.Mysql.Update(incoming.Mysql)
        }
    }
}

func (d *Database) getSpecificDatabase() DatabaseConnector {
    switch d.Type {
    case DatabaseTypePostgres:
        return d.Postgresql
    case DatabaseTypeMysql:
        return d.Mysql
    }
    panic("invalid database type: " + string(d.Type))
}
```

---

## Service Updates

### `backend/internal/features/databases/service.go` - CopyDatabase

```go
func (s *DatabaseService) CopyDatabase(
    user *users_models.User,
    databaseID uuid.UUID,
) (*Database, error) {
    existingDatabase, err := s.dbRepository.FindByID(databaseID)
    if err != nil {
        return nil, err
    }

    // ... permission checks ...

    newDatabase := &Database{
        ID:                     uuid.Nil,
        WorkspaceID:            existingDatabase.WorkspaceID,
        Name:                   existingDatabase.Name + " (Copy)",
        Type:                   existingDatabase.Type,
        Notifiers:              existingDatabase.Notifiers,
        LastBackupTime:         nil,
        LastBackupErrorMessage: nil,
        HealthStatus:           existingDatabase.HealthStatus,
    }

    switch existingDatabase.Type {
    case DatabaseTypePostgres:
        if existingDatabase.Postgresql != nil {
            newDatabase.Postgresql = &postgresql.PostgresqlDatabase{
                // ... copy fields ...
            }
        }
    case DatabaseTypeMysql:
        if existingDatabase.Mysql != nil {
            newDatabase.Mysql = &mysql.MysqlDatabase{
                ID:         uuid.Nil,
                DatabaseID: nil,
                Version:    existingDatabase.Mysql.Version,
                Host:       existingDatabase.Mysql.Host,
                Port:       existingDatabase.Mysql.Port,
                Username:   existingDatabase.Mysql.Username,
                Password:   existingDatabase.Mysql.Password,
                Database:   existingDatabase.Mysql.Database,
                UseTLS:     existingDatabase.Mysql.UseTLS,
            }
        }
    }

    // ... rest of method ...
}
```

---

## Config Updates

### `backend/internal/config/config.go` additions

```go
type EnvVariables struct {
    // ... existing fields ...

    MysqlInstallDir string `env:"MYSQL_INSTALL_DIR"`

    // Testing MySQL
    TestMysql57Port string `env:"TEST_MYSQL_57_PORT"`
    TestMysql80Port string `env:"TEST_MYSQL_80_PORT"`
    TestMysql84Port string `env:"TEST_MYSQL_84_PORT"`
}

// In loadEnvVariables():
env.MysqlInstallDir = filepath.Join(backendRoot, "tools", "mysql")
tools.VerifyMysqlInstallation(log, env.EnvMode, env.MysqlInstallDir)

if env.IsTesting {
    // ... existing checks ...

    if env.TestMysql57Port == "" {
        log.Error("TEST_MYSQL_57_PORT is empty")
        os.Exit(1)
    }
    if env.TestMysql80Port == "" {
        log.Error("TEST_MYSQL_80_PORT is empty")
        os.Exit(1)
    }
    if env.TestMysql84Port == "" {
        log.Error("TEST_MYSQL_84_PORT is empty")
        os.Exit(1)
    }
}
```

---

## Test Implementation

### `backend/internal/features/tests/mysql_backup_restore_test.go`

```go
package tests

import (
    "database/sql"
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
    "github.com/stretchr/testify/assert"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mysqltypes "postgresus-backend/internal/features/databases/databases/mysql"
    "postgresus-backend/internal/features/restores"
    restores_enums "postgresus-backend/internal/features/restores/enums"
    restores_models "postgresus-backend/internal/features/restores/models"
    "postgresus-backend/internal/features/storages"
    users_enums "postgresus-backend/internal/features/users/enums"
    users_testing "postgresus-backend/internal/features/users/testing"
    workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
    workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
    test_utils "postgresus-backend/internal/util/testing"
)

const createMysqlTestDataQuery = `
DROP TABLE IF EXISTS test_data;

CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300);
`

type MysqlContainer struct {
    Host     string
    Port     int
    Username string
    Password string
    Database string
    Version  string
    DB       *sql.DB
}

func Test_BackupAndRestoreMysql_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version string
        port    string
    }{
        {"MySQL 5.7", "5.7", env.TestMysql57Port},
        {"MySQL 8.0", "8.0", env.TestMysql80Port},
        {"MySQL 8.4", "8.4", env.TestMysql84Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMysqlBackupRestoreForVersion(t, tc.version, tc.port)
        })
    }
}

func Test_BackupAndRestoreMysqlWithEncryption_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version string
        port    string
    }{
        {"MySQL 5.7", "5.7", env.TestMysql57Port},
        {"MySQL 8.0", "8.0", env.TestMysql80Port},
        {"MySQL 8.4", "8.4", env.TestMysql84Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMysqlBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
        })
    }
}

func testMysqlBackupRestoreForVersion(t *testing.T, mysqlVersion string, port string) {
    container, err := connectToMysqlContainer(mysqlVersion, port)
    assert.NoError(t, err)
    defer container.DB.Close()

    _, err = container.DB.Exec(createMysqlTestDataQuery)
    assert.NoError(t, err)

    router := createMysqlTestRouter()
    user := users_testing.CreateTestUser(users_enums.UserRoleMember)
    workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)

    storage := storages.CreateTestStorage(workspace.ID)

    database := createMysqlDatabaseViaAPI(
        t, router, "Test MySQL Database", workspace.ID,
        container.Host, container.Port,
        container.Username, container.Password, container.Database,
        user.Token,
    )

    enableBackupsViaAPI(
        t, router, database.ID, storage.ID,
        backups_config.BackupEncryptionNone, user.Token,
    )

    createBackupViaAPI(t, router, database.ID, user.Token)

    backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
    assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

    // Create new database for restore
    newDBName := "restoreddb"
    _, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
    assert.NoError(t, err)
    _, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
    assert.NoError(t, err)

    createMysqlRestoreViaAPI(
        t, router, backup.ID,
        container.Host, container.Port,
        container.Username, container.Password, newDBName,
        user.Token,
    )

    restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
    assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

    // Verify data integrity
    newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
        container.Username, container.Password, container.Host, container.Port, newDBName)
    newDB, err := sql.Open("mysql", newDSN)
    assert.NoError(t, err)
    defer newDB.Close()

    var count int
    err = newDB.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 3, count)

    // Cleanup
    err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
    if err != nil {
        t.Logf("Warning: Failed to delete backup file: %v", err)
    }

    test_utils.MakeDeleteRequest(
        t, router,
        "/api/v1/databases/"+database.ID.String(),
        "Bearer "+user.Token,
        http.StatusNoContent,
    )
    storages.RemoveTestStorage(storage.ID)
    workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func connectToMysqlContainer(version string, port string) (*MysqlContainer, error) {
    dbName := "testdb"
    password := "testpassword"
    username := "testuser"
    host := "localhost"

    portInt, err := strconv.Atoi(port)
    if err != nil {
        return nil, fmt.Errorf("failed to parse port: %w", err)
    }

    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
        username, password, host, portInt, dbName)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    if err = db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &MysqlContainer{
        Host:     host,
        Port:     portInt,
        Username: username,
        Password: password,
        Database: dbName,
        DB:       db,
    }, nil
}

func createMysqlDatabaseViaAPI(
    t *testing.T,
    router *gin.Engine,
    name string,
    workspaceID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    token string,
) *databases.Database {
    request := databases.Database{
        Name:        name,
        WorkspaceID: &workspaceID,
        Type:        databases.DatabaseTypeMysql,
        Mysql: &mysqltypes.MysqlDatabase{
            Host:     host,
            Port:     port,
            Username: username,
            Password: password,
            Database: &database,
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
        t.Fatalf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String())
    }

    var createdDatabase databases.Database
    if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
        t.Fatalf("Failed to unmarshal database response: %v", err)
    }

    return &createdDatabase
}

func createMysqlRestoreViaAPI(
    t *testing.T,
    router *gin.Engine,
    backupID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    token string,
) {
    request := restores.RestoreBackupRequest{
        MysqlDatabase: &mysqltypes.MysqlDatabase{
            Host:     host,
            Port:     port,
            Username: username,
            Password: password,
            Database: &database,
        },
    }

    test_utils.MakePostRequest(
        t, router,
        fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
        "Bearer "+token,
        request,
        http.StatusOK,
    )
}
```

---

## Docker Compose for Testing

### `backend/docker-compose.test.yml` additions

```yaml
services:
  # ... existing PostgreSQL services ...

  mysql57:
    image: mysql:5.7
    container_name: postgresus-test-mysql57
    ports:
      - "${TEST_MYSQL_57_PORT:-33057}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test:
        [
          "CMD",
          "mysqladmin",
          "ping",
          "-h",
          "localhost",
          "-u",
          "root",
          "-prootpassword",
        ]
      interval: 5s
      timeout: 5s
      retries: 10

  mysql80:
    image: mysql:8.0
    container_name: postgresus-test-mysql80
    ports:
      - "${TEST_MYSQL_80_PORT:-33080}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci --default-authentication-plugin=mysql_native_password
    healthcheck:
      test:
        [
          "CMD",
          "mysqladmin",
          "ping",
          "-h",
          "localhost",
          "-u",
          "root",
          "-prootpassword",
        ]
      interval: 5s
      timeout: 5s
      retries: 10

  mysql84:
    image: mysql:8.4
    container_name: postgresus-test-mysql84
    ports:
      - "${TEST_MYSQL_84_PORT:-33084}:3306"
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpassword
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test:
        [
          "CMD",
          "mysqladmin",
          "ping",
          "-h",
          "localhost",
          "-u",
          "root",
          "-prootpassword",
        ]
      interval: 5s
      timeout: 5s
      retries: 10
```

---

## MySQL Client Tools Installation

### For Docker (Production)

Add to Dockerfile:

```dockerfile
# Install MySQL client tools for all supported versions
RUN apt-get update && apt-get install -y \
    mysql-client-5.7 \
    mysql-client-8.0 \
    && rm -rf /var/lib/apt/lists/*

# Or use official MySQL APT repository for specific versions
RUN wget https://dev.mysql.com/get/mysql-apt-config_0.8.29-1_all.deb \
    && dpkg -i mysql-apt-config_0.8.29-1_all.deb \
    && apt-get update \
    && apt-get install -y mysql-client
```

### For Development (Windows/macOS)

Create `backend/tools/mysql/readme.md`:

````markdown
# MySQL Client Tools Setup

## Windows

1. Download MySQL Community Server (client only) from:
   - MySQL 5.7: https://dev.mysql.com/downloads/mysql/5.7.html
   - MySQL 8.0: https://dev.mysql.com/downloads/mysql/8.0.html
   - MySQL 8.4: https://dev.mysql.com/downloads/mysql/8.4.html

2. Extract to:
   - `backend/tools/mysql/mysql-5.7/bin/`
   - `backend/tools/mysql/mysql-8.0/bin/`
   - `backend/tools/mysql/mysql-8.4/bin/`

3. Required executables:
   - `mysqldump.exe`
   - `mysql.exe`

## macOS (Homebrew)

```bash
brew install mysql-client@5.7
brew install mysql-client@8.0
brew install mysql-client@8.4
```
````

Then symlink or copy to tools directory.

```

---

## Frontend Changes (Overview)

The frontend will need updates to:

1. **Database Type Selection** - Add MySQL option in database creation form
2. **MySQL Connection Form** - Similar to PostgreSQL but with:
   - Host, Port, Username, Password
   - Database name (single database per config)
   - Use TLS checkbox (instead of IsHttps)
3. **Version Display** - Show MySQL version in database details
4. **Restore Form** - Add MySQL target configuration option

---

## Implementation Order

1. **Phase 1: Core Infrastructure**
   - [ ] Create `MysqlVersion` enums and `mysql.go` tools
   - [ ] Create `mysql/model.go` with `MysqlDatabase` struct
   - [ ] Add migration for `mysql_databases` table
   - [ ] Update `databases/model.go` to include MySQL
   - [ ] Update `databases/enums.go` with `DatabaseTypeMysql`
   - [ ] Update `databases/repository.go` to preload MySQL

2. **Phase 2: Backup Implementation**
   - [ ] Create `usecases/mysql/create_backup_uc.go`
   - [ ] Create `usecases/mysql/di.go`
   - [ ] Update `usecases/create_backup_uc.go` to route to MySQL
   - [ ] Update `usecases/di.go` to wire MySQL use case

3. **Phase 3: Restore Implementation**
   - [ ] Create `restores/usecases/mysql/restore_backup_uc.go`
   - [ ] Create `restores/usecases/mysql/di.go`
   - [ ] Update `restores/usecases/restore_backup_uc.go` to route to MySQL
   - [ ] Update `restores/service.go` for MySQL restore DTO

4. **Phase 4: Service Layer Updates**
   - [ ] Update `databases/service.go` - CopyDatabase, IsUserReadOnly, CreateReadOnlyUser
   - [ ] Update `config/config.go` with MySQL configuration
   - [ ] Add MySQL installation verification

5. **Phase 5: Testing**
   - [ ] Add Docker Compose services for MySQL containers
   - [ ] Create `mysql_backup_restore_test.go`
   - [ ] Test all MySQL versions (5.7, 8.0, 8.4)
   - [ ] Test encryption with MySQL backups

6. **Phase 6: Frontend**
   - [ ] Add MySQL database type option
   - [ ] Create MySQL connection form
   - [ ] Update restore dialog for MySQL

---

## Notes and Considerations

### MySQL vs PostgreSQL Differences

| Feature | PostgreSQL | MySQL |
|---------|------------|-------|
| Backup format | Custom (-Fc) with built-in compression | Plain SQL + zstd compression |
| Network compression | N/A (local compression only) | zstd (8.0+) or zlib (5.7) via protocol |
| Parallel dump | Supported (--jobs) | Not in mysqldump (consider mydumper) |
| Schemas | Multiple schemas per database | One schema = one database |
| Password file | .pgpass | .my.cnf |
| SSL flag | IsHttps / PGSSLMODE | UseTLS / --ssl-mode |
| Version detection | SELECT version() | SELECT VERSION() |

### Security Considerations

1. **Password handling**: Use `.my.cnf` temp file, not `MYSQL_PWD` env var (deprecated and insecure)
2. **File permissions**: `.my.cnf` must be 0600
3. **TLS**: Support SSL/TLS connections for cloud databases

### Cloud Database Compatibility

- **AWS RDS**: Works with standard mysqldump
- **Google Cloud SQL**: Works with standard mysqldump
- **Azure Database for MySQL**: Works with standard mysqldump
- **PlanetScale**: May require `--ssl-mode=VERIFY_IDENTITY` and specific TLS configuration

### Compression Strategy

MySQL backup uses **two-layer compression** to minimize both network and storage usage:

#### Layer 1: Network Compression (MySQL Protocol)

Compresses data during transfer between MySQL server and mysqldump client:

| MySQL Version | Algorithm | Flag | Level |
|---------------|-----------|------|-------|
| 5.7 | zlib | `--compress` | N/A (fixed) |
| 8.0+ | zstd | `--compression-algorithms=zstd` | 3 (medium, 1-19 range) |
| 8.0+ fallback | zlib | `--compress` | N/A |

**Why zstd for 8.0+?**
- ~30% faster compression than zlib
- ~10% better compression ratio
- Lower CPU usage

#### Layer 2: Storage Compression (zstd)

Compresses mysqldump SQL output before storage:

```

MySQL Server --[zstd/zlib network]--> mysqldump --[zstd level 3]--> [encryption] --> storage

```

| Setting | Value | Notes |
|---------|-------|-------|
| Algorithm | zstd | `github.com/klauspost/compress/zstd` (pure Go) |
| Level | 3 | Medium (1-19 range, 3 is default/balanced) |

**Why zstd over gzip?**

| Metric | gzip (level 6) | zstd (level 3) | Improvement |
|--------|---------------|----------------|-------------|
| Compression speed | ~25 MB/s | ~300 MB/s | **12x faster** |
| Decompression speed | ~250 MB/s | ~800 MB/s | **3x faster** |
| Compression ratio | ~75% | ~75% | Same |
| CPU usage | High | Low | **Much lower** |

#### Compression Ratio Expectations

| Data Type | Network Compression | Storage Compression | Total Ratio |
|-----------|--------------------|--------------------|-------------|
| Text-heavy tables | 60-70% | 80-90% | 90-95% |
| Mixed data | 40-50% | 70-80% | 85-90% |
| Binary/BLOB data | 10-20% | 20-30% | 30-40% |

#### Consistent Algorithm Benefit

Using zstd for both layers provides:
- Consistent behavior and tuning
- Single dependency (`github.com/klauspost/compress/zstd`)
- Optimal performance for parallel backup scenarios
```
