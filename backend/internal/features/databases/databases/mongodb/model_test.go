package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
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

	"databasus-backend/internal/config"
	"databasus-backend/internal/util/tools"
)

func Test_TestConnection_InsufficientPermissions_ReturnsError(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MongodbVersion
		port    string
	}{
		{"MongoDB 4.0", tools.MongodbVersion4, env.TestMongodb40Port},
		{"MongoDB 4.2", tools.MongodbVersion4, env.TestMongodb42Port},
		{"MongoDB 4.4", tools.MongodbVersion4, env.TestMongodb44Port},
		{"MongoDB 5.0", tools.MongodbVersion5, env.TestMongodb50Port},
		{"MongoDB 6.0", tools.MongodbVersion6, env.TestMongodb60Port},
		{"MongoDB 7.0", tools.MongodbVersion7, env.TestMongodb70Port},
		{"MongoDB 8.2", tools.MongodbVersion8, env.TestMongodb82Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMongodbContainer(t, tc.port, tc.version)
			defer container.Client.Disconnect(context.Background())

			ctx := context.Background()
			db := container.Client.Database(container.Database)

			_ = db.Collection("permission_test").Drop(ctx)
			_, err := db.Collection("permission_test").InsertOne(ctx, bson.M{"data": "test1"})
			assert.NoError(t, err)

			limitedUsername := fmt.Sprintf("limited_%s", uuid.New().String()[:8])
			limitedPassword := "limitedpassword123"

			adminDB := container.Client.Database(container.AuthDatabase)
			err = adminDB.RunCommand(ctx, bson.D{
				{Key: "createUser", Value: limitedUsername},
				{Key: "pwd", Value: limitedPassword},
				{Key: "roles", Value: bson.A{}},
			}).Err()
			assert.NoError(t, err)

			defer dropUserSafe(container.Client, limitedUsername, container.AuthDatabase)

			port := container.Port
			mongodbModel := &MongodbDatabase{
				Version:      tc.version,
				Host:         container.Host,
				Port:         &port,
				Username:     limitedUsername,
				Password:     limitedPassword,
				Database:     container.Database,
				AuthDatabase: container.AuthDatabase,
				IsHttps:      false,
				IsSrv:        false,
				CpuCount:     1,
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			err = mongodbModel.TestConnection(logger, nil, uuid.New())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient permissions")
		})
	}
}

func Test_TestConnection_SufficientPermissions_Success(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MongodbVersion
		port    string
	}{
		{"MongoDB 4.0", tools.MongodbVersion4, env.TestMongodb40Port},
		{"MongoDB 4.2", tools.MongodbVersion4, env.TestMongodb42Port},
		{"MongoDB 4.4", tools.MongodbVersion4, env.TestMongodb44Port},
		{"MongoDB 5.0", tools.MongodbVersion5, env.TestMongodb50Port},
		{"MongoDB 6.0", tools.MongodbVersion6, env.TestMongodb60Port},
		{"MongoDB 7.0", tools.MongodbVersion7, env.TestMongodb70Port},
		{"MongoDB 8.2", tools.MongodbVersion8, env.TestMongodb82Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMongodbContainer(t, tc.port, tc.version)
			defer container.Client.Disconnect(context.Background())

			ctx := context.Background()
			db := container.Client.Database(container.Database)

			_ = db.Collection("backup_test").Drop(ctx)
			_, err := db.Collection("backup_test").InsertOne(ctx, bson.M{"data": "test1"})
			assert.NoError(t, err)

			backupUsername := fmt.Sprintf("backup_%s", uuid.New().String()[:8])
			backupPassword := "backuppassword123"

			adminDB := container.Client.Database(container.AuthDatabase)
			err = adminDB.RunCommand(ctx, bson.D{
				{Key: "createUser", Value: backupUsername},
				{Key: "pwd", Value: backupPassword},
				{Key: "roles", Value: bson.A{
					bson.D{
						{Key: "role", Value: "read"},
						{Key: "db", Value: container.Database},
					},
				}},
			}).Err()
			assert.NoError(t, err)

			defer dropUserSafe(container.Client, backupUsername, container.AuthDatabase)

			port := container.Port
			mongodbModel := &MongodbDatabase{
				Version:      tc.version,
				Host:         container.Host,
				Port:         &port,
				Username:     backupUsername,
				Password:     backupPassword,
				Database:     container.Database,
				AuthDatabase: container.AuthDatabase,
				IsHttps:      false,
				IsSrv:        false,
				CpuCount:     1,
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			err = mongodbModel.TestConnection(logger, nil, uuid.New())
			assert.NoError(t, err)
		})
	}
}

