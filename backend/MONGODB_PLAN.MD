# MongoDB Implementation Plan

## Overview

This document outlines the implementation plan for adding MongoDB backup and restore support to Postgresus. The implementation will follow the existing PostgreSQL, MySQL and MariaDB architecture patterns.

## Supported MongoDB Versions

| Version         | Status  | EOL Date   | Support |
| --------------- | ------- | ---------- | ------- |
| **MongoDB 4.0** | EOL     | April 2022 | ✅ Yes  |
| **MongoDB 4.2** | EOL     | April 2023 | ✅ Yes  |
| **MongoDB 4.4** | EOL     | Feb 2024   | ✅ Yes  |
| **MongoDB 5.0** | EOL     | Oct 2024   | ✅ Yes  |
| **MongoDB 6.0** | Active  | July 2025  | ✅ Yes  |
| **MongoDB 7.0** | LTS     | Aug 2026   | ✅ Yes  |
| **MongoDB 8.0** | Current | TBD        | ✅ Yes  |

**Notes:**

- MongoDB 4.0-4.4 reached EOL but are still supported for legacy systems that cannot be upgraded.
- MongoDB 7.0 is the current LTS (Long Term Support) version.
- MongoDB 8.0 is the latest stable release.

## Key Decisions

| Aspect             | Decision                                                             |
| ------------------ | -------------------------------------------------------------------- |
| **Architecture**   | Separate `DatabaseTypeMongodb`                                       |
| **Client tools**   | Single latest version - backward compatible with all server versions |
| **Backup tool**    | `mongodump` with `--archive` and `--gzip` flags                      |
| **Restore tool**   | `mongorestore` with `--archive` and `--gzip` flags                   |
| **Compression**    | Built-in gzip (`--gzip` flag) - no external compression needed       |
| **Auto-detection** | Parse version from `db.version()` or `buildInfo` command             |
| **Cross-restore**  | Yes (newer mongorestore can restore older backups)                   |
| **ARM64**          | Full support (all versions)                                          |

---

## File Structure

### New Files to Create

```
backend/internal/features/databases/databases/mongodb/
├── model.go                    # MongodbDatabase struct and methods

backend/internal/features/backups/backups/usecases/mongodb/
├── create_backup_uc.go         # MongoDB backup use case
├── di.go                       # Dependency injection

backend/internal/features/restores/usecases/mongodb/
├── restore_backup_uc.go        # MongoDB restore use case
├── di.go                       # Dependency injection

backend/internal/util/tools/
├── mongodb.go                  # MongoDB executable helpers and version enums

backend/internal/features/tests/
├── mongodb_backup_restore_test.go  # Integration tests

backend/internal/features/databases/databases/mongodb/
├── readonly_user_test.go           # Read-only user integration tests

backend/migrations/
├── XXXXXX_add_mongodb_databases_table.up.sql
├── XXXXXX_add_mongodb_databases_table.down.sql
```

### Files to Modify

```
backend/internal/features/databases/
├── enums.go                    # Add DatabaseTypeMongodb
├── model.go                    # Add Mongodb field, update methods
├── service.go                  # Handle MongoDB in CopyDatabase, IsUserReadOnly, CreateReadOnlyUser
├── repository.go               # Preload MongoDB relation
├── controller_test.go          # Add MongoDB test cases for CRUD and sensitive data lifecycle

backend/internal/features/backups/backups/usecases/
├── create_backup_uc.go         # Add MongoDB case
├── di.go                       # Wire MongoDB use case

backend/internal/features/restores/usecases/
├── restore_backup_uc.go        # Add MongoDB case
├── di.go                       # Wire MongoDB use case

backend/internal/features/restores/
├── service.go                  # Add MongoDB validation, version compatibility
├── dto.go                      # Add MongodbDatabase field to RestoreBackupRequest

backend/internal/features/healthcheck/attempt/
├── check_database_health_uc.go # Add MongoDB case in validateDatabase()

backend/internal/config/
├── config.go                   # Add MongoDB test ports, verify MongoDB installation

backend/tools/
├── download_linux.sh           # Add MongoDB tools download
├── download_macos.sh           # Add MongoDB tools download
├── download_windows.bat        # Add MongoDB tools download
├── readme.md                   # Update with MongoDB instructions

Dockerfile                      # Add MongoDB tools installation
docker-compose.yml.example      # Add MongoDB test containers
.github/workflows/ci-release.yml # Add MongoDB test setup
```

---

## Database Type Enum

### `backend/internal/features/databases/enums.go`

```go
const (
    DatabaseTypePostgres DatabaseType = "POSTGRES"
    DatabaseTypeMysql    DatabaseType = "MYSQL"
    DatabaseTypeMariadb  DatabaseType = "MARIADB"
    DatabaseTypeMongodb  DatabaseType = "MONGODB"
)
```

---

## MongoDB Version Enums

### `backend/internal/util/tools/mongodb.go`

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
                "path", binDir,
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
                    "command", cmd,
                    "path", cmdPath,
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
```

---

## Model Definition

### `backend/internal/features/databases/databases/mongodb/model.go`

```go
package mongodb

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "regexp"
    "time"

    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"

    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbDatabase struct {
    ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

    Version tools.MongodbVersion `json:"version" gorm:"type:text;not null"`

    Host         string `json:"host"         gorm:"type:text;not null"`
    Port         int    `json:"port"         gorm:"type:int;not null"`
    Username     string `json:"username"     gorm:"type:text;not null"`
    Password     string `json:"password"     gorm:"type:text;not null"`
    Database     string `json:"database"     gorm:"type:text;not null"`
    AuthDatabase string `json:"authDatabase" gorm:"type:text;not null;default:'admin'"`
    UseTLS       bool   `json:"useTls"       gorm:"type:boolean;default:false"`
}

func (m *MongodbDatabase) TableName() string {
    return "mongodb_databases"
}

func (m *MongodbDatabase) Validate() error {
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
    if m.Database == "" {
        return errors.New("database is required")
    }
    return nil
}

func (m *MongodbDatabase) TestConnection(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return fmt.Errorf("failed to decrypt password: %w", err)
    }

    uri := m.buildConnectionURI(password)

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return fmt.Errorf("failed to connect to MongoDB: %w", err)
    }
    defer func() {
        if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
            logger.Error("Failed to disconnect from MongoDB", "error", disconnectErr)
        }
    }()

    if err := client.Ping(ctx, nil); err != nil {
        return fmt.Errorf("failed to ping MongoDB database '%s': %w", m.Database, err)
    }

    // Detect version
    detectedVersion, err := detectMongodbVersion(ctx, client)
    if err != nil {
        return err
    }
    m.Version = detectedVersion

    return nil
}

func (m *MongodbDatabase) HideSensitiveData() {
    if m == nil {
        return
    }
    m.Password = ""
}

func (m *MongodbDatabase) Update(incoming *MongodbDatabase) {
    m.Version = incoming.Version
    m.Host = incoming.Host
    m.Port = incoming.Port
    m.Username = incoming.Username
    m.Database = incoming.Database
    m.AuthDatabase = incoming.AuthDatabase
    m.UseTLS = incoming.UseTLS

    if incoming.Password != "" {
        m.Password = incoming.Password
    }
}

func (m *MongodbDatabase) EncryptSensitiveFields(
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

func (m *MongodbDatabase) PopulateVersionIfEmpty(
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) error {
    if m.Version != "" {
        return nil
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return fmt.Errorf("failed to decrypt password: %w", err)
    }

    uri := m.buildConnectionURI(password)

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
            logger.Error("Failed to disconnect", "error", disconnectErr)
        }
    }()

    detectedVersion, err := detectMongodbVersion(ctx, client)
    if err != nil {
        return err
    }

    m.Version = detectedVersion
    return nil
}

func (m *MongodbDatabase) IsUserReadOnly(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (bool, error) {
    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return false, fmt.Errorf("failed to decrypt password: %w", err)
    }

    uri := m.buildConnectionURI(password)

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return false, fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
            logger.Error("Failed to disconnect", "error", disconnectErr)
        }
    }()

    // Check user roles using usersInfo command
    adminDB := client.Database(m.AuthDatabase)
    var result bson.M
    err = adminDB.RunCommand(ctx, bson.D{
        {Key: "usersInfo", Value: bson.D{
            {Key: "user", Value: m.Username},
            {Key: "db", Value: m.AuthDatabase},
        }},
    }).Decode(&result)
    if err != nil {
        return false, fmt.Errorf("failed to get user info: %w", err)
    }

    // Check if user has any write roles
    writeRoles := []string{
        "readWrite", "readWriteAnyDatabase", "dbAdmin", "dbAdminAnyDatabase",
        "userAdmin", "userAdminAnyDatabase", "clusterAdmin", "root",
        "dbOwner", "backup", "restore",
    }

    users, ok := result["users"].(bson.A)
    if !ok || len(users) == 0 {
        return true, nil // User not found, assume read-only
    }

    user := users[0].(bson.M)
    roles, ok := user["roles"].(bson.A)
    if !ok {
        return true, nil
    }

    for _, roleDoc := range roles {
        role := roleDoc.(bson.M)
        roleName, _ := role["role"].(string)
        for _, writeRole := range writeRoles {
            if roleName == writeRole {
                return false, nil // Has write access
            }
        }
    }

    return true, nil
}

