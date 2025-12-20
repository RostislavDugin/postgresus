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
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"postgresus-backend/internal/config"
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	pgtypes "postgresus-backend/internal/features/databases/databases/postgresql"
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

const createAndFillTableQuery = `
DROP TABLE IF EXISTS test_data;

CREATE TABLE test_data (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    value INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300);
`

type PostgresContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  string
	DB       *sqlx.DB
}

type TestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

func Test_BackupAndRestorePostgresql_RestoreIsSuccesful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestorePostgresqlWithEncryption_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestoreSupabase_PublicSchemaOnly_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()

	if env.TestSupabaseHost == "" {
		t.Skip("Skipping Supabase test: missing environment variables")
	}

	portInt, err := strconv.Atoi(env.TestSupabasePort)
	assert.NoError(t, err)

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		env.TestSupabaseHost,
		portInt,
		env.TestSupabaseUsername,
		env.TestSupabasePassword,
		env.TestSupabaseDatabase,
	)

	supabaseDB, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)
	defer supabaseDB.Close()

	tableName := fmt.Sprintf("backup_test_%s", uuid.New().String()[:8])
	createTableQuery := fmt.Sprintf(`
		DROP TABLE IF EXISTS public.%s;
		CREATE TABLE public.%s (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO public.%s (name, value) VALUES
			('test1', 100),
			('test2', 200),
			('test3', 300);
	`, tableName, tableName, tableName)

	_, err = supabaseDB.Exec(createTableQuery)
	assert.NoError(t, err)

	defer func() {
		_, _ = supabaseDB.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS public.%s`, tableName))
	}()

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Supabase Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createSupabaseDatabaseViaAPI(
		t, router, "Supabase Test Database", workspace.ID,
		env.TestSupabaseHost, portInt,
		env.TestSupabaseUsername, env.TestSupabasePassword, env.TestSupabaseDatabase,
		[]string{"public"},
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_config.BackupEncryptionNone, user.Token,
	)

	createBackupViaAPI(t, router, database.ID, user.Token)

	backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

	_, err = supabaseDB.Exec(fmt.Sprintf(`DELETE FROM public.%s`, tableName))
	assert.NoError(t, err)

	var countAfterDelete int
	err = supabaseDB.Get(
		&countAfterDelete,
		fmt.Sprintf(`SELECT COUNT(*) FROM public.%s`, tableName),
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, countAfterDelete, "Table should be empty after delete")

	createSupabaseRestoreViaAPI(
		t, router, backup.ID,
		env.TestSupabaseHost, portInt,
		env.TestSupabaseUsername, env.TestSupabasePassword, env.TestSupabaseDatabase,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var countAfterRestore int
	err = supabaseDB.Get(
		&countAfterRestore,
		fmt.Sprintf(`SELECT COUNT(*) FROM public.%s`, tableName),
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, countAfterRestore, "Table should have 3 rows after restore")

	var restoredData []TestDataItem
	err = supabaseDB.Select(
		&restoredData,
		fmt.Sprintf(`SELECT id, name, value, created_at FROM public.%s ORDER BY id`, tableName),
	)
	assert.NoError(t, err)
	assert.Len(t, restoredData, 3)
	assert.Equal(t, "test1", restoredData[0].Name)
	assert.Equal(t, 100, restoredData[0].Value)
	assert.Equal(t, "test2", restoredData[1].Name)
	assert.Equal(t, 200, restoredData[1].Value)
	assert.Equal(t, "test3", restoredData[2].Name)
	assert.Equal(t, 300, restoredData[2].Value)

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

func Test_BackupPostgresql_SchemaSelection_AllSchemasWhenNoneSpecified(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testSchemaSelectionAllSchemasForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestorePostgresql_WithExcludeExtensions_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreWithExcludeExtensionsForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestorePostgresql_WithoutExcludeExtensions_ExtensionsAreRecovered(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreWithoutExcludeExtensionsForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupPostgresql_SchemaSelection_OnlySpecifiedSchemas(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testSchemaSelectionOnlySpecifiedSchemasForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestorePostgresql_WithReadOnlyUser_RestoreIsSuccessful(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreWithReadOnlyUserForVersion(t, tc.version, tc.port)
		})
	}
}

func testBackupRestoreForVersion(t *testing.T, pgVersion string, port string) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	_, err = container.DB.Exec(createAndFillTableQuery)
	assert.NoError(t, err)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Test Database", workspace.ID,
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

	newDBName := "restoreddb"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_data')",
	)
	assert.NoError(t, err)
	assert.True(t, tableExists, "Table 'test_data' should exist in restored database")

	verifyDataIntegrity(t, container.DB, newDB)

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

func testSchemaSelectionAllSchemasForVersion(t *testing.T, pgVersion string, port string) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer container.DB.Close()

	_, err = container.DB.Exec(`
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;
		
		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);
		
		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS public.public_table;
			DROP SCHEMA IF EXISTS schema_a CASCADE;
			DROP SCHEMA IF EXISTS schema_b CASCADE;
		`)
	}()

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Schema Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithSchemasViaAPI(
		t, router, "All Schemas Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		nil,
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_config.BackupEncryptionNone, user.Token,
	)

	createBackupViaAPI(t, router, database.ID, user.Token)

	backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

	newDBName := "restored_all_schemas_" + pgVersion
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var publicTableExists bool
	err = newDB.Get(&publicTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'public_table'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, publicTableExists, "public.public_table should exist in restored database")

	var schemaATableExists bool
	err = newDB.Get(&schemaATableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'schema_a' AND table_name = 'table_a'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaATableExists, "schema_a.table_a should exist in restored database")

	var schemaBTableExists bool
	err = newDB.Get(&schemaBTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'schema_b' AND table_name = 'table_b'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaBTableExists, "schema_b.table_b should exist in restored database")

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

func testBackupRestoreWithExcludeExtensionsForVersion(t *testing.T, pgVersion string, port string) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer container.DB.Close()

	// Create table with uuid-ossp extension and add a comment on the extension
	// The comment is important to test that COMMENT ON EXTENSION statements are also excluded
	_, err = container.DB.Exec(`
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		COMMENT ON EXTENSION "uuid-ossp" IS 'Test comment on uuid-ossp extension';
		
		DROP TABLE IF EXISTS test_extension_data;
		CREATE TABLE test_extension_data (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		
		INSERT INTO test_extension_data (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS test_extension_data;
			DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		`)
	}()

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Extension Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Extension Test Database", workspace.ID,
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

	// Create new database for restore with extension pre-installed
	newDBName := "restored_exclude_ext_" + pgVersion
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Pre-install the extension in the target database (simulating managed service behavior)
	_, err = newDB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	assert.NoError(t, err)

	// Restore with isExcludeExtensions=true
	createRestoreWithOptionsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		true, // isExcludeExtensions
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	// Verify the table was restored
	var tableExists bool
	err = newDB.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'test_extension_data'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, tableExists, "test_extension_data should exist in restored database")

	// Verify data was restored
	var count int
	err = newDB.Get(&count, `SELECT COUNT(*) FROM test_extension_data`)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 rows after restore")

	// Verify extension still works (uuid_generate_v4 should work)
	var newUUID string
	err = newDB.Get(&newUUID, `SELECT uuid_generate_v4()::text`)
	assert.NoError(t, err)
	assert.NotEmpty(t, newUUID, "uuid_generate_v4 should work")

	// Cleanup
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

func testBackupRestoreWithoutExcludeExtensionsForVersion(
	t *testing.T,
	pgVersion string,
	port string,
) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer container.DB.Close()

	// Create table with uuid-ossp extension
	_, err = container.DB.Exec(`
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		
		DROP TABLE IF EXISTS test_extension_recovery;
		CREATE TABLE test_extension_recovery (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		
		INSERT INTO test_extension_recovery (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS test_extension_recovery;
			DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		`)
	}()

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		"Extension Recovery Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Extension Recovery Test Database", workspace.ID,
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

	// Create new database for restore WITHOUT pre-installed extension
	newDBName := "restored_with_ext_" + pgVersion
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Verify extension does NOT exist before restore
	var extensionExistsBefore bool
	err = newDB.Get(&extensionExistsBefore, `
		SELECT EXISTS (
			SELECT FROM pg_extension WHERE extname = 'uuid-ossp'
		)
	`)
	assert.NoError(t, err)
	assert.False(t, extensionExistsBefore, "Extension should NOT exist before restore")

	// Restore with isExcludeExtensions=false (extensions should be recovered)
	createRestoreWithOptionsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		false, // isExcludeExtensions = false means extensions ARE included
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	// Verify the extension was recovered
	var extensionExists bool
	err = newDB.Get(&extensionExists, `
		SELECT EXISTS (
			SELECT FROM pg_extension WHERE extname = 'uuid-ossp'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, extensionExists, "Extension 'uuid-ossp' should be recovered during restore")

	// Verify the table was restored
	var tableExists bool
	err = newDB.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'test_extension_recovery'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, tableExists, "test_extension_recovery should exist in restored database")

	// Verify data was restored
	var count int
	err = newDB.Get(&count, `SELECT COUNT(*) FROM test_extension_recovery`)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 rows after restore")

	// Verify extension works (uuid_generate_v4 should work)
	var newUUID string
	err = newDB.Get(&newUUID, `SELECT uuid_generate_v4()::text`)
	assert.NoError(t, err)
	assert.NotEmpty(t, newUUID, "uuid_generate_v4 should work after extension recovery")

	// Cleanup
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

func testBackupRestoreWithReadOnlyUserForVersion(t *testing.T, pgVersion string, port string) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	_, err = container.DB.Exec(createAndFillTableQuery)
	assert.NoError(t, err)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("ReadOnly Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	readOnlyUser := createReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateDatabaseCredentialsViaAPI(
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

	newDBName := "restoreddb_readonly"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_data')",
	)
	assert.NoError(t, err)
	assert.True(t, tableExists, "Table 'test_data' should exist in restored database")

	verifyDataIntegrity(t, container.DB, newDB)

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

func testSchemaSelectionOnlySpecifiedSchemasForVersion(
	t *testing.T,
	pgVersion string,
	port string,
) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer container.DB.Close()

	_, err = container.DB.Exec(`
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;
		
		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);
		
		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS public.public_table;
			DROP SCHEMA IF EXISTS schema_a CASCADE;
			DROP SCHEMA IF EXISTS schema_b CASCADE;
		`)
	}()

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Schema Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithSchemasViaAPI(
		t, router, "Specific Schemas Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		[]string{"public", "schema_a"},
		user.Token,
	)

	enableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_config.BackupEncryptionNone, user.Token,
	)

	createBackupViaAPI(t, router, database.ID, user.Token)

	backup := waitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups.BackupStatusCompleted, backup.Status)

	newDBName := "restored_specific_schemas_" + pgVersion
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var publicTableExists bool
	err = newDB.Get(&publicTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'public_table'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, publicTableExists, "public.public_table should exist (was included)")

	var schemaATableExists bool
	err = newDB.Get(&schemaATableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'schema_a' AND table_name = 'table_a'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaATableExists, "schema_a.table_a should exist (was included)")

	var schemaBTableExists bool
	err = newDB.Get(&schemaBTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'schema_b' AND table_name = 'table_b'
		)
	`)
	assert.NoError(t, err)
	assert.False(t, schemaBTableExists, "schema_b.table_b should NOT exist (was excluded)")

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

func testBackupRestoreWithEncryptionForVersion(t *testing.T, pgVersion string, port string) {
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	_, err = container.DB.Exec(createAndFillTableQuery)
	assert.NoError(t, err)

	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
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

	newDBName := "restoreddb_encrypted"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_enums.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_data')",
	)
	assert.NoError(t, err)
	assert.True(t, tableExists, "Table 'test_data' should exist in restored database")

	verifyDataIntegrity(t, container.DB, newDB)

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

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config.GetBackupConfigController(),
		backups.GetBackupController(),
		restores.GetRestoreController(),
	)
	return router
}

func waitForBackupCompletion(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
	timeout time.Duration,
) *backups.Backup {
	startTime := time.Now()
	pollInterval := 500 * time.Millisecond

	for {
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout waiting for backup completion after %v", timeout)
		}

		var response backups.GetBackupsResponse
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/backups?database_id=%s&limit=1", databaseID.String()),
			"Bearer "+token,
			http.StatusOK,
			&response,
		)

		if len(response.Backups) > 0 {
			backup := response.Backups[0]
			if backup.Status == backups.BackupStatusCompleted {
				return backup
			}
			if backup.Status == backups.BackupStatusFailed {
				failMsg := "unknown error"
				if backup.FailMessage != nil {
					failMsg = *backup.FailMessage
				}
				t.Fatalf("Backup failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func waitForRestoreCompletion(
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
			t.Fatalf("Timeout waiting for restore completion after %v", timeout)
		}

		var restores []*restores_models.Restore
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/restores/%s", backupID.String()),
			"Bearer "+token,
			http.StatusOK,
			&restores,
		)

		for _, restore := range restores {
			if restore.Status == restores_enums.RestoreStatusCompleted {
				return restore
			}
			if restore.Status == restores_enums.RestoreStatusFailed {
				failMsg := "unknown error"
				if restore.FailMessage != nil {
					failMsg = *restore.FailMessage
				}
				t.Fatalf("Restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func createDatabaseViaAPI(
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
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &pgtypes.PostgresqlDatabase{
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

func enableBackupsViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	encryption backups_config.BackupEncryption,
	token string,
) {
	var backupConfig backups_config.BackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/database/%s", databaseID.String()),
		"Bearer "+token,
		http.StatusOK,
		&backupConfig,
	)

	storage := &storages.Storage{ID: storageID}
	backupConfig.IsBackupsEnabled = true
	backupConfig.Storage = storage
	backupConfig.Encryption = encryption

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+token,
		backupConfig,
		http.StatusOK,
	)
}

func createBackupViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
) {
	request := backups.MakeBackupRequest{DatabaseID: databaseID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+token,
		request,
		http.StatusOK,
	)
}

func createRestoreViaAPI(
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
	createRestoreWithOptionsViaAPI(
		t,
		router,
		backupID,
		host,
		port,
		username,
		password,
		database,
		false,
		token,
	)
}

func createRestoreWithOptionsViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	isExcludeExtensions bool,
	token string,
) {
	request := restores.RestoreBackupRequest{
		PostgresqlDatabase: &pgtypes.PostgresqlDatabase{
			Host:                host,
			Port:                port,
			Username:            username,
			Password:            password,
			Database:            &database,
			IsExcludeExtensions: isExcludeExtensions,
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

func createDatabaseWithSchemasViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	includeSchemas []string,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &pgtypes.PostgresqlDatabase{
			Host:           host,
			Port:           port,
			Username:       username,
			Password:       password,
			Database:       &database,
			IncludeSchemas: includeSchemas,
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
		t.Fatalf(
			"Failed to create database with schemas. Status: %d, Body: %s",
			w.Code,
			w.Body.String(),
		)
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createSupabaseDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	includeSchemas []string,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &pgtypes.PostgresqlDatabase{
			Host:           host,
			Port:           port,
			Username:       username,
			Password:       password,
			Database:       &database,
			IsHttps:        true,
			IncludeSchemas: includeSchemas,
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
		t.Fatalf(
			"Failed to create Supabase database. Status: %d, Body: %s",
			w.Code,
			w.Body.String(),
		)
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createSupabaseRestoreViaAPI(
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
		PostgresqlDatabase: &pgtypes.PostgresqlDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			IsHttps:  true,
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

func verifyDataIntegrity(t *testing.T, originalDB *sqlx.DB, restoredDB *sqlx.DB) {
	var originalData []TestDataItem
	var restoredData []TestDataItem

	err := originalDB.Select(&originalData, "SELECT * FROM test_data ORDER BY id")
	assert.NoError(t, err)

	err = restoredDB.Select(&restoredData, "SELECT * FROM test_data ORDER BY id")
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

func createReadOnlyUserViaAPI(
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

func updateDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.Postgresql.Username = username
	database.Postgresql.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}

func connectToPostgresContainer(version string, port string) (*PostgresContainer, error) {
	dbName := "testdb"
	password := "testpassword"
	username := "testuser"
	host := "localhost"

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port: %w", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, portInt, username, password, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		DB:       db,
	}, nil
}