func Test_IsUserReadOnly_AdminUser_ReturnsFalse(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MongodbVersion
		port    string
	}{
		{"MongoDB 4.0", tools.MongodbVersion4, env.TestMongodb40Port},
		{"MongoDB 4.2", tools.MongodbVersion4, env.TestMongodb42Port},
		{"MongoDB 4.4", tools.MongodbVersion4, env.TestMongodb44Port},
		{"MongoDB 5.0", tools.MongodbVersion5, env.TestMongodb50Port},
		{"MongoDB 6.0", tools.MongodbVersion6, env.TestMongodb60Port},
		{"MongoDB 7.0", tools.MongodbVersion7, env.TestMongodb70Port},
		{"MongoDB 8.2", tools.MongodbVersion8, env.TestMongodb82Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMongodbContainer(t, tc.port, tc.version)
			defer container.Client.Disconnect(context.Background())

			mongodbModel := createMongodbModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			isReadOnly, roles, err := mongodbModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.False(t, isReadOnly, "Root user should not be read-only")
			assert.NotEmpty(t, roles, "Root user should have roles")
		})
	}
}

func Test_IsUserReadOnly_ReadOnlyUser_ReturnsTrue(t *testing.T) {
	env := config.GetEnv()
	container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion7)
	defer container.Client.Disconnect(context.Background())

	ctx := context.Background()
	db := container.Client.Database(container.Database)

	_ = db.Collection("readonly_check_test").Drop(ctx)
	_, err := db.Collection("readonly_check_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	readOnlyModel := &MongodbDatabase{
		Version:      mongodbModel.Version,
		Host:         mongodbModel.Host,
		Port:         mongodbModel.Port,
		Username:     username,
		Password:     password,
		Database:     mongodbModel.Database,
		AuthDatabase: mongodbModel.AuthDatabase,
		IsHttps:      false,
		CpuCount:     1,
	}

	isReadOnly, roles, err := readOnlyModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)
	assert.True(t, isReadOnly, "Read-only user should be read-only")
	assert.NotEmpty(t, roles, "Read-only user should have roles (read, backup)")

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