func (m *MongodbDatabase) CreateReadOnlyUser(
    ctx context.Context,
    logger *slog.Logger,
    encryptor encryption.FieldEncryptor,
    databaseID uuid.UUID,
) (string, string, error) {
    password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
    if err != nil {
        return "", "", fmt.Errorf("failed to decrypt password: %w", err)
    }

    uri := m.buildConnectionURI(password)

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return "", "", fmt.Errorf("failed to connect to database: %w", err)
    }
    defer func() {
        if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
            logger.Error("Failed to disconnect", "error", disconnectErr)
        }
    }()

    maxRetries := 3
    for attempt := range maxRetries {
        newUsername := fmt.Sprintf("postgresus-%s", uuid.New().String()[:8])
        newPassword := uuid.New().String()

        // Create user with backup role (allows mongodump)
        adminDB := client.Database(m.AuthDatabase)
        err = adminDB.RunCommand(ctx, bson.D{
            {Key: "createUser", Value: newUsername},
            {Key: "pwd", Value: newPassword},
            {Key: "roles", Value: bson.A{
                bson.D{
                    {Key: "role", Value: "backup"},
                    {Key: "db", Value: "admin"},
                },
                bson.D{
                    {Key: "role", Value: "read"},
                    {Key: "db", Value: m.Database},
                },
            }},
        }).Err()

        if err != nil {
            if attempt < maxRetries-1 {
                continue
            }
            return "", "", fmt.Errorf("failed to create user: %w", err)
        }

        logger.Info(
            "Read-only MongoDB user created successfully",
            "username", newUsername,
        )
        return newUsername, newPassword, nil
    }

    return "", "", errors.New("failed to generate unique username after 3 attempts")
}

// buildConnectionURI builds a MongoDB connection URI
func (m *MongodbDatabase) buildConnectionURI(password string) string {
    authDB := m.AuthDatabase
    if authDB == "" {
        authDB = "admin"
    }

    tlsOption := "false"
    if m.UseTLS {
        tlsOption = "true"
    }

    return fmt.Sprintf(
        "mongodb://%s:%s@%s:%d/%s?authSource=%s&tls=%s&connectTimeoutMS=15000",
        m.Username,
        password,
        m.Host,
        m.Port,
        m.Database,
        authDB,
        tlsOption,
    )
}

// BuildMongodumpURI builds a URI suitable for mongodump (without database in path)
func (m *MongodbDatabase) BuildMongodumpURI(password string) string {
    authDB := m.AuthDatabase
    if authDB == "" {
        authDB = "admin"
    }

    tlsOption := "false"
    if m.UseTLS {
        tlsOption = "true"
    }

    return fmt.Sprintf(
        "mongodb://%s:%s@%s:%d/?authSource=%s&tls=%s&connectTimeoutMS=15000",
        m.Username,
        password,
        m.Host,
        m.Port,
        authDB,
        tlsOption,
    )
}

