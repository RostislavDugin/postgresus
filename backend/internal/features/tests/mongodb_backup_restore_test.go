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
	restores_models "postgresus-backend/internal/features/restores/models"
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
			testMongodbBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestoreMongodb_WithReadOnlyUser_RestoreIsSuccessful(t *testing.T) {
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
			testMongodbBackupRestoreWithReadOnlyUserForVersion(t, tc.version, tc.port)
		})
	}
}

func testMongodbBackupRestoreForVersion(
	t *testing.T,
	mongodbVersion tools.MongodbVersion,
	port string,
) {
	container, err := connectToMongodbContainer(mongodbVersion, port)
	if err != nil {
		t.Skipf("Skipping MongoDB %s test: %v", mongodbVersion, err)
		return
	}
	defer container.Client.Disconnect(context.Background())

	setupMongodbTestData(t, container)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("MongoDB Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_config.BackupEncryptionNone, user.Token,
	)

	createBackupViaAPI(t, router, database.ID, user.Token)

	backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mongodb_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := context.Background()
	_ = container.Client.Database(newDBName).Drop(ctx)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(storage.ID)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func testMongodbBackupRestoreWithEncryptionForVersion(
	t *testing.T,
	mongodbVersion tools.MongodbVersion,
	port string,
) {
	container, err := connectToMongodbContainer(mongodbVersion, port)
	if err != nil {
		t.Skipf("Skipping MongoDB %s test: %v", mongodbVersion, err)
		return
	}
	defer container.Client.Disconnect(context.Background())

	setupMongodbTestData(t, container)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		"MongoDB Encrypted Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB Encrypted Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_config.BackupEncryptionEncrypted, user.Token,
	)

	createBackupViaAPI(t, router, database.ID, user.Token)

	backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)
	assert.Equal(t, backups_config.BackupEncryptionEncrypted, backup.Encryption)

	newDBName := "restoreddb_mongodb_enc_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := context.Background()
	_ = container.Client.Database(newDBName).Drop(ctx)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(storage.ID)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func testMongodbBackupRestoreWithReadOnlyUserForVersion(
	t *testing.T,
	mongodbVersion tools.MongodbVersion,
	port string,
) {
	container, err := connectToMongodbContainer(mongodbVersion, port)
	if err != nil {
		t.Skipf("Skipping MongoDB %s test: %v", mongodbVersion, err)
		return
	}
	defer container.Client.Disconnect(context.Background())

	setupMongodbTestData(t, container)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		"MongoDB ReadOnly Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	readOnlyUser := createMongodbReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateMongodbDatabaseCredentialsViaAPI(
		t, router, database,
		readOnlyUser.Username, readOnlyUser.Password,
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, updatedDatabase.ID, storage.ID,
		backups_config.BackupEncryptionNone, user.Token,
	)

	createBackupViaAPI(t, router, updatedDatabase.ID, user.Token)

	backup := waitForBackupCompletion(t, router, updatedDatabase.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mongodb_ro_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := context.Background()
	_ = container.Client.Database(newDBName).Drop(ctx)

	dropMongodbUserSafe(container.Client, readOnlyUser.Username, container.AuthDatabase)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+updatedDatabase.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(storage.ID)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
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
			IsHttps:      false,
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
			IsHttps:      false,
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

func waitForMongodbRestoreCompletion(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	token string,
	timeout time.Duration,
) *restores_models.Restore {
	startTime := time.Now()
	pollInterval := 500 * time.Millisecond

	for {
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout waiting for MongoDB restore completion after %v", timeout)
		}

		var restoresList []*restores_models.Restore
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/restores/%s", backupID.String()),
			"Bearer "+token,
			http.StatusOK,
			&restoresList,
		)

		for _, restore := range restoresList {
			if restore.Status == restores_enums.RestoreStatusCompleted {
				return restore
			}
			if restore.Status == restores_enums.RestoreStatusFailed {
				failMsg := "unknown error"
				if restore.FailMessage != nil {
					failMsg = *restore.FailMessage
				}
				t.Fatalf("MongoDB restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func verifyMongodbDataIntegrity(t *testing.T, container *MongodbContainer, restoredDBName string) {
	ctx := context.Background()

	originalCollection := container.Client.Database(container.Database).Collection("test_data")
	restoredCollection := container.Client.Database(restoredDBName).Collection("test_data")

	originalCount, err := originalCollection.CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)

	restoredCount, err := restoredCollection.CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)

	assert.Equal(t, originalCount, restoredCount, "Should have same number of documents")

	var originalDocs []MongodbTestDataItem
	cursor, err := originalCollection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	assert.NoError(t, err)
	err = cursor.All(ctx, &originalDocs)
	assert.NoError(t, err)

	var restoredDocs []MongodbTestDataItem
	cursor, err = restoredCollection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	assert.NoError(t, err)
	err = cursor.All(ctx, &restoredDocs)
	assert.NoError(t, err)

	assert.Equal(t, len(originalDocs), len(restoredDocs), "Should have same number of documents")

	for i := range originalDocs {
		assert.Equal(t, originalDocs[i].ID, restoredDocs[i].ID, "ID should match")
		assert.Equal(t, originalDocs[i].Name, restoredDocs[i].Name, "Name should match")
		assert.Equal(t, originalDocs[i].Value, restoredDocs[i].Value, "Value should match")
	}
}

func connectToMongodbContainer(
	version tools.MongodbVersion,
	port string,
) (*MongodbContainer, error) {
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

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		username, password, host, portInt, dbName, authDatabase,
	)

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

func setupMongodbTestData(t *testing.T, container *MongodbContainer) {
	ctx := context.Background()
	collection := container.Client.Database(container.Database).Collection("test_data")

	_ = collection.Drop(ctx)

	testDocs := []interface{}{
		MongodbTestDataItem{
			ID:        "1",
			Name:      "test1",
			Value:     100,
			CreatedAt: time.Now().UTC(),
		},
		MongodbTestDataItem{
			ID:        "2",
			Name:      "test2",
			Value:     200,
			CreatedAt: time.Now().UTC(),
		},
		MongodbTestDataItem{
			ID:        "3",
			Name:      "test3",
			Value:     300,
			CreatedAt: time.Now().UTC(),
		},
	}

	_, err := collection.InsertMany(ctx, testDocs)
	assert.NoError(t, err)
}

func createMongodbReadOnlyUserViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
) *databases.CreateReadOnlyUserResponse {
	var database databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/databases/%s", databaseID.String()),
		"Bearer "+token,
		http.StatusOK,
		&database,
	)

	var response databases.CreateReadOnlyUserResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create-readonly-user",
		"Bearer "+token,
		database,
		http.StatusOK,
		&response,
	)

	return &response
}

func updateMongodbDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.Mongodb.Username = username
	database.Mongodb.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update MongoDB database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}

func dropMongodbUserSafe(client *mongo.Client, username, authDatabase string) {
	ctx := context.Background()
	adminDB := client.Database(authDatabase)
	_ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}
