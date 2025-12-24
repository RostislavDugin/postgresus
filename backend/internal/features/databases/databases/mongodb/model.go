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
	IsHttps      bool   `json:"isHttps"      gorm:"type:boolean;default:false"`
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
	m.IsHttps = incoming.IsHttps

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
	return m.PopulateVersion(logger, encryptor, databaseID)
}

func (m *MongodbDatabase) PopulateVersion(
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
		return false, fmt.Errorf("failed to get user info: %w", err)
	}

	writeRoles := []string{
		"readWrite", "readWriteAnyDatabase", "dbAdmin", "dbAdminAnyDatabase",
		"userAdmin", "userAdminAnyDatabase", "clusterAdmin", "root",
		"dbOwner", "backup", "restore",
	}

	users, ok := result["users"].(bson.A)
	if !ok || len(users) == 0 {
		return true, nil
	}

	user, ok := users[0].(bson.M)
	if !ok {
		return true, nil
	}

	roles, ok := user["roles"].(bson.A)
	if !ok {
		return true, nil
	}

	for _, roleDoc := range roles {
		role, ok := roleDoc.(bson.M)
		if !ok {
			continue
		}
		roleName, _ := role["role"].(string)
		for _, writeRole := range writeRoles {
			if roleName == writeRole {
				return false, nil
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

	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	maxRetries := 3
	for attempt := range maxRetries {
		newUsername := fmt.Sprintf("postgresus-%s", uuid.New().String()[:8])
		newPassword := uuid.New().String()

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
	if m.IsHttps {
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
	if m.IsHttps {
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

	re := regexp.MustCompile(`^(\d+)\.`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse MongoDB version: %s", versionStr)
	}

	major := matches[1]

	switch major {
	case "4":
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
			"unsupported MongoDB major version: %s (supported: 4.x, 5.x, 6.x, 7.x, 8.x)",
			major,
		)
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