// detectMongodbVersion gets MongoDB server version from buildInfo command
func detectMongodbVersion(ctx context.Context, client *mongo.Client) (tools.MongodbVersion, error) {
    adminDB := client.Database("admin")
    var result bson.M
    err := adminDB.RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
    if err != nil {
        return "", fmt.Errorf("failed to get MongoDB version: %w", err)
    }

    versionStr, ok := result["version"].(string)
    if !ok {
        return "", errors.New("could not parse MongoDB version from buildInfo")
    }

    // Parse version string (e.g., "7.0.14", "8.0.3", "4.4.29")
    re := regexp.MustCompile(`^(\d+)\.(\d+)`)
    matches := re.FindStringSubmatch(versionStr)
    if len(matches) < 3 {
        return "", fmt.Errorf("could not parse MongoDB version: %s", versionStr)
    }

    major := matches[1]
    minor := matches[2]
    versionKey := fmt.Sprintf("%s.%s", major, minor)

    switch versionKey {
    case "4.0":
        return tools.MongodbVersion40, nil
    case "4.2":
        return tools.MongodbVersion42, nil
    case "4.4":
        return tools.MongodbVersion44, nil
    case "5.0":
        return tools.MongodbVersion50, nil
    case "6.0":
        return tools.MongodbVersion60, nil
    case "7.0":
        return tools.MongodbVersion70, nil
    case "8.0":
        return tools.MongodbVersion80, nil
    default:
        return "", fmt.Errorf("unsupported MongoDB version: %s (supported: 4.0, 4.2, 4.4, 5.0, 6.0, 7.0, 8.0)", versionKey)
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

### `backend/internal/features/backups/backups/usecases/mongodb/create_backup_uc.go`

```go
package usecases_mongodb

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
    "time"

    "github.com/google/uuid"

    "postgresus-backend/internal/config"
    backup_encryption "postgresus-backend/internal/features/backups/backups/encryption"
    usecases_common "postgresus-backend/internal/features/backups/backups/usecases/common"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mongodbtypes "postgresus-backend/internal/features/databases/databases/mongodb"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/storages"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/tools"
)

const (
    backupTimeout            = 23 * time.Hour
    shutdownCheckInterval    = 1 * time.Second
    copyBufferSize           = 8 * 1024 * 1024
    progressReportIntervalMB = 1.0
)

type CreateMongodbBackupUsecase struct {
    logger           *slog.Logger
    secretKeyService *encryption_secrets.SecretKeyService
    fieldEncryptor   encryption.FieldEncryptor
}

func (uc *CreateMongodbBackupUsecase) Execute(
    ctx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    db *databases.Database,
    storage *storages.Storage,
    backupProgressListener func(completedMBs float64),
) (*usecases_common.BackupMetadata, error) {
    uc.logger.Info(
        "Creating MongoDB backup via mongodump",
        "databaseId", db.ID,
        "storageId", storage.ID,
    )

    if !backupConfig.IsBackupsEnabled {
        return nil, fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
    }

    mdb := db.Mongodb
    if mdb == nil {
        return nil, fmt.Errorf("mongodb database configuration is required")
    }

    if mdb.Database == "" {
        return nil, fmt.Errorf("database name is required for mongodump backups")
    }

    decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, mdb.Password)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt database password: %w", err)
    }

    args := uc.buildMongodumpArgs(mdb, decryptedPassword)

    return uc.streamToStorage(
        ctx,
        backupID,
        backupConfig,
        tools.GetMongodbExecutable(
            tools.MongodbExecutableMongodump,
            config.GetEnv().EnvMode,
            config.GetEnv().MongodbInstallDir,
        ),
        args,
        storage,
        backupProgressListener,
    )
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpArgs(
    mdb *mongodbtypes.MongodbDatabase,
    password string,
) []string {
    uri := mdb.BuildMongodumpURI(password)

    args := []string{
        "--uri=" + uri,
        "--db=" + mdb.Database,
        "--archive",  // Output to stdout as archive
        "--gzip",     // Built-in gzip compression
    }

    return args
}

func (uc *CreateMongodbBackupUsecase) streamToStorage(
    parentCtx context.Context,
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    mongodumpBin string,
    args []string,
    storage *storages.Storage,
    backupProgressListener func(completedMBs float64),
) (*usecases_common.BackupMetadata, error) {
    uc.logger.Info("Streaming MongoDB backup to storage", "mongodumpBin", mongodumpBin)

    ctx, cancel := uc.createBackupContext(parentCtx)
    defer cancel()

    cmd := exec.CommandContext(ctx, mongodumpBin, args...)

    // Log command without password
    safeArgs := make([]string, len(args))
    for i, arg := range args {
        if len(arg) > 6 && arg[:6] == "--uri=" {
            safeArgs[i] = "--uri=mongodb://***:***@***"
        } else {
            safeArgs[i] = arg
        }
    }
    uc.logger.Info("Executing MongoDB backup command", "command", mongodumpBin, "args", safeArgs)

    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env,
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

    // Setup encryption if enabled
    // Note: mongodump --gzip already compresses, so we don't add another compression layer
    finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
        backupID,
        backupConfig,
        storageWriter,
    )
    if err != nil {
        return nil, err
    }

    countingWriter := usecases_common.NewCountingWriter(finalWriter)

    saveErrCh := make(chan error, 1)
    go func() {
        saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
        saveErrCh <- saveErr
    }()

    if err = cmd.Start(); err != nil {
        return nil, fmt.Errorf("start %s: %w", filepath.Base(mongodumpBin), err)
    }

    copyResultCh := make(chan error, 1)
    bytesWrittenCh := make(chan int64, 1)
    go func() {
        bytesWritten, copyErr := uc.copyWithShutdownCheck(
            ctx,
            countingWriter,
            pgStdout,
            backupProgressListener,
        )
        bytesWrittenCh <- bytesWritten
        copyResultCh <- copyErr
    }()

    copyErr := <-copyResultCh
    bytesWritten := <-bytesWrittenCh
    waitErr := cmd.Wait()

    select {
    case <-ctx.Done():
        uc.cleanupOnCancellation(encryptionWriter, storageWriter, saveErrCh)
        return nil, uc.checkCancellationReason()
    default:
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
        return nil, uc.buildMongodumpErrorMessage(waitErr, stderrOutput, mongodumpBin)
    case copyErr != nil:
        return nil, fmt.Errorf("copy to storage: %w", copyErr)
    case saveErr != nil:
        return nil, fmt.Errorf("save to storage: %w", saveErr)
    }

    return &backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) createBackupContext(
    parentCtx context.Context,
) (context.Context, context.CancelFunc) {
    ctx, cancel := context.WithTimeout(parentCtx, backupTimeout)

    go func() {
        ticker := time.NewTicker(shutdownCheckInterval)
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

    return ctx, cancel
}

func (uc *CreateMongodbBackupUsecase) setupBackupEncryption(
    backupID uuid.UUID,
    backupConfig *backups_config.BackupConfig,
    storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, usecases_common.BackupMetadata, error) {
    backupMetadata := usecases_common.BackupMetadata{
        Encryption: backups_config.BackupEncryptionNone,
    }

    if backupConfig.Encryption != backups_config.BackupEncryptionEncrypted {
        return storageWriter, nil, backupMetadata, nil
    }

    secretKey, err := uc.secretKeyService.GetOrCreateSecretKey(backupConfig.ProjectID)
    if err != nil {
        return nil, nil, backupMetadata, fmt.Errorf("failed to get secret key: %w", err)
    }

    encryptionWriter, salt, nonce, err := backup_encryption.NewEncryptionWriter(
        storageWriter,
        secretKey,
    )
    if err != nil {
        return nil, nil, backupMetadata, fmt.Errorf("failed to create encryption writer: %w", err)
    }

    saltBase64 := base64.StdEncoding.EncodeToString(salt)
    nonceBase64 := base64.StdEncoding.EncodeToString(nonce)

    backupMetadata.Encryption = backups_config.BackupEncryptionEncrypted
    backupMetadata.EncryptionSalt = &saltBase64
    backupMetadata.EncryptionIV = &nonceBase64

    return encryptionWriter, encryptionWriter, backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) copyWithShutdownCheck(
    ctx context.Context,
    dst io.Writer,
    src io.Reader,
    backupProgressListener func(completedMBs float64),
) (int64, error) {
    buf := make([]byte, copyBufferSize)
    var totalWritten int64
    var lastReportedMB float64

    for {
        select {
        case <-ctx.Done():
            return totalWritten, ctx.Err()
        default:
        }

        if config.IsShouldShutdown() {
            return totalWritten, errors.New("shutdown requested")
        }

        nr, readErr := src.Read(buf)
        if nr > 0 {
            nw, writeErr := dst.Write(buf[:nr])
            if nw > 0 {
                totalWritten += int64(nw)

                if backupProgressListener != nil {
                    currentMB := float64(totalWritten) / (1024 * 1024)
                    if currentMB-lastReportedMB >= progressReportIntervalMB {
                        backupProgressListener(currentMB)
                        lastReportedMB = currentMB
                    }
                }
            }
            if writeErr != nil {
                return totalWritten, writeErr
            }
            if nr != nw {
                return totalWritten, io.ErrShortWrite
            }
        }
        if readErr != nil {
            if readErr == io.EOF {
                return totalWritten, nil
            }
            return totalWritten, readErr
        }
    }
}

func (uc *CreateMongodbBackupUsecase) cleanupOnCancellation(
    encryptionWriter *backup_encryption.EncryptionWriter,
    storageWriter *io.PipeWriter,
    saveErrCh chan error,
) {
    if encryptionWriter != nil {
        _ = encryptionWriter.Close()
    }
    _ = storageWriter.CloseWithError(errors.New("backup cancelled"))
    <-saveErrCh
}

func (uc *CreateMongodbBackupUsecase) closeWriters(
    encryptionWriter *backup_encryption.EncryptionWriter,
    storageWriter *io.PipeWriter,
) error {
    if encryptionWriter != nil {
        if err := encryptionWriter.Close(); err != nil {
            uc.logger.Error("Failed to close encryption writer", "error", err)
            return fmt.Errorf("failed to close encryption writer: %w", err)
        }
    }
    if err := storageWriter.Close(); err != nil {
        uc.logger.Error("Failed to close storage writer", "error", err)
        return fmt.Errorf("failed to close storage writer: %w", err)
    }
    return nil
}

func (uc *CreateMongodbBackupUsecase) checkCancellationReason() error {
    if config.IsShouldShutdown() {
        return errors.New("backup cancelled due to shutdown")
    }
    return errors.New("backup cancelled due to timeout")
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpErrorMessage(
    waitErr error,
    stderrOutput []byte,
    mongodumpBin string,
) error {
    stderrStr := string(stderrOutput)

    if len(stderrStr) > 0 {
        return fmt.Errorf("%s failed: %w\nstderr: %s", filepath.Base(mongodumpBin), waitErr, stderrStr)
    }

    return fmt.Errorf("%s failed: %w", filepath.Base(mongodumpBin), waitErr)
}
```

### `backend/internal/features/backups/backups/usecases/mongodb/di.go`

```go
package usecases_mongodb

import (
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/logger"
)

var createMongodbBackupUsecase *CreateMongodbBackupUsecase

func init() {
    createMongodbBackupUsecase = &CreateMongodbBackupUsecase{
        logger:           logger.GetLogger(),
        secretKeyService: encryption_secrets.GetSecretKeyService(),
        fieldEncryptor:   encryption.GetFieldEncryptor(),
    }
}

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
    return createMongodbBackupUsecase
}
```

---

## Restore Implementation

### `backend/internal/features/restores/usecases/mongodb/restore_backup_uc.go`

```go
package usecases_mongodb

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
    "time"

    "github.com/google/uuid"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    "postgresus-backend/internal/features/backups/backups/encryption"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mongodbtypes "postgresus-backend/internal/features/databases/databases/mongodb"
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/features/restores/models"
    "postgresus-backend/internal/features/storages"
    util_encryption "postgresus-backend/internal/util/encryption"
    files_utils "postgresus-backend/internal/util/files"
    "postgresus-backend/internal/util/tools"
)

const (
    restoreTimeout = 60 * time.Minute
)

type RestoreMongodbBackupUsecase struct {
    logger           *slog.Logger
    secretKeyService *encryption_secrets.SecretKeyService
    fieldEncryptor   util_encryption.FieldEncryptor
}

func (uc *RestoreMongodbBackupUsecase) Execute(
    originalDB *databases.Database,
    restoringToDB *databases.Database,
    backupConfig *backups_config.BackupConfig,
    restore models.Restore,
    backup *backups.Backup,
    storage *storages.Storage,
) error {
    if originalDB.Type != databases.DatabaseTypeMongodb {
        return errors.New("database type not supported")
    }

    uc.logger.Info(
        "Restoring MongoDB backup via mongorestore",
        "restoreId", restore.ID,
        "backupId", backup.ID,
    )

    mdb := restoringToDB.Mongodb
    if mdb == nil {
        return fmt.Errorf("mongodb configuration is required for restore")
    }

    if mdb.Database == "" {
        return fmt.Errorf("target database name is required for mongorestore")
    }

    decryptedPassword, err := uc.fieldEncryptor.Decrypt(restoringToDB.ID, mdb.Password)
    if err != nil {
        return fmt.Errorf("failed to decrypt password: %w", err)
    }

    args := uc.buildMongorestoreArgs(mdb, decryptedPassword, originalDB.Mongodb.Database)

    return uc.restoreFromStorage(
        originalDB,
        tools.GetMongodbExecutable(
            tools.MongodbExecutableMongorestore,
            config.GetEnv().EnvMode,
            config.GetEnv().MongodbInstallDir,
        ),
        args,
        backup,
        storage,
    )
}

func (uc *RestoreMongodbBackupUsecase) buildMongorestoreArgs(
    mdb *mongodbtypes.MongodbDatabase,
    password string,
    sourceDatabase string,
) []string {
    uri := mdb.BuildMongodumpURI(password)

    args := []string{
        "--uri=" + uri,
        "--archive",  // Read from stdin as archive
        "--gzip",     // Input is gzip compressed
        "--nsFrom=" + sourceDatabase + ".*",
        "--nsTo=" + mdb.Database + ".*",
        "--drop",     // Drop existing collections before restore
    }

    return args
}

func (uc *RestoreMongodbBackupUsecase) restoreFromStorage(
    database *databases.Database,
    mongorestoreBin string,
    args []string,
    backup *backups.Backup,
    storage *storages.Storage,
) error {
    ctx, cancel := context.WithTimeout(context.Background(), restoreTimeout)
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

    // Download backup to temp file
    tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
    if err != nil {
        return fmt.Errorf("failed to download backup: %w", err)
    }
    defer cleanupFunc()

    return uc.executeMongoRestore(ctx, mongorestoreBin, args, tempBackupFile, backup)
}

func (uc *RestoreMongodbBackupUsecase) executeMongoRestore(
    ctx context.Context,
    mongorestoreBin string,
    args []string,
    backupFile string,
    backup *backups.Backup,
) error {
    cmd := exec.CommandContext(ctx, mongorestoreBin, args...)

    // Log command without password
    safeArgs := make([]string, len(args))
    for i, arg := range args {
        if len(arg) > 6 && arg[:6] == "--uri=" {
            safeArgs[i] = "--uri=mongodb://***:***@***"
        } else {
            safeArgs[i] = arg
        }
    }
    uc.logger.Info("Executing MongoDB restore command", "command", mongorestoreBin, "args", safeArgs)

    // Open backup file
    backupFileHandle, err := os.Open(backupFile)
    if err != nil {
        return fmt.Errorf("failed to open backup file: %w", err)
    }
    defer func() { _ = backupFileHandle.Close() }()

    var inputReader io.Reader = backupFileHandle

    // Decrypt if needed
    if backup.Encryption == backups_config.BackupEncryptionEncrypted {
        decryptReader, err := uc.setupDecryption(backupFileHandle, backup)
        if err != nil {
            return fmt.Errorf("failed to setup decryption: %w", err)
        }
        inputReader = decryptReader
    }

    // Note: mongorestore expects gzip-compressed archive, which is already the format
    // No additional decompression needed since --gzip flag handles it

    cmd.Stdin = inputReader
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8", "LANG=C.UTF-8")

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
        return fmt.Errorf("start mongorestore: %w", err)
    }

    waitErr := cmd.Wait()
    stderrOutput := <-stderrCh

    if config.IsShouldShutdown() {
        return fmt.Errorf("restore cancelled due to shutdown")
    }

    if waitErr != nil {
        return uc.handleMongoRestoreError(waitErr, stderrOutput, mongorestoreBin)
    }

    return nil
}

func (uc *RestoreMongodbBackupUsecase) downloadBackupToTempFile(
    ctx context.Context,
    backup *backups.Backup,
    storage *storages.Storage,
) (string, func(), error) {
    tempDir := config.GetEnv().TempFolder
    tempFile := filepath.Join(tempDir, fmt.Sprintf("restore-%s.archive.gz", backup.ID.String()))

    reader, err := storage.GetFile(ctx, uc.fieldEncryptor, uc.logger, backup.ID)
    if err != nil {
        return "", nil, fmt.Errorf("failed to get backup from storage: %w", err)
    }
    defer func() { _ = reader.Close() }()

    file, err := os.Create(tempFile)
    if err != nil {
        return "", nil, fmt.Errorf("failed to create temp file: %w", err)
    }

    _, err = io.Copy(file, reader)
    if err != nil {
        _ = file.Close()
        _ = os.Remove(tempFile)
        return "", nil, fmt.Errorf("failed to write backup to temp file: %w", err)
    }

    if err := file.Close(); err != nil {
        _ = os.Remove(tempFile)
        return "", nil, fmt.Errorf("failed to close temp file: %w", err)
    }

    cleanup := func() {
        if err := os.Remove(tempFile); err != nil {
            uc.logger.Warn("Failed to remove temp backup file", "file", tempFile, "error", err)
        }
    }

    return tempFile, cleanup, nil
}

func (uc *RestoreMongodbBackupUsecase) setupDecryption(
    reader io.Reader,
    backup *backups.Backup,
) (io.Reader, error) {
    if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
        return nil, errors.New("encrypted backup missing salt or IV")
    }

    salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
    if err != nil {
        return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
    }

    nonce, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
    if err != nil {
        return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
    }

    secretKey, err := uc.secretKeyService.GetSecretKey(backup.ProjectID)
    if err != nil {
        return nil, fmt.Errorf("failed to get secret key: %w", err)
    }

    decryptReader, err := encryption.NewDecryptionReader(reader, secretKey, salt, nonce)
    if err != nil {
        return nil, fmt.Errorf("failed to create decryption reader: %w", err)
    }

    return decryptReader, nil
}

func (uc *RestoreMongodbBackupUsecase) handleMongoRestoreError(
    waitErr error,
    stderrOutput []byte,
    mongorestoreBin string,
) error {
    stderrStr := string(stderrOutput)

    if len(stderrStr) > 0 {
        return fmt.Errorf("%s failed: %w\nstderr: %s", filepath.Base(mongorestoreBin), waitErr, stderrStr)
    }

    return fmt.Errorf("%s failed: %w", filepath.Base(mongorestoreBin), waitErr)
}
```

### `backend/internal/features/restores/usecases/mongodb/di.go`

```go
package usecases_mongodb

import (
    encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
    "postgresus-backend/internal/util/encryption"
    "postgresus-backend/internal/util/logger"
)

var restoreMongodbBackupUsecase *RestoreMongodbBackupUsecase

func init() {
    restoreMongodbBackupUsecase = &RestoreMongodbBackupUsecase{
        logger:           logger.GetLogger(),
        secretKeyService: encryption_secrets.GetSecretKeyService(),
        fieldEncryptor:   encryption.GetFieldEncryptor(),
    }
}

func GetRestoreMongodbBackupUsecase() *RestoreMongodbBackupUsecase {
    return restoreMongodbBackupUsecase
}
```

---

## Migration Script

### `backend/migrations/XXXXXX_add_mongodb_databases_table.up.sql`

```sql
CREATE TABLE mongodb_databases (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id   UUID REFERENCES databases(id) ON DELETE CASCADE,
    version       TEXT NOT NULL,
    host          TEXT NOT NULL,
    port          INT NOT NULL,
    username      TEXT NOT NULL,
    password      TEXT NOT NULL,
    database      TEXT NOT NULL,
    auth_database TEXT NOT NULL DEFAULT 'admin',
    use_tls       BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_mongodb_databases_database_id ON mongodb_databases(database_id);
```

### `backend/migrations/XXXXXX_add_mongodb_databases_table.down.sql`

```sql
DROP INDEX IF EXISTS idx_mongodb_databases_database_id;
DROP TABLE IF EXISTS mongodb_databases;
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
    Mongodb    *mongodb.MongodbDatabase       `json:"mongodb,omitempty"    gorm:"foreignKey:DatabaseID"`

    // ... rest of fields ...
}
```

Update methods to handle MongoDB case:

```go
func (d *Database) Validate() error {
    // ... existing code ...
    case DatabaseTypeMongodb:
        if d.Mongodb == nil {
            return errors.New("mongodb database is required")
        }
        return d.Mongodb.Validate()
    // ...
}

func (d *Database) TestConnection(...) error {
    // ... add Mongodb case
}

func (d *Database) EncryptSensitiveFields(...) error {
    // ... add Mongodb case
}

func (d *Database) PopulateVersionIfEmpty(...) error {
    // ... add Mongodb case
}

func (d *Database) Update(incoming *Database) {
    // ... add Mongodb case
}

func (d *Database) HideSensitiveData() {
    // ... add Mongodb case
}
```

---

## Service Updates

### `backend/internal/features/databases/service.go` - CopyDatabase

Add MongoDB case:

```go
case DatabaseTypeMongodb:
    if existingDatabase.Mongodb != nil {
        newDatabase.Mongodb = &mongodb.MongodbDatabase{
            ID:           uuid.Nil,
            DatabaseID:   nil,
            Version:      existingDatabase.Mongodb.Version,
            Host:         existingDatabase.Mongodb.Host,
            Port:         existingDatabase.Mongodb.Port,
            Username:     existingDatabase.Mongodb.Username,
            Password:     existingDatabase.Mongodb.Password,
            Database:     existingDatabase.Mongodb.Database,
            AuthDatabase: existingDatabase.Mongodb.AuthDatabase,
            UseTLS:       existingDatabase.Mongodb.UseTLS,
        }
    }
```

### `backend/internal/features/databases/service.go` - IsUserReadOnly

Add MongoDB case:

```go
case DatabaseTypeMongodb:
    return usingDatabase.Mongodb.IsUserReadOnly(
        ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
    )
```

### `backend/internal/features/databases/service.go` - CreateReadOnlyUser

Add MongoDB case:

```go
case DatabaseTypeMongodb:
    username, password, err = usingDatabase.Mongodb.CreateReadOnlyUser(
        ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
    )
```

---

## Repository Updates

### `backend/internal/features/databases/repository.go`

Add MongoDB preload and handling:

```go
func (r *DatabaseRepository) FindByID(id uuid.UUID) (*Database, error) {
    // ... existing code ...
    Preload("Mongodb").
    // ...
}

func (r *DatabaseRepository) Save(database *Database) (*Database, error) {
    // ... add case for DatabaseTypeMongodb
    case DatabaseTypeMongodb:
        if database.Mongodb == nil {
            return errors.New("mongodb configuration is required for MongoDB database")
        }
        database.Mongodb.DatabaseID = &database.ID
    // ...
}

func (r *DatabaseRepository) Delete(id uuid.UUID) error {
    // ... add case for DatabaseTypeMongodb
    case DatabaseTypeMongodb:
        if err := tx.
            Where("database_id = ?", id).
            Delete(&mongodb.MongodbDatabase{}).Error; err != nil {
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
    MongodbDatabase    *mongodb.MongodbDatabase       `json:"mongodb,omitempty"`
}
```

### `backend/internal/features/restores/service.go`

Add version validation for MongoDB:

```go
func (s *RestoreService) validateVersionCompatibility(...) error {
    // ... existing cases ...
    case databases.DatabaseTypeMongodb:
        if requestDTO.MongodbDatabase == nil {
            return errors.New("mongodb database configuration is required for restore")
        }
        if tools.IsMongodbBackupVersionHigherThanRestoreVersion(
            backupDatabase.Mongodb.Version,
            requestDTO.MongodbDatabase.Version,
        ) {
            return errors.New(`backup database version is higher than restore database version. ` +
                `Should be restored to the same version as the backup database or higher. ` +
                `For example, you can restore MongoDB 6.0 backup to MongoDB 6.0, 7.0 or 8.0. But cannot restore to 5.0`)
        }
    // ...
}
```

Add MongoDB restore handling:

```go
func (s *RestoreService) RestoreBackup(...) error {
    // ... existing code ...
    case databases.DatabaseTypeMongodb:
        if requestDTO.MongodbDatabase == nil {
            return errors.New("mongodb database is required")
        }
    // ...
}
```

---

## Healthcheck Updates

### `backend/internal/features/healthcheck/attempt/check_database_health_uc.go`

Add MongoDB case to `validateDatabase()`:

```go
func (uc *CheckDatabaseHealthUseCase) validateDatabase(
    database *databases.Database,
) error {
    switch database.Type {
    case databases.DatabaseTypePostgres:
        if database.Postgresql == nil {
            return fmt.Errorf("database Postgresql config is not set")
        }
    case databases.DatabaseTypeMysql:
        if database.Mysql == nil {
            return fmt.Errorf("database MySQL config is not set")
        }
    case databases.DatabaseTypeMariadb:
        if database.Mariadb == nil {
            return fmt.Errorf("database MariaDB config is not set")
        }
    case databases.DatabaseTypeMongodb:
        if database.Mongodb == nil {
            return fmt.Errorf("database MongoDB config is not set")
        }
    default:
        return fmt.Errorf("unsupported database type: %s", database.Type)
    }

    return nil
}
```

---

## Config Updates

### `backend/internal/config/config.go`

```go
type EnvVariables struct {
    // ... existing fields ...

    MongodbInstallDir string `env:"MONGODB_INSTALL_DIR"`

    // Testing MongoDB
    TestMongodb40Port string `env:"TEST_MONGODB_40_PORT"`
    TestMongodb42Port string `env:"TEST_MONGODB_42_PORT"`
    TestMongodb44Port string `env:"TEST_MONGODB_44_PORT"`
    TestMongodb50Port string `env:"TEST_MONGODB_50_PORT"`
    TestMongodb60Port string `env:"TEST_MONGODB_60_PORT"`
    TestMongodb70Port string `env:"TEST_MONGODB_70_PORT"`
    TestMongodb80Port string `env:"TEST_MONGODB_80_PORT"`
}

// In loadEnvVariables():
env.MongodbInstallDir = filepath.Join(backendRoot, "tools", "mongodb")
tools.VerifyMongodbInstallation(log, env.EnvMode, env.MongodbInstallDir)

if env.IsTesting {
    // ... existing checks ...
    if env.TestMongodb40Port == "" {
        log.Error("TEST_MONGODB_40_PORT is empty")
        os.Exit(1)
    }
    // ... similar for other MongoDB ports (4.2, 4.4, 5.0, 6.0, 7.0, 8.0)
}
```

---

## Docker Compose for Testing

### `backend/docker-compose.yml.example` additions

```yaml
services:
  # ... existing services ...

  test-mongodb-40:
    image: mongo:4.0
    container_name: test-mongodb-40
    ports:
      - "${TEST_MONGODB_40_PORT:-27040}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongo", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-42:
    image: mongo:4.2
    container_name: test-mongodb-42
    ports:
      - "${TEST_MONGODB_42_PORT:-27042}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongo", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-44:
    image: mongo:4.4
    container_name: test-mongodb-44
    ports:
      - "${TEST_MONGODB_44_PORT:-27044}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongo", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-50:
    image: mongo:5.0
    container_name: test-mongodb-50
    ports:
      - "${TEST_MONGODB_50_PORT:-27050}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-60:
    image: mongo:6.0
    container_name: test-mongodb-60
    ports:
      - "${TEST_MONGODB_60_PORT:-27060}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-70:
    image: mongo:7.0
    container_name: test-mongodb-70
    ports:
      - "${TEST_MONGODB_70_PORT:-27070}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10

  test-mongodb-80:
    image: mongo:8.0
    container_name: test-mongodb-80
    ports:
      - "${TEST_MONGODB_80_PORT:-27080}:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: rootpassword
      MONGO_INITDB_DATABASE: testdb
    command: mongod --auth
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 10
```

---

## Dockerfile Updates

Add to the runtime stage:

```dockerfile
# Install MongoDB Database Tools (single latest version - backward compatible)
# MongoDB Database Tools support all server versions (4.0-8.0)
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget gnupg && \
    # Download MongoDB Database Tools
    wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-100.10.0.deb && \
    dpkg -i mongodb-database-tools-debian12-x86_64-100.10.0.deb && \
    rm mongodb-database-tools-debian12-x86_64-100.10.0.deb && \
    # Create symlinks to standard location
    mkdir -p /usr/local/mongodb-database-tools/bin && \
    ln -sf /usr/bin/mongodump /usr/local/mongodb-database-tools/bin/mongodump && \
    ln -sf /usr/bin/mongorestore /usr/local/mongodb-database-tools/bin/mongorestore && \
    rm -rf /var/lib/apt/lists/*
```

---

## CI/CD Updates

### `.github/workflows/ci-release.yml` additions

```yaml
# In .env file creation:
# testing MongoDB
TEST_MONGODB_40_PORT=27040
TEST_MONGODB_42_PORT=27042
TEST_MONGODB_44_PORT=27044
TEST_MONGODB_50_PORT=27050
TEST_MONGODB_60_PORT=27060
TEST_MONGODB_70_PORT=27070
TEST_MONGODB_80_PORT=27080

# In "Wait for containers to be ready":
# Wait for MongoDB containers
echo "Waiting for MongoDB 4.0..."
timeout 120 bash -c 'until docker exec test-mongodb-40 mongo --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 4.2..."
timeout 120 bash -c 'until docker exec test-mongodb-42 mongo --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 4.4..."
timeout 120 bash -c 'until docker exec test-mongodb-44 mongo --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 5.0..."
timeout 120 bash -c 'until docker exec test-mongodb-50 mongosh --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 6.0..."
timeout 120 bash -c 'until docker exec test-mongodb-60 mongosh --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 7.0..."
timeout 120 bash -c 'until docker exec test-mongodb-70 mongosh --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'
echo "Waiting for MongoDB 8.0..."
timeout 120 bash -c 'until docker exec test-mongodb-80 mongosh --eval "db.adminCommand(\"ping\")" -u root -p rootpassword --authenticationDatabase admin 2>/dev/null; do sleep 2; done'

# Cache MongoDB Database Tools
- name: Cache MongoDB Database Tools
  id: cache-mongodb
  uses: actions/cache@v4
  with:
    path: backend/tools/mongodb
    key: mongodb-database-tools-100.10.0-v1
```

---

## Download Scripts Updates

### `backend/tools/download_linux.sh` additions

```bash
# ========== MongoDB Installation ==========
echo "========================================"
echo "Installing MongoDB Database Tools (single latest version)..."
echo "========================================"

MONGODB_DIR="$(pwd)/mongodb"
mkdir -p "$MONGODB_DIR/bin"

# MongoDB Database Tools are backward compatible - single version supports all servers
MONGODB_TOOLS_URL="https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-100.10.0.deb"

TEMP_DIR="/tmp/mongodb_install"
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

echo "Downloading MongoDB Database Tools..."
wget -q "$MONGODB_TOOLS_URL" -O mongodb-database-tools.deb

echo "Installing MongoDB Database Tools..."
$SUDO dpkg -i mongodb-database-tools.deb 2>/dev/null || $SUDO apt-get install -f -y

# Create symlinks to tools directory
ln -sf /usr/bin/mongodump "$MONGODB_DIR/bin/mongodump"
ln -sf /usr/bin/mongorestore "$MONGODB_DIR/bin/mongorestore"

cd - >/dev/null
rm -rf "$TEMP_DIR"

echo "MongoDB Database Tools installed successfully"
echo "  mongodump: $MONGODB_DIR/bin/mongodump"
echo "  mongorestore: $MONGODB_DIR/bin/mongorestore"
```

### `backend/tools/download_macos.sh` additions

```bash
# ========== MongoDB Installation ==========
echo "========================================"
echo "Installing MongoDB Database Tools..."
echo "========================================"

MONGODB_DIR="$(pwd)/mongodb"
mkdir -p "$MONGODB_DIR/bin"

# Install via Homebrew
echo "Installing MongoDB Database Tools via Homebrew..."
brew tap mongodb/brew 2>/dev/null || true
brew install mongodb-database-tools 2>/dev/null || {
    echo "Warning: Could not install mongodb-database-tools via Homebrew"
}

# Find Homebrew MongoDB tools path
BREW_MONGODB=""
if [ -f "/opt/homebrew/bin/mongodump" ]; then
    BREW_MONGODB="/opt/homebrew/bin"
elif [ -f "/usr/local/bin/mongodump" ]; then
    BREW_MONGODB="/usr/local/bin"
fi

if [ -n "$BREW_MONGODB" ]; then
    ln -sf "$BREW_MONGODB/mongodump" "$MONGODB_DIR/bin/mongodump"
    ln -sf "$BREW_MONGODB/mongorestore" "$MONGODB_DIR/bin/mongorestore"
    echo "MongoDB Database Tools linked from Homebrew"

    mongodump_ver=$("$MONGODB_DIR/bin/mongodump" --version 2>/dev/null | head -1)
    echo "  Verified: $mongodump_ver"
else
    echo "Warning: Could not find MongoDB Database Tools binaries"
    echo "Please install manually: brew tap mongodb/brew && brew install mongodb-database-tools"
fi
```

### `backend/tools/download_windows.bat` additions

```batch
:: ========== MongoDB Installation ==========
echo ========================================
echo Installing MongoDB Database Tools...
echo ========================================
echo.

:: MongoDB Database Tools download URL for Windows x64
set "MONGODB_TOOLS_URL=https://fastdl.mongodb.org/tools/db/mongodb-database-tools-windows-x86_64-100.10.0.zip"

set "mongodb_install_dir=%MONGODB_DIR%"

:: Check if already installed
if exist "!mongodb_install_dir!\bin\mongodump.exe" (
    echo MongoDB Database Tools already installed, skipping...
) else (
    set "mongodb_filename=mongodb-database-tools.zip"

    if not exist "!mongodb_filename!" (
        echo Downloading MongoDB Database Tools...
        curl -L -o "!mongodb_filename!" -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" "!MONGODB_TOOLS_URL!"
        if !errorlevel! neq 0 (
            echo ERROR: Download request failed
            goto :skip_mongodb
        )
    ) else (
        echo MongoDB Database Tools already downloaded
    )

    :: Extract MongoDB Database Tools
    echo Extracting MongoDB Database Tools...
    mkdir "!mongodb_install_dir!" 2>nul
    mkdir "!mongodb_install_dir!\bin" 2>nul

    powershell -Command "Expand-Archive -Path '!mongodb_filename!' -DestinationPath '!mongodb_install_dir!_temp' -Force"

    :: Move files from nested directory to install_dir
    for /d %%d in ("!mongodb_install_dir!_temp\mongodb-database-tools-*") do (
        if exist "%%d\bin\mongodump.exe" (
            copy "%%d\bin\mongodump.exe" "!mongodb_install_dir!\bin\" >nul 2>&1
            copy "%%d\bin\mongorestore.exe" "!mongodb_install_dir!\bin\" >nul 2>&1
        )
    )

    :: Cleanup temp directory
    rmdir /s /q "!mongodb_install_dir!_temp" 2>nul

    :: Verify installation
    if exist "!mongodb_install_dir!\bin\mongodump.exe" (
        echo MongoDB Database Tools installed successfully
    ) else (
        echo Failed to install MongoDB Database Tools - mongodump.exe not found
    )
)

:skip_mongodb
echo.
```

---

## Test Implementation

### `backend/internal/features/tests/mongodb_backup_restore_test.go`

```go
package tests

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/features/backups/backups"
    backups_config "postgresus-backend/internal/features/backups/config"
    "postgresus-backend/internal/features/databases"
    mongodbtypes "postgresus-backend/internal/features/databases/databases/mongodb"
    "postgresus-backend/internal/features/restores"
    restores_enums "postgresus-backend/internal/features/restores/enums"
    "postgresus-backend/internal/features/storages"
    users_enums "postgresus-backend/internal/features/users/enums"
    users_testing "postgresus-backend/internal/features/users/testing"
    workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
    test_utils "postgresus-backend/internal/util/testing"
    "postgresus-backend/internal/util/tools"
)

type MongodbContainer struct {
    Host         string
    Port         int
    Username     string
    Password     string
    Database     string
    AuthDatabase string
    Version      tools.MongodbVersion
    Client       *mongo.Client
}

type MongodbTestDataItem struct {
    ID        string    `bson:"_id"`
    Name      string    `bson:"name"`
    Value     int       `bson:"value"`
    CreatedAt time.Time `bson:"created_at"`
}

func Test_BackupAndRestoreMongodb_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MongodbVersion
        port    string
    }{
        {"MongoDB 4.0", tools.MongodbVersion40, env.TestMongodb40Port},
        {"MongoDB 4.2", tools.MongodbVersion42, env.TestMongodb42Port},
        {"MongoDB 4.4", tools.MongodbVersion44, env.TestMongodb44Port},
        {"MongoDB 5.0", tools.MongodbVersion50, env.TestMongodb50Port},
        {"MongoDB 6.0", tools.MongodbVersion60, env.TestMongodb60Port},
        {"MongoDB 7.0", tools.MongodbVersion70, env.TestMongodb70Port},
        {"MongoDB 8.0", tools.MongodbVersion80, env.TestMongodb80Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMongodbBackupRestoreForVersion(t, tc.version, tc.port)
        })
    }
}

func Test_BackupAndRestoreMongodbWithEncryption_RestoreIsSuccessful(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MongodbVersion
        port    string
    }{
        {"MongoDB 4.0", tools.MongodbVersion40, env.TestMongodb40Port},
        {"MongoDB 4.2", tools.MongodbVersion42, env.TestMongodb42Port},
        {"MongoDB 4.4", tools.MongodbVersion44, env.TestMongodb44Port},
        {"MongoDB 5.0", tools.MongodbVersion50, env.TestMongodb50Port},
        {"MongoDB 6.0", tools.MongodbVersion60, env.TestMongodb60Port},
        {"MongoDB 7.0", tools.MongodbVersion70, env.TestMongodb70Port},
        {"MongoDB 8.0", tools.MongodbVersion80, env.TestMongodb80Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            testMongodbBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
        })
    }
}

func testMongodbBackupRestoreForVersion(t *testing.T, mongodbVersion tools.MongodbVersion, port string) {
    container, err := connectToMongodbContainer(mongodbVersion, port)
    assert.NoError(t, err)
    defer func() { _ = container.Client.Disconnect(context.Background()) }()

    // Create test data
    ctx := context.Background()
    collection := container.Client.Database(container.Database).Collection("test_data")

    // Drop existing collection
    _ = collection.Drop(ctx)

    // Insert test data
    testDocs := []interface{}{
        MongodbTestDataItem{ID: "1", Name: "test1", Value: 100, CreatedAt: time.Now()},
        MongodbTestDataItem{ID: "2", Name: "test2", Value: 200, CreatedAt: time.Now()},
        MongodbTestDataItem{ID: "3", Name: "test3", Value: 300, CreatedAt: time.Now()},
    }
    _, err = collection.InsertMany(ctx, testDocs)
    assert.NoError(t, err)

    router := createMongodbTestRouter()
    user := users_testing.CreateTestUser(users_enums.UserRoleMember)
    workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)

    storage := storages.CreateTestStorage(workspace.ID)

    database := createMongodbDatabaseViaAPI(
        t, router, "Test MongoDB Database", workspace.ID,
        container.Host, container.Port,
        container.Username, container.Password,
        container.Database, container.AuthDatabase,
        mongodbVersion, user.Token,
    )

    enableBackupsViaAPI(
        t, router, database.ID, storage.ID,
        backups_config.BackupEncryptionNone, user.Token,
    )

    createBackupViaAPI(t, router, database.ID, user.Token)

    backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
    assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

    // Create new database for restore
    restoreDBName := "restoreddb_" + uuid.New().String()[:8]

    createMongodbRestoreViaAPI(
        t, router, backup.ID,
        container.Host, container.Port,
        container.Username, container.Password,
        restoreDBName, container.AuthDatabase,
        mongodbVersion, user.Token,
    )

    restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
    assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

    // Verify data integrity
    restoredCollection := container.Client.Database(restoreDBName).Collection("test_data")
    count, err := restoredCollection.CountDocuments(ctx, bson.M{})
    assert.NoError(t, err)
    assert.Equal(t, int64(3), count)

    // Cleanup
    err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
    if err != nil {
        t.Logf("Warning: Failed to delete backup file: %v", err)
    }

    // Drop restored database
    _ = container.Client.Database(restoreDBName).Drop(ctx)

    test_utils.MakeDeleteRequest(
        t, router,
        "/api/v1/databases/"+database.ID.String(),
        "Bearer "+user.Token,
        http.StatusNoContent,
    )
    storages.RemoveTestStorage(storage.ID)
    workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func testMongodbBackupRestoreWithEncryptionForVersion(t *testing.T, mongodbVersion tools.MongodbVersion, port string) {
    // Similar to testMongodbBackupRestoreForVersion but with encryption enabled
    // Implementation follows the same pattern as MySQL/MariaDB encrypted tests
}

func connectToMongodbContainer(version tools.MongodbVersion, port string) (*MongodbContainer, error) {
    if port == "" {
        return nil, fmt.Errorf("MongoDB %s port not configured", version)
    }

    dbName := "testdb"
    password := "rootpassword"
    username := "root"
    authDatabase := "admin"
    host := "127.0.0.1"

    portInt, err := strconv.Atoi(port)
    if err != nil {
        return nil, fmt.Errorf("failed to parse port: %w", err)
    }

    uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s",
        username, password, host, portInt, authDatabase)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    if err = client.Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    return &MongodbContainer{
        Host:         host,
        Port:         portInt,
        Username:     username,
        Password:     password,
        Database:     dbName,
        AuthDatabase: authDatabase,
        Version:      version,
        Client:       client,
    }, nil
}

func createMongodbDatabaseViaAPI(
    t *testing.T,
    router *gin.Engine,
    name string,
    workspaceID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    authDatabase string,
    version tools.MongodbVersion,
    token string,
) *databases.Database {
    request := databases.Database{
        Name:        name,
        WorkspaceID: &workspaceID,
        Type:        databases.DatabaseTypeMongodb,
        Mongodb: &mongodbtypes.MongodbDatabase{
            Host:         host,
            Port:         port,
            Username:     username,
            Password:     password,
            Database:     database,
            AuthDatabase: authDatabase,
            Version:      version,
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
        t.Fatalf("Failed to create MongoDB database. Status: %d, Body: %s", w.Code, w.Body.String())
    }

    var createdDatabase databases.Database
    if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
        t.Fatalf("Failed to unmarshal database response: %v", err)
    }

    return &createdDatabase
}

func createMongodbRestoreViaAPI(
    t *testing.T,
    router *gin.Engine,
    backupID uuid.UUID,
    host string,
    port int,
    username string,
    password string,
    database string,
    authDatabase string,
    version tools.MongodbVersion,
    token string,
) {
    request := restores.RestoreBackupRequest{
        MongodbDatabase: &mongodbtypes.MongodbDatabase{
            Host:         host,
            Port:         port,
            Username:     username,
            Password:     password,
            Database:     database,
            AuthDatabase: authDatabase,
            Version:      version,
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

func createMongodbTestRouter() *gin.Engine {
    // Same pattern as MySQL/MariaDB test router creation
    gin.SetMode(gin.TestMode)
    router := gin.New()
    // ... setup routes
    return router
}
```

---

## Controller Test Updates

### `backend/internal/features/databases/controller_test.go`

Add MongoDB test case to `Test_DatabaseSensitiveDataLifecycle_AllTypes`:

```go
func Test_DatabaseSensitiveDataLifecycle_AllTypes(t *testing.T) {
    testCases := []struct {
        name                string
        databaseType        DatabaseType
        createDatabase      func(workspaceID uuid.UUID) *Database
        updateDatabase      func(workspaceID uuid.UUID, databaseID uuid.UUID) *Database
        verifySensitiveData func(t *testing.T, database *Database)
        verifyHiddenData    func(t *testing.T, database *Database)
    }{
        // ... existing PostgreSQL and MariaDB cases ...

        {
            name:         "MongoDB Database",
            databaseType: DatabaseTypeMongodb,
            createDatabase: func(workspaceID uuid.UUID) *Database {
                return &Database{
                    WorkspaceID: &workspaceID,
                    Name:        "Test MongoDB Database",
                    Type:        DatabaseTypeMongodb,
                    Mongodb: &mongodb.MongodbDatabase{
                        Version:      tools.MongodbVersion70,
                        Host:         "localhost",
                        Port:         27017,
                        Username:     "root",
                        Password:     "original-password-secret",
                        Database:     "testdb",
                        AuthDatabase: "admin",
                    },
                }
            },
            updateDatabase: func(workspaceID uuid.UUID, databaseID uuid.UUID) *Database {
                return &Database{
                    ID:          databaseID,
                    WorkspaceID: &workspaceID,
                    Name:        "Updated MongoDB Database",
                    Type:        DatabaseTypeMongodb,
                    Mongodb: &mongodb.MongodbDatabase{
                        Version:      tools.MongodbVersion80,
                        Host:         "updated-host",
                        Port:         27018,
                        Username:     "updated_user",
                        Password:     "",
                        Database:     "updated_testdb",
                        AuthDatabase: "admin",
                    },
                }
            },
            verifySensitiveData: func(t *testing.T, database *Database) {
                assert.True(t, strings.HasPrefix(database.Mongodb.Password, "enc:"),
                    "Password should be encrypted in database")

                encryptor := encryption.GetFieldEncryptor()
                decrypted, err := encryptor.Decrypt(database.ID, database.Mongodb.Password)
                assert.NoError(t, err)
                assert.Equal(t, "original-password-secret", decrypted)
            },
            verifyHiddenData: func(t *testing.T, database *Database) {
                assert.Equal(t, "", database.Mongodb.Password)
            },
        },
    }

    // ... rest of the test function ...
}
```

---

## Read-Only User Tests

### `backend/internal/features/databases/databases/mongodb/readonly_user_test.go`

```go
package mongodb

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "strconv"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "postgresus-backend/internal/config"
    "postgresus-backend/internal/util/tools"
)

func Test_IsUserReadOnly_AdminUser_ReturnsFalse(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MongodbVersion
        port    string
    }{
        {"MongoDB 4.0", tools.MongodbVersion40, env.TestMongodb40Port},
        {"MongoDB 4.2", tools.MongodbVersion42, env.TestMongodb42Port},
        {"MongoDB 4.4", tools.MongodbVersion44, env.TestMongodb44Port},
        {"MongoDB 5.0", tools.MongodbVersion50, env.TestMongodb50Port},
        {"MongoDB 6.0", tools.MongodbVersion60, env.TestMongodb60Port},
        {"MongoDB 7.0", tools.MongodbVersion70, env.TestMongodb70Port},
        {"MongoDB 8.0", tools.MongodbVersion80, env.TestMongodb80Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            container := connectToMongodbContainer(t, tc.port, tc.version)
            defer func() { _ = container.Client.Disconnect(context.Background()) }()

            mongodbModel := createMongodbModel(container)
            logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
            ctx := context.Background()

            isReadOnly, err := mongodbModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
            assert.NoError(t, err)
            assert.False(t, isReadOnly, "Root user should not be read-only")
        })
    }
}

func Test_CreateReadOnlyUser_UserCanReadButNotWrite(t *testing.T) {
    env := config.GetEnv()
    cases := []struct {
        name    string
        version tools.MongodbVersion
        port    string
    }{
        {"MongoDB 4.0", tools.MongodbVersion40, env.TestMongodb40Port},
        {"MongoDB 4.2", tools.MongodbVersion42, env.TestMongodb42Port},
        {"MongoDB 4.4", tools.MongodbVersion44, env.TestMongodb44Port},
        {"MongoDB 5.0", tools.MongodbVersion50, env.TestMongodb50Port},
        {"MongoDB 6.0", tools.MongodbVersion60, env.TestMongodb60Port},
        {"MongoDB 7.0", tools.MongodbVersion70, env.TestMongodb70Port},
        {"MongoDB 8.0", tools.MongodbVersion80, env.TestMongodb80Port},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            container := connectToMongodbContainer(t, tc.port, tc.version)
            defer func() { _ = container.Client.Disconnect(context.Background()) }()

            // Create test collection and data
            ctx := context.Background()
            collection := container.Client.Database(container.Database).Collection("readonly_test")
            _ = collection.Drop(ctx)

            _, err := collection.InsertMany(ctx, []interface{}{
                bson.M{"data": "test1"},
                bson.M{"data": "test2"},
            })
            assert.NoError(t, err)

            mongodbModel := createMongodbModel(container)
            logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

            username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
            assert.NoError(t, err)
            assert.NotEmpty(t, username)
            assert.NotEmpty(t, password)
            assert.True(t, strings.HasPrefix(username, "postgresus-"))

            // Connect as read-only user
            readOnlyModel := &MongodbDatabase{
                Version:      mongodbModel.Version,
                Host:         mongodbModel.Host,
                Port:         mongodbModel.Port,
                Username:     username,
                Password:     password,
                Database:     mongodbModel.Database,
                AuthDatabase: mongodbModel.AuthDatabase,
            }

            isReadOnly, err := readOnlyModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
            assert.NoError(t, err)
            assert.True(t, isReadOnly, "Created user should be read-only")

            // Connect as read-only user to test operations
            readOnlyURI := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s",
                username, password, container.Host, container.Port, container.AuthDatabase)
            readOnlyClient, err := mongo.Connect(ctx, options.Client().ApplyURI(readOnlyURI))
            assert.NoError(t, err)
            defer func() { _ = readOnlyClient.Disconnect(ctx) }()

            readOnlyCollection := readOnlyClient.Database(container.Database).Collection("readonly_test")

            // Test read - should succeed
            count, err := readOnlyCollection.CountDocuments(ctx, bson.M{})
            assert.NoError(t, err)
            assert.Equal(t, int64(2), count)

            // Test write operations - should fail
            _, err = readOnlyCollection.InsertOne(ctx, bson.M{"data": "should-fail"})
            assert.Error(t, err)

            _, err = readOnlyCollection.UpdateOne(ctx, bson.M{"data": "test1"}, bson.M{"$set": bson.M{"data": "hacked"}})
            assert.Error(t, err)

            _, err = readOnlyCollection.DeleteOne(ctx, bson.M{"data": "test1"})
            assert.Error(t, err)

            // Cleanup - drop user
            adminDB := container.Client.Database("admin")
            _ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
        })
    }
}

func Test_ReadOnlyUser_CannotCreateOrDropCollections(t *testing.T) {
    env := config.GetEnv()
    container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion70)
    defer func() { _ = container.Client.Disconnect(context.Background()) }()

    ctx := context.Background()
    mongodbModel := createMongodbModel(container)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
    assert.NoError(t, err)

    // Connect as read-only user
    readOnlyURI := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s",
        username, password, container.Host, container.Port, container.AuthDatabase)
    readOnlyClient, err := mongo.Connect(ctx, options.Client().ApplyURI(readOnlyURI))
    assert.NoError(t, err)
    defer func() { _ = readOnlyClient.Disconnect(ctx) }()

    // Test create collection - should fail
    err = readOnlyClient.Database(container.Database).CreateCollection(ctx, "hack_collection")
    assert.Error(t, err)

    // Test drop collection - should fail
    err = readOnlyClient.Database(container.Database).Collection("readonly_test").Drop(ctx)
    assert.Error(t, err)

    // Cleanup
    adminDB := container.Client.Database("admin")
    _ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}

func Test_CreateReadOnlyUser_DatabaseNameWithSpecialChars_Success(t *testing.T) {
    env := config.GetEnv()
    container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion70)
    defer func() { _ = container.Client.Disconnect(context.Background()) }()

    ctx := context.Background()

    // Create database with special characters (dash)
    specialDbName := "test-db-with-dash"

    // Create a collection in the special database to ensure it exists
    specialCollection := container.Client.Database(specialDbName).Collection("test_collection")
    _ = specialCollection.Drop(ctx)
    _, err := specialCollection.InsertOne(ctx, bson.M{"data": "test"})
    assert.NoError(t, err)

    defer func() {
        _ = container.Client.Database(specialDbName).Drop(ctx)
    }()

    mongodbModel := &MongodbDatabase{
        Version:      tools.MongodbVersion70,
        Host:         container.Host,
        Port:         container.Port,
        Username:     container.Username,
        Password:     container.Password,
        Database:     specialDbName,
        AuthDatabase: container.AuthDatabase,
    }

    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
    assert.NoError(t, err)
    assert.NotEmpty(t, username)
    assert.NotEmpty(t, password)
    assert.True(t, strings.HasPrefix(username, "postgresus-"))

    // Verify read-only user can read from special database
    readOnlyURI := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s",
        username, password, container.Host, container.Port, container.AuthDatabase)
    readOnlyClient, err := mongo.Connect(ctx, options.Client().ApplyURI(readOnlyURI))
    assert.NoError(t, err)
    defer func() { _ = readOnlyClient.Disconnect(ctx) }()

    count, err := readOnlyClient.Database(specialDbName).Collection("test_collection").CountDocuments(ctx, bson.M{})
    assert.NoError(t, err)
    assert.Equal(t, int64(1), count)

    // Verify write fails
    _, err = readOnlyClient.Database(specialDbName).Collection("test_collection").InsertOne(ctx, bson.M{"data": "fail"})
    assert.Error(t, err)

    // Cleanup
    adminDB := container.Client.Database("admin")
    _ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}

type MongodbContainer struct {
    Host         string
    Port         int
    Username     string
    Password     string
    Database     string
    AuthDatabase string
    Version      tools.MongodbVersion
    Client       *mongo.Client
}

func connectToMongodbContainer(
    t *testing.T,
    port string,
    version tools.MongodbVersion,
) *MongodbContainer {
    if port == "" {
        t.Skipf("MongoDB port not configured for version %s", version)
    }

    dbName := "testdb"
    host := "127.0.0.1"
    username := "root"
    password := "rootpassword"
    authDatabase := "admin"

    portInt, err := strconv.Atoi(port)
    assert.NoError(t, err)

    uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s",
        username, password, host, portInt, authDatabase)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        t.Skipf("Failed to connect to MongoDB %s: %v", version, err)
    }

    if err = client.Ping(ctx, nil); err != nil {
        t.Skipf("Failed to ping MongoDB %s: %v", version, err)
    }

    return &MongodbContainer{
        Host:         host,
        Port:         portInt,
        Username:     username,
        Password:     password,
        Database:     dbName,
        AuthDatabase: authDatabase,
        Version:      version,
        Client:       client,
    }
}

func createMongodbModel(container *MongodbContainer) *MongodbDatabase {
    return &MongodbDatabase{
        Version:      container.Version,
        Host:         container.Host,
        Port:         container.Port,
        Username:     container.Username,
        Password:     container.Password,
        Database:     container.Database,
        AuthDatabase: container.AuthDatabase,
    }
}
```

---

## Implementation Order

1. **Phase 1: Core Infrastructure**
   - [ ] Create `MongodbVersion` enums and `mongodb.go` tools
   - [ ] Create `mongodb/model.go` with `MongodbDatabase` struct
   - [ ] Add migration for `mongodb_databases` table
   - [ ] Update `databases/model.go` to include MongoDB
   - [ ] Update `databases/enums.go` with `DatabaseTypeMongodb`
   - [ ] Update `databases/repository.go` to preload MongoDB

2. **Phase 2: Backup Implementation**
   - [ ] Create `usecases/mongodb/create_backup_uc.go`
   - [ ] Create `usecases/mongodb/di.go`
   - [ ] Update `usecases/create_backup_uc.go` to route to MongoDB
   - [ ] Update `usecases/di.go` to wire MongoDB use case

3. **Phase 3: Restore Implementation**
   - [ ] Create `restores/usecases/mongodb/restore_backup_uc.go`
   - [ ] Create `restores/usecases/mongodb/di.go`
   - [ ] Update `restores/usecases/restore_backup_uc.go` to route to MongoDB
   - [ ] Update `restores/dto.go` for MongoDB restore DTO
   - [ ] Update `restores/service.go` for MongoDB version validation

4. **Phase 4: Service Layer Updates**
   - [ ] Update `databases/service.go` - CopyDatabase, IsUserReadOnly, CreateReadOnlyUser
   - [ ] Update `config/config.go` with MongoDB configuration
   - [ ] Add MongoDB installation verification

5. **Phase 5: Healthcheck Updates**
   - [ ] Update `healthcheck/attempt/check_database_health_uc.go` - Add MongoDB case to `validateDatabase()`

6. **Phase 6: Infrastructure**
   - [ ] Update Dockerfile with MongoDB Database Tools installation
   - [ ] Update docker-compose.yml.example with MongoDB test containers
   - [ ] Update download_linux.sh with MongoDB tools
   - [ ] Update download_macos.sh with MongoDB tools
   - [ ] Update download_windows.bat with MongoDB tools
   - [ ] Update CI/CD workflow

7. **Phase 7: Testing**
   - [ ] Create `mongodb_backup_restore_test.go` - Integration tests for backup/restore
   - [ ] Create `mongodb/readonly_user_test.go` - Read-only user tests for all MongoDB versions
   - [ ] Update `databases/controller_test.go` - Add MongoDB case to `Test_DatabaseSensitiveDataLifecycle_AllTypes`
   - [ ] Test all MongoDB versions (4.0, 4.2, 4.4, 5.0, 6.0, 7.0, 8.0)
   - [ ] Test encryption with MongoDB backups
   - [ ] Test read-only user creation and permissions

---

## Notes and Considerations

### MongoDB vs MySQL/MariaDB Differences

| Feature              | MySQL/MariaDB               | MongoDB                          |
| -------------------- | --------------------------- | -------------------------------- |
| Backup tool          | `mysqldump`/`mariadb-dump`  | `mongodump`                      |
| Restore tool         | `mysql`/`mariadb`           | `mongorestore`                   |
| Output format        | SQL text + zstd             | Archive + gzip (built-in)        |
| Password handling    | `.my.cnf` temp file         | URI-based (in connection string) |
| Default port         | 3306                        | 27017                            |
| Auth database        | N/A                         | Usually "admin"                  |
| Compression          | External (zstd in pipeline) | Built-in (`--gzip` flag)         |
| Client compatibility | Version-specific (MySQL)    | Single version (backward compat) |

### Backup Pipeline

```
mongodump --archive --gzip → [encryption] → storage
```

Note: Unlike MySQL/MariaDB where we add zstd compression externally, MongoDB's `--gzip` flag provides built-in compression. No additional compression layer is needed.

### Restore Pipeline

```
storage → [decryption] → mongorestore --archive --gzip
```

### Version Detection

MongoDB's `buildInfo` command returns version strings like:

- `7.0.14`
- `8.0.3`
- `4.4.29`

The detection logic parses major.minor version to map to `MongodbVersion` enum.

### Client Backward Compatibility

MongoDB Database Tools are backward compatible:

- Latest `mongodump` (100.x) can backup MongoDB 4.0-8.0 servers
- Latest `mongorestore` (100.x) can restore to MongoDB 4.0-8.0 servers
- This simplifies installation - only one client version needed

### Security Considerations

1. **Password handling**: Passed via URI in connection string (standard MongoDB practice)
2. **TLS**: Support via `tls=true` parameter in URI
3. **Auth database**: Configurable (default: "admin")

### Cloud Database Compatibility

- **MongoDB Atlas**: Works with standard mongodump/mongorestore
- **AWS DocumentDB**: Compatible (with some limitations)
- **Azure Cosmos DB for MongoDB**: Compatible
- **Google Cloud MongoDB (via Atlas)**: Works with standard tools

### ARM64 Support

All MongoDB versions (4.0-8.0) have full ARM64 support for both server and client tools.

### Healthcheck Differences by Version

| MongoDB Version | Shell Command | Healthcheck                                |
| --------------- | ------------- | ------------------------------------------ |
| 4.0 - 4.4       | `mongo`       | `mongo --eval "db.adminCommand('ping')"`   |
| 5.0+            | `mongosh`     | `mongosh --eval "db.adminCommand('ping')"` |

The `mongo` shell was replaced by `mongosh` starting from MongoDB 5.0.

### Go Driver Dependency

The MongoDB model uses the official Go driver:

```go
import "go.mongodb.org/mongo-driver/mongo"
```

Add to `go.mod`:

```
go.mongodb.org/mongo-driver v1.17.1
```
