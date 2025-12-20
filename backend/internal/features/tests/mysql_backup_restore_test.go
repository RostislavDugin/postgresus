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
	mysqltypes "postgresus-backend/internal/features/databases/databases/mysql"
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

const dropMysqlTestTableQuery = `DROP TABLE IF EXISTS test_data`

const createMysqlTestTableQuery = `
CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

const insertMysqlTestDataQuery = `
INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300)`

type MysqlContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MysqlVersion
	DB       *sqlx.DB
}

type MysqlTestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

func Test_BackupAndRestoreMysql_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MysqlVersion
		port    string
	}{
		{"MySQL 5.7", tools.MysqlVersion57, env.TestMysql57Port},
		{"MySQL 8.0", tools.MysqlVersion80, env.TestMysql80Port},
		{"MySQL 8.4", tools.MysqlVersion84, env.TestMysql84Port},
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
		version tools.MysqlVersion
		port    string
	}{
		{"MySQL 5.7", tools.MysqlVersion57, env.TestMysql57Port},
		{"MySQL 8.0", tools.MysqlVersion80, env.TestMysql80Port},
		{"MySQL 8.4", tools.MysqlVersion84, env.TestMysql84Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testMysqlBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestoreMysql_WithReadOnlyUser_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MysqlVersion
		port    string
	}{
		{"MySQL 5.7", tools.MysqlVersion57, env.TestMysql57Port},
		{"MySQL 8.0", tools.MysqlVersion80, env.TestMysql80Port},
		{"MySQL 8.4", tools.MysqlVersion84, env.TestMysql84Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testMysqlBackupRestoreWithReadOnlyUserForVersion(t, tc.version, tc.port)
		})
	}
}

func testMysqlBackupRestoreForVersion(t *testing.T, mysqlVersion tools.MysqlVersion, port string) {
	container, err := connectToMysqlContainer(mysqlVersion, port)
	if err != nil {
		t.Skipf("Skipping MySQL %s test: %v", mysqlVersion, err)
		return
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("MySQL Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
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

	newDBName := "restoreddb_mysql"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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

func testMysqlBackupRestoreWithEncryptionForVersion(
	t *testing.T,
	mysqlVersion tools.MysqlVersion,
	port string,
) {
	container, err := connectToMysqlContainer(mysqlVersion, port)
	if err != nil {
		t.Skipf("Skipping MySQL %s test: %v", mysqlVersion, err)
		return
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		"MySQL Encrypted Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL Encrypted Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
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

	newDBName := "restoreddb_mysql_encrypted"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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

func testMysqlBackupRestoreWithReadOnlyUserForVersion(
	t *testing.T,
	mysqlVersion tools.MysqlVersion,
	port string,
) {
	container, err := connectToMysqlContainer(mysqlVersion, port)
	if err != nil {
		t.Skipf("Skipping MySQL %s test: %v", mysqlVersion, err)
		return
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		"MySQL ReadOnly Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	readOnlyUser := createMysqlReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateMysqlDatabaseCredentialsViaAPI(
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

	newDBName := "restoreddb_mysql_readonly"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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
	version tools.MysqlVersion,
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
		t.Fatalf("Failed to create MySQL database. Status: %d, Body: %s", w.Code, w.Body.String())
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
	version tools.MysqlVersion,
	token string,
) {
	request := restores.RestoreBackupRequest{
		MysqlDatabase: &mysqltypes.MysqlDatabase{
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

func waitForMysqlRestoreCompletion(
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
			t.Fatalf("Timeout waiting for MySQL restore completion after %v", timeout)
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
				t.Fatalf("MySQL restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func verifyMysqlDataIntegrity(t *testing.T, originalDB *sqlx.DB, restoredDB *sqlx.DB) {
	var originalData []MysqlTestDataItem
	var restoredData []MysqlTestDataItem

	err := originalDB.Select(
		&originalData,
		"SELECT id, name, value, created_at FROM test_data ORDER BY id",
	)
	assert.NoError(t, err)

	err = restoredDB.Select(
		&restoredData,
		"SELECT id, name, value, created_at FROM test_data ORDER BY id",
	)
	assert.NoError(t, err)

	assert.Equal(t, len(originalData), len(restoredData), "Should have same number of rows")

	if len(originalData) > 0 && len(restoredData) > 0 {
		for i := range originalData {
			assert.Equal(t, originalData[i].ID, restoredData[i].ID, "ID should match")
			assert.Equal(t, originalData[i].Name, restoredData[i].Name, "Name should match")
			assert.Equal(t, originalData[i].Value, restoredData[i].Value, "Value should match")
		}
	}
}

func connectToMysqlContainer(version tools.MysqlVersion, port string) (*MysqlContainer, error) {
	if port == "" {
		return nil, fmt.Errorf("MySQL %s port not configured", version)
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
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	return &MysqlContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}, nil
}

func setupMysqlTestData(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec(dropMysqlTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(createMysqlTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(insertMysqlTestDataQuery)
	assert.NoError(t, err)
}

func createMysqlReadOnlyUserViaAPI(
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

func updateMysqlDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.Mysql.Username = username
	database.Mysql.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update MySQL database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}