func Test_CreateReadOnlyUser_UserCanReadButNotWrite(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MongodbVersion
		port    string
	}{
		{"MongoDB 4.0", tools.MongodbVersion4, env.TestMongodb40Port},
		{"MongoDB 4.2", tools.MongodbVersion4, env.TestMongodb42Port},
		{"MongoDB 4.4", tools.MongodbVersion4, env.TestMongodb44Port},
		{"MongoDB 5.0", tools.MongodbVersion5, env.TestMongodb50Port},
		{"MongoDB 6.0", tools.MongodbVersion6, env.TestMongodb60Port},
		{"MongoDB 7.0", tools.MongodbVersion7, env.TestMongodb70Port},
		{"MongoDB 8.2", tools.MongodbVersion8, env.TestMongodb82Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMongodbContainer(t, tc.port, tc.version)
			defer container.Client.Disconnect(context.Background())

			ctx := context.Background()
			db := container.Client.Database(container.Database)

			_ = db.Collection("readonly_test").Drop(ctx)
			_ = db.Collection("hack_collection").Drop(ctx)

			_, err := db.Collection("readonly_test").InsertMany(ctx, []interface{}{
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
			assert.True(t, strings.HasPrefix(username, "databasus-"))

			if err != nil {
				return
			}

			readOnlyClient := connectWithCredentials(t, container, username, password)
			defer readOnlyClient.Disconnect(ctx)

			readOnlyDB := readOnlyClient.Database(container.Database)

			var count int64
			count, err = readOnlyDB.Collection("readonly_test").CountDocuments(ctx, bson.M{})
			assert.NoError(t, err)
			assert.Equal(t, int64(2), count)

			_, err = readOnlyDB.Collection("readonly_test").
				InsertOne(ctx, bson.M{"data": "should-fail"})
			assert.Error(t, err)
			assertWriteDenied(t, err)

			_, err = readOnlyDB.Collection("readonly_test").UpdateOne(
				ctx,
				bson.M{"data": "test1"},
				bson.M{"$set": bson.M{"data": "hacked"}},
			)
			assert.Error(t, err)
			assertWriteDenied(t, err)

			_, err = readOnlyDB.Collection("readonly_test").DeleteOne(ctx, bson.M{"data": "test1"})
			assert.Error(t, err)
			assertWriteDenied(t, err)

			err = readOnlyDB.CreateCollection(ctx, "hack_collection")
			assert.Error(t, err)
			assertWriteDenied(t, err)

			dropUserSafe(container.Client, username, container.AuthDatabase)
		})
	}
}

func Test_ReadOnlyUser_FutureCollections_CanSelect(t *testing.T) {
	env := config.GetEnv()
	container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion7)
	defer container.Client.Disconnect(context.Background())

	ctx := context.Background()
	db := container.Client.Database(container.Database)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	_ = db.Collection("future_collection").Drop(ctx)
	_, err = db.Collection("future_collection").InsertOne(ctx, bson.M{"data": "future_data"})
	assert.NoError(t, err)

	readOnlyClient := connectWithCredentials(t, container, username, password)
	defer readOnlyClient.Disconnect(ctx)

	readOnlyDB := readOnlyClient.Database(container.Database)

	var result bson.M
	err = readOnlyDB.Collection("future_collection").FindOne(ctx, bson.M{}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "future_data", result["data"])

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

func Test_ReadOnlyUser_CannotDropOrModifyCollections(t *testing.T) {
	env := config.GetEnv()
	container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion7)
	defer container.Client.Disconnect(context.Background())

	ctx := context.Background()
	db := container.Client.Database(container.Database)

	_ = db.Collection("drop_test").Drop(ctx)
	_, err := db.Collection("drop_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	readOnlyClient := connectWithCredentials(t, container, username, password)
	defer readOnlyClient.Disconnect(ctx)

	readOnlyDB := readOnlyClient.Database(container.Database)

	err = readOnlyDB.Collection("drop_test").Drop(ctx)
	assert.Error(t, err)
	assertWriteDenied(t, err)

	_, err = readOnlyDB.Collection("drop_test").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "data", Value: 1}},
	})
	assert.Error(t, err)
	assertWriteDenied(t, err)

	dropUserSafe(container.Client, username, container.AuthDatabase)
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
	host := config.GetEnv().TestLocalhost
	username := "root"
	password := "rootpassword"
	authDatabase := "admin"

	portInt, err := strconv.Atoi(port)
	assert.NoError(t, err)

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&serverSelectionTimeoutMS=5000&connectTimeoutMS=5000",
		username,
		password,
		host,
		portInt,
		dbName,
		authDatabase,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Skipf("Failed to connect to MongoDB %s: %v", version, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
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
	port := container.Port
	return &MongodbDatabase{
		Version:      container.Version,
		Host:         container.Host,
		Port:         &port,
		Username:     container.Username,
		Password:     container.Password,
		Database:     container.Database,
		AuthDatabase: container.AuthDatabase,
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}
}

