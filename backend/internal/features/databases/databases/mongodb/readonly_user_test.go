package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

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
			defer container.Client.Disconnect(context.Background())

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
			assert.True(t, strings.HasPrefix(username, "postgresus-"))

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
	container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion70)
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
	container := connectToMongodbContainer(t, env.TestMongodb70Port, tools.MongodbVersion70)
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
	host := "127.0.0.1"
	username := "root"
	password := "rootpassword"
	authDatabase := "admin"

	portInt, err := strconv.Atoi(port)
	assert.NoError(t, err)

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		username, password, host, portInt, dbName, authDatabase,
	)

	ctx := context.Background()
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
	return &MongodbDatabase{
		Version:      container.Version,
		Host:         container.Host,
		Port:         container.Port,
		Username:     container.Username,
		Password:     container.Password,
		Database:     container.Database,
		AuthDatabase: container.AuthDatabase,
		IsHttps:      false,
	}
}

func connectWithCredentials(
	t *testing.T,
	container *MongodbContainer,
	username, password string,
) *mongo.Client {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		username, password, container.Host, container.Port,
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
