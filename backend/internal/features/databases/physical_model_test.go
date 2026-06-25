package databases

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

type physicalFixture struct {
	name    string
	version tools.PostgresqlVersion
	port    func() string
}

func physicalFixtures() []physicalFixture {
	return []physicalFixture{
		{"pg17", tools.PostgresqlVersion17, func() string { return config.GetEnv().TestPhysicalPostgres17Port }},
		{"pg18", tools.PostgresqlVersion18, func() string { return config.GetEnv().TestPhysicalPostgres18Port }},
	}
}

func buildPhysicalDatabaseRequest(t *testing.T, workspaceID uuid.UUID, fx physicalFixture) Database {
	t.Helper()

	port, err := strconv.Atoi(fx.port())
	require.NoError(t, err)

	return Database{
		WorkspaceID: &workspaceID,
		Name:        "Physical DB " + uuid.New().String(),
		Type:        DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    fx.version,
			Host:       config.GetEnv().TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: postgresql_physical.BackupTypeFullOnly,
		},
	}
}

func createPhysicalDatabaseInternalAPI(
	t *testing.T,
	router *gin.Engine,
	workspaceID uuid.UUID,
	token string,
	fx physicalFixture,
) *Database {
	t.Helper()

	request := buildPhysicalDatabaseRequest(t, workspaceID, fx)

	w := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create",
		"Bearer "+token, request,
	)
	require.Equal(t, http.StatusCreated, w.Code, "create physical database failed: %s", w.Body.String())

	var database Database
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))

	return &database
}

func Test_TestConnection_OnPhysicalDatabase_DispatchesToReplicationConnection(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(database)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/"+database.ID.String()+"/test-connection",
				"Bearer "+owner.Token,
				nil,
				http.StatusOK,
			)
		})
	}
}

func Test_CreateReplicationOnlyUser_OnPhysicalDatabase_ReturnsCredentials(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(database)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			var response CreateReadOnlyUserResponse
			test_utils.MakePostRequestAndUnmarshal(
				t, router,
				"/api/v1/databases/create-replication-only-user",
				"Bearer "+owner.Token,
				database,
				http.StatusOK,
				&response,
			)

			assert.NotEmpty(t, response.Username)
			assert.NotEmpty(t, response.Password)
			assert.Contains(t, response.Username, "databasus-")
		})
	}
}

func Test_TestDatabaseConnectionDirect_OnReplicationReadyCluster_ReturnsSuccess(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(database)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/test-connection-direct",
				"Bearer "+owner.Token,
				database,
				http.StatusOK,
			)
		})
	}
}

func Test_CreatePhysicalDatabase_DetectsVersionFromServer_OverridingPayloadVersion(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			request := buildPhysicalDatabaseRequest(t, workspace.ID, fx)
			request.PostgresqlPhysical.Version = tools.PostgresqlVersion16

			w := workspaces_testing.MakeAPIRequest(
				router, "POST", "/api/v1/databases/create",
				"Bearer "+owner.Token, request,
			)
			require.Equal(t, http.StatusCreated, w.Code, "create failed: %s", w.Body.String())

			var database Database
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))
			defer func() {
				RemoveTestDatabase(&database)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			assert.Equal(t, fx.version, database.PostgresqlPhysical.Version)
		})
	}
}

func Test_CreatePhysicalDatabase_OnUnsupportedServerVersion_ReturnsError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	port, err := strconv.Atoi(config.GetEnv().TestLogicalPostgres16Port)
	require.NoError(t, err)

	request := Database{
		WorkspaceID: &workspace.ID,
		Name:        "Physical DB " + uuid.New().String(),
		Type:        DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    tools.PostgresqlVersion18,
			Host:       config.GetEnv().TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: postgresql_physical.BackupTypeFullOnly,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create",
		"Bearer "+owner.Token, request,
	)

	require.Equal(t, http.StatusBadRequest, w.Code, "expected rejection, got: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "17 or 18")
}

func Test_CreateReplicationOnlyUser_OnLogicalDatabase_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	logicalDB := createTestDatabaseViaAPI("Logical DB", workspace.ID, owner.Token, router)
	defer func() {
		RemoveTestDatabase(logicalDB)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	testResp := test_utils.MakePostRequest(
		t, router,
		"/api/v1/databases/create-replication-only-user",
		"Bearer "+owner.Token,
		logicalDB,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "POSTGRES_PHYSICAL")
}

func Test_CopyDatabase_OnPhysicalDatabase_DoesNotCopySystemIdentifier(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			originalDatabase := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)

			originalPersistedDatabase, err := databaseRepository.FindByID(originalDatabase.ID)
			require.NoError(t, err)
			require.NotNil(t, originalPersistedDatabase.PostgresqlPhysical)
			require.NotNil(t, originalPersistedDatabase.PostgresqlPhysical.SystemIdentifier,
				"original SystemIdentifier should be populated on create")
			originalSlotName := originalPersistedDatabase.PostgresqlPhysical.ReplicationSlotName

			var copiedDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t, router,
				"/api/v1/databases/"+originalDatabase.ID.String()+"/copy",
				"Bearer "+owner.Token,
				nil,
				http.StatusCreated,
				&copiedDatabase,
			)
			defer func() {
				RemoveTestDatabase(&copiedDatabase)
				RemoveTestDatabase(originalDatabase)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			copiedPersistedDatabase, err := databaseRepository.FindByID(copiedDatabase.ID)
			require.NoError(t, err)
			require.NotNil(t, copiedPersistedDatabase.PostgresqlPhysical)
			assert.NotEqual(t, originalDatabase.ID, copiedDatabase.ID)
			assert.NotEqual(t, originalSlotName, copiedPersistedDatabase.PostgresqlPhysical.ReplicationSlotName,
				"copied database must have a fresh replication slot name")
		})
	}
}