func connectWithCredentials(
	t *testing.T,
	container *MongodbContainer,
	username, password string,
) *mongo.Client {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		url.QueryEscape(username), url.QueryEscape(password),
		container.Host, container.Port,
		container.Database, container.AuthDatabase,
	)

	ctx := context.Background()
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	assert.NoError(t, err)

	return client
}

func dropUserSafe(client *mongo.Client, username, authDatabase string) {
	ctx := context.Background()
	adminDB := client.Database(authDatabase)
	_ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}

func assertWriteDenied(t *testing.T, err error) {
	errStr := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errStr, "not authorized") ||
			strings.Contains(errStr, "unauthorized") ||
			strings.Contains(errStr, "permission denied"),
		"Expected authorization error, got: %v", err)
}

func Test_BuildConnectionURI_WithSrvFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb+srv://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "cluster0.example.mongodb.net")
	assert.Contains(t, uri, "/mydb")
	assert.Contains(t, uri, "authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, ":27017")
}

func Test_BuildConnectionURI_WithStandardFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "localhost:27017")
	assert.Contains(t, uri, "/mydb")
	assert.Contains(t, uri, "authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, "mongodb+srv://")
}

func Test_BuildConnectionURI_WithNullPort_UsesDefault(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.Contains(t, uri, "localhost:27017")
}

func Test_BuildMongodumpURI_WithSrvFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
	}

	uri := model.BuildMongodumpURI("testpass123")

	assert.Contains(t, uri, "mongodb+srv://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "cluster0.example.mongodb.net")
	assert.Contains(t, uri, "/?authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, ":27017")
	assert.NotContains(t, uri, "/mydb")
}

func Test_BuildMongodumpURI_WithStandardFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildMongodumpURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "localhost:27017")
	assert.Contains(t, uri, "/?authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, "mongodb+srv://")
	assert.NotContains(t, uri, "/mydb")
}

func Test_Validate_SrvConnection_AllowsNullPort(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
		CpuCount:     1,
	}

	err := model.Validate()

	assert.NoError(t, err)
}

func Test_BuildConnectionURI_WithDirectConnection_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:             "mongo.example.local",
		Port:             &port,
		Username:         "testuser",
		Password:         "testpass123",
		Database:         "mydb",
		AuthDatabase:     "admin",
		IsHttps:          false,
		IsSrv:            false,
		DirectConnection: true,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "directConnection=true")
	assert.Contains(t, uri, "mongo.example.local:27017")
	assert.Contains(t, uri, "authSource=admin")
}

func Test_BuildConnectionURI_WithoutDirectConnection_OmitsParam(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:             "localhost",
		Port:             &port,
		Username:         "testuser",
		Password:         "testpass123",
		Database:         "mydb",
		AuthDatabase:     "admin",
		IsHttps:          false,
		IsSrv:            false,
		DirectConnection: false,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.NotContains(t, uri, "directConnection")
}

func Test_BuildMongodumpURI_WithDirectConnection_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:             "mongo.example.local",
		Port:             &port,
		Username:         "testuser",
		Password:         "testpass123",
		Database:         "mydb",
		AuthDatabase:     "admin",
		IsHttps:          false,
		IsSrv:            false,
		DirectConnection: true,
	}

	uri := model.BuildMongodumpURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "directConnection=true")
	assert.NotContains(t, uri, "/mydb")
}

func Test_BuildConnectionURI_WithDirectConnectionAndTls_ReturnsBothParams(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:             "mongo.example.local",
		Port:             &port,
		Username:         "testuser",
		Password:         "testpass123",
		Database:         "mydb",
		AuthDatabase:     "admin",
		IsHttps:          true,
		IsSrv:            false,
		DirectConnection: true,
	}

	uri := model.buildConnectionURI("testpass123")

	assert.Contains(t, uri, "directConnection=true")
	assert.Contains(t, uri, "tls=true")
	assert.Contains(t, uri, "tlsInsecure=true")
}

func Test_Validate_StandardConnection_RequiresPort(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}

	err := model.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port is required for standard connections")
}
