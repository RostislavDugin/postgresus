package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

type MongodbDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.MongodbVersion `json:"version" gorm:"type:text;not null"`

	Host               string `json:"host"               gorm:"type:text;not null"`
	Port               *int   `json:"port"               gorm:"type:int"`
	Username           string `json:"username"           gorm:"type:text;not null"`
	Password           string `json:"password"           gorm:"type:text;not null"`
	Database           string `json:"database"           gorm:"type:text;not null"`
	AuthDatabase       string `json:"authDatabase"       gorm:"type:text;not null;default:'admin'"`
	IsHttps            bool   `json:"isHttps"            gorm:"type:boolean;default:false"`
	IsSrv              bool   `json:"isSrv"              gorm:"column:is_srv;type:boolean;not null;default:false"`
	IsDirectConnection bool   `json:"isDirectConnection" gorm:"column:is_direct_connection;type:boolean;not null;default:false"`
	CpuCount           int    `json:"cpuCount"           gorm:"column:cpu_count;type:int;not null;default:1"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	ExcludeCollections       []string `json:"excludeCollections" gorm:"-"`
	ExcludeCollectionsString string   `json:"-"                  gorm:"column:exclude_collections;type:text;not null;default:''"`
}

func (m *MongodbDatabase) TableName() string {
	return "mongodb_databases"
}

func (m *MongodbDatabase) BeforeSave(_ *gorm.DB) error {
	m.ExcludeCollectionsString = namelist.FormatUniqueNames(m.ExcludeCollections)

	return nil
}

func (m *MongodbDatabase) AfterFind(_ *gorm.DB) error {
	m.ExcludeCollections = namelist.ParseUniqueNames(m.ExcludeCollectionsString)

	return nil
}

func (m *MongodbDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}

	if !m.IsSrv {
		if m.Port == nil || *m.Port == 0 {
			return errors.New("port is required for standard connections")
		}
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
	if m.CpuCount <= 0 {
		return errors.New("cpu count must be greater than 0")
	}

	// SRV resolves its own host list from DNS and never carries a port, so there is nothing for a
	// local forward to stand in front of.
	if m.SshTunnel.IsEnabled && m.IsSrv {
		return errors.New(
			"SSH tunnel cannot be used with an SRV connection: " +
				"SRV resolves its own host list and bypasses the forwarded port",
		)
	}

	return m.SshTunnel.Validate()
}

func (m *MongodbDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.BuildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		// The URI carries the password, so only the database name goes into the log.
		logger.ErrorContext(ctx, "failed to open the mongodb connection", "database_name", m.Database, "error", err)

		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.ErrorContext(ctx, "failed to disconnect from mongodb", "error", disconnectErr)
		}
	}()

	if err := client.Ping(ctx, nil); err != nil {
		logger.ErrorContext(ctx, "failed to ping the mongodb database", "database_name", m.Database, "error", err)

		return fmt.Errorf("failed to ping MongoDB database '%s': %w", m.Database, err)
	}

	detectedVersion, err := detectMongodbVersion(ctx, client)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	if err := checkBackupPermissions(
		ctx,
		client,
		m.Username,
		m.Database,
		m.AuthDatabase,
	); err != nil {
		return err
	}

	return nil
}

// GetRawDbSizeMb reads dataSize from the dbStats command — the uncompressed
// total of user data, the closest analog to the relational "raw DB size".
func (m *MongodbDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if m.Database == "" {
		return 0, nil
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.BuildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.ErrorContext(ctx, "failed to disconnect from MongoDB", "error", disconnectErr)
		}
	}()

	result := client.Database(m.Database).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}})

	var stats struct {
		DataSize float64 `bson:"dataSize"`
	}

	if err := result.Decode(&stats); err != nil {
		return 0, fmt.Errorf("failed to decode dbStats for MongoDB database '%s': %w", m.Database, err)
	}

	return stats.DataSize / (1024 * 1024), nil
}

func (m *MongodbDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
	m.SshTunnel.HideSensitiveData()
}

func (m *MongodbDatabase) Update(incoming *MongodbDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.AuthDatabase = incoming.AuthDatabase
	m.IsHttps = incoming.IsHttps
	m.IsSrv = incoming.IsSrv
	m.IsDirectConnection = incoming.IsDirectConnection
	m.CpuCount = incoming.CpuCount
	m.ExcludeCollections = incoming.ExcludeCollections
	m.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MongodbDatabase) CopyForNewDatabase() *MongodbDatabase {
	if m == nil {
		return nil
	}

	copiedDatabase := *m
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.ExcludeCollections = slices.Clone(m.ExcludeCollections)

	if m.Port != nil {
		copiedDatabase.Port = new(*m.Port)
	}

	return &copiedDatabase
}

func (m *MongodbDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}

	return m.SshTunnel.EncryptSensitiveFields(encryptor)
}

func (m *MongodbDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return m.PopulateVersion(logger, encryptor)
}

func (m *MongodbDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.BuildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("failed to disconnect", "error", disconnectErr)
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
) (bool, []string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return false, nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.BuildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.ErrorContext(ctx, "failed to disconnect", "error", disconnectErr)
		}
	}()

	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	adminDB := client.Database(authDB)
	var result bson.M
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: m.Username},
			{Key: "db", Value: authDB},
		}},
	}).Decode(&result)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	writeRoles := map[string]bool{
		"readWrite":            true,
		"readWriteAnyDatabase": true,
		"dbAdmin":              true,
		"dbAdminAnyDatabase":   true,
		"userAdmin":            true,
		"userAdminAnyDatabase": true,
		"clusterAdmin":         true,
		"clusterManager":       true,
		"hostManager":          true,
		"root":                 true,
		"dbOwner":              true,
		"restore":              true,
		"__system":             true,
	}

	// Roles that are read-only for our backup purposes
	// The "backup" role has insert/update on mms.backup collection but is needed for mongodump
	readOnlyRoles := map[string]bool{
		"read":   true,
		"backup": true,
	}

	writeActions := map[string]bool{
		"insert":             true,
		"update":             true,
		"remove":             true,
		"createCollection":   true,
		"dropCollection":     true,
		"createIndex":        true,
		"dropIndex":          true,
		"convertToCapped":    true,
		"dropDatabase":       true,
		"renameCollection":   true,
		"createUser":         true,
		"dropUser":           true,
		"updateUser":         true,
		"grantRole":          true,
		"revokeRole":         true,
		"dropRole":           true,
		"createRole":         true,
		"updateRole":         true,
		"enableSharding":     true,
		"shardCollection":    true,
		"addShard":           true,
		"removeShard":        true,
		"shutdown":           true,
		"replSetReconfig":    true,
		"replSetStateChange": true,
	}

	var detectedRoles []string

	users, ok := result["users"].(bson.A)
	if !ok || len(users) == 0 {
		return true, detectedRoles, nil
	}

	user, ok := users[0].(bson.M)
	if !ok {
		return true, detectedRoles, nil
	}

	roles, ok := user["roles"].(bson.A)
	if !ok {
		return true, detectedRoles, nil
	}

	// Collect all role names and check for write roles
	for _, roleDoc := range roles {
		role, ok := roleDoc.(bson.M)
		if !ok {
			continue
		}
		roleName, _ := role["role"].(string)
		if roleName != "" {
			detectedRoles = append(detectedRoles, roleName)
		}
	}

	// Check if any detected role is a write role
	for _, roleName := range detectedRoles {
		if writeRoles[roleName] {
			return false, detectedRoles, nil
		}
	}

	// If all roles are known read-only roles (read, backup), skip inherited privilege check
	allRolesReadOnly := true
	for _, roleName := range detectedRoles {
		if !readOnlyRoles[roleName] {
			allRolesReadOnly = false
			break
		}
	}
	if allRolesReadOnly && len(detectedRoles) > 0 {
		return true, detectedRoles, nil
	}

	// Check inherited privileges for custom roles
	var privResult bson.M
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: m.Username},
			{Key: "db", Value: authDB},
		}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&privResult)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user privileges: %w", err)
	}

	privUsers, ok := privResult["users"].(bson.A)
	if !ok || len(privUsers) == 0 {
		return true, detectedRoles, nil
	}

	privUser, ok := privUsers[0].(bson.M)
	if !ok {
		return true, detectedRoles, nil
	}

	// Check inheritedPrivileges for write actions
	inheritedPrivileges, ok := privUser["inheritedPrivileges"].(bson.A)
	if ok {
		for _, privDoc := range inheritedPrivileges {
			priv, ok := privDoc.(bson.M)
			if !ok {
				continue
			}
			actions, ok := priv["actions"].(bson.A)
			if !ok {
				continue
			}
			for _, action := range actions {
				actionStr, ok := action.(string)
				if ok && writeActions[actionStr] {
					return false, detectedRoles, nil
				}
			}
		}
	}

	return true, detectedRoles, nil
}

func (m *MongodbDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.BuildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.ErrorContext(ctx, "failed to disconnect", "error", disconnectErr)
		}
	}()

	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	maxRetries := 3
	for attempt := range maxRetries {
		newUsername := fmt.Sprintf("databasus-%s", uuid.New().String()[:8])
		newPassword := encryption.GenerateComplexPassword()

		adminDB := client.Database(authDB)
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

		logger.InfoContext(
			ctx,
			"read-only MongoDB user created successfully",
			"username", newUsername,
		)
		return newUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

// BuildConnectionURI builds a MongoDB connection URI with the database embedded
// in the path. Used by the Go driver and the bundled mongodump invocation —
// embedding the database in the URI lets us drop `--db`, which stricter
// mongodump versions reject when combined with `--uri`.
func (m *MongodbDatabase) BuildConnectionURI(password string) string {
	return m.buildURI(password, true)
}

// BuildRestoreURI builds a connection URI without the database in the path.
// mongorestore treats a URI-embedded database as an implicit target rewrite
// that collides with --nsFrom / --nsTo, silently producing a zero-document
// restore. Restore must use a database-less URI and rely on the namespace
// flags exclusively.
func (m *MongodbDatabase) BuildRestoreURI(password string) string {
	return m.buildURI(password, false)
}

func (m *MongodbDatabase) buildURI(password string, includeDatabase bool) string {
	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	extraParams := ""
	if m.IsHttps {
		extraParams += "&tls=true&tlsInsecure=true"
	} else {
		extraParams += "&tls=false"
	}
	if m.IsDirectConnection {
		extraParams += "&directConnection=true"
	}

	dbPath := ""
	if includeDatabase {
		dbPath = url.PathEscape(m.Database)
	}

	if m.IsSrv {
		return fmt.Sprintf(
			"mongodb+srv://%s:%s@%s/%s?authSource=%s&connectTimeoutMS=15000%s",
			url.QueryEscape(m.Username),
			url.QueryEscape(password),
			m.Host,
			dbPath,
			url.QueryEscape(authDB),
			extraParams,
		)
	}

	port := 27017
	if m.Port != nil {
		port = *m.Port
	}

	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&connectTimeoutMS=15000%s",
		url.QueryEscape(m.Username),
		url.QueryEscape(password),
		m.Host,
		port,
		dbPath,
		url.QueryEscape(authDB),
		extraParams,
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

	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MongoDB version: %s", versionStr)
	}

	return mapMongodbVersion(matches[1], matches[2])
}

// mapMongodbVersion validates a parsed major.minor against the supported
// matrix. The bundled MongoDB Database Tools (100.16.1) drop wire-protocol
// v7, so the minimum supported server version is 4.2 — matching
// assets/tools/README.md.
func mapMongodbVersion(major, minor string) (tools.MongodbVersion, error) {
	switch major {
	case "4":
		if minor == "0" || minor == "1" {
			return "", fmt.Errorf(
				"unsupported MongoDB version: %s.%s (minimum supported: 4.2)",
				major, minor,
			)
		}
		return tools.MongodbVersion4, nil
	case "5":
		return tools.MongodbVersion5, nil
	case "6":
		return tools.MongodbVersion6, nil
	case "7":
		return tools.MongodbVersion7, nil
	case "8":
		return tools.MongodbVersion8, nil
	default:
		return "", fmt.Errorf(
			"unsupported MongoDB version: %s.%s (supported: 4.2+, 5.x, 6.x, 7.x, 8.x)",
			major, minor,
		)
	}
}

// checkBackupPermissions verifies the user has sufficient privileges for mongodump backup.
// Required: 'read' role on target database OR 'backup' role on admin OR 'readAnyDatabase' role.
func checkBackupPermissions(
	ctx context.Context,
	client *mongo.Client,
	username, database, authDatabase string,
) error {
	authDB := authDatabase
	if authDB == "" {
		authDB = "admin"
	}

	adminDB := client.Database(authDB)
	var result bson.M
	err := adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: username},
			{Key: "db", Value: authDB},
		}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	users, ok := result["users"].(bson.A)
	if !ok || len(users) == 0 {
		return errors.New("insufficient permissions for backup. User not found")
	}

	user, ok := users[0].(bson.M)
	if !ok {
		return errors.New("insufficient permissions for backup. Could not parse user info")
	}

	// Check roles for backup permissions
	roles, ok := user["roles"].(bson.A)
	if !ok {
		return errors.New("insufficient permissions for backup. No roles assigned")
	}

	backupRoles := map[string]bool{
		"backup":               true,
		"root":                 true,
		"readAnyDatabase":      true,
		"dbOwner":              true,
		"__system":             true,
		"clusterAdmin":         true,
		"readWriteAnyDatabase": true,
	}

	var userRoles []string
	hasBackupRole := false
	hasReadOnTargetDB := false

	for _, roleDoc := range roles {
		role, ok := roleDoc.(bson.M)
		if !ok {
			continue
		}
		roleName, _ := role["role"].(string)
		roleDB, _ := role["db"].(string)

		if roleName != "" {
			userRoles = append(userRoles, roleName)
		}

		if backupRoles[roleName] {
			hasBackupRole = true
		}

		if roleName == "read" && (roleDB == database || roleDB == "") {
			hasReadOnTargetDB = true
		}
		if roleName == "readWrite" && (roleDB == database || roleDB == "") {
			hasReadOnTargetDB = true
		}
	}

	if hasBackupRole || hasReadOnTargetDB {
		return nil
	}

	// Check inherited privileges for 'find' action on target database
	inheritedPrivileges, ok := user["inheritedPrivileges"].(bson.A)
	if ok {
		for _, privDoc := range inheritedPrivileges {
			priv, ok := privDoc.(bson.M)
			if !ok {
				continue
			}
			resource, ok := priv["resource"].(bson.M)
			if !ok {
				continue
			}

			resourceDB, _ := resource["db"].(string)
			resourceCluster, _ := resource["cluster"].(bool)

			isTargetDB := resourceDB == database || resourceDB == "" || resourceCluster

			actions, ok := priv["actions"].(bson.A)
			if !ok {
				continue
			}

			for _, action := range actions {
				actionStr, ok := action.(string)
				if ok && actionStr == "find" && isTargetDB {
					return nil
				}
			}
		}
	}

	return fmt.Errorf(
		"insufficient permissions for backup. Current roles: %s. Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
		strings.Join(userRoles, ", "),
		database,
	)
}

func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}
	return encryptor.Decrypt(password)
}
