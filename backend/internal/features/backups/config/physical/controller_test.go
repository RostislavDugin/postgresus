package backups_config_physical

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

func createPhysicalTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetBackupConfigController(),
		storages.GetStorageController(),
		notifiers.GetNotifierController(),
	)

	storages.SetupDependencies()
	databases.SetupDependencies()
	notifiers.SetupDependencies()
	SetupDependencies()

	return router
}

func createPhysicalDatabaseViaAPI(
	t *testing.T,
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
	backupType postgresql_physical.BackupType,
	versionTag string,
) *databases.Database {
	t.Helper()

	env := config.GetEnv()

	var portStr string
	var version tools.PostgresqlVersion

	switch versionTag {
	case "17":
		portStr = env.TestPhysicalPostgres17Port
		version = tools.PostgresqlVersion17
	case "18":
		portStr = env.TestPhysicalPostgres18Port
		version = tools.PostgresqlVersion18
	default:
		t.Fatalf("unsupported physical postgres version tag: %s", versionTag)
	}

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	request := databases.Database{
		WorkspaceID: &workspaceID,
		Name:        name,
		Type:        databases.DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    version,
			Host:       env.TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: backupType,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)
	require.Equal(t, http.StatusCreated, w.Code, "create physical database failed: %s", w.Body.String())

	var database databases.Database
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))

	return &database
}

func validPhysicalConfigForFullOnly(databaseID uuid.UUID) PhysicalBackupConfig {
	timeOfDay := "04:00"

	return PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: true,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Retention: RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  7,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
	}
}

func Test_SaveBackupConfig_PhysicalWithDifferentRoles_EnforcesPermissions(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can save physical backup config",
			workspaceRole:      new(users_enums.WorkspaceRoleOwner),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can save physical backup config",
			workspaceRole:      new(users_enums.WorkspaceRoleAdmin),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can save physical backup config",
			workspaceRole:      new(users_enums.WorkspaceRoleMember),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot save physical backup config",
			workspaceRole:      new(users_enums.WorkspaceRoleViewer),
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can save physical backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createPhysicalTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			database := createPhysicalDatabaseViaAPI(
				t,
				"Physical DB "+uuid.New().String(),
				workspace.ID,
				owner.Token,
				router,
				postgresql_physical.BackupTypeFullOnly,
				"17",
			)

			defer func() {
				databases.RemoveTestDatabase(database)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			var testUserToken string

			switch {
			case tt.isGlobalAdmin:
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			case tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner:
				testUserToken = owner.Token
			case tt.workspaceRole != nil:
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace, member, *tt.workspaceRole, owner.Token, router,
				)
				testUserToken = member.Token
			}

			request := validPhysicalConfigForFullOnly(database.ID)

			var response PhysicalBackupConfig
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/physical/save",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.True(t, response.IsBackupsEnabled)
				assert.Equal(t, RetentionFullBackups, response.Retention)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_SaveBackupConfig_PhysicalFullOnlyWithChainsRetention_ReturnsBadRequest(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(database)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	request.Retention = RetentionChains
	request.ChainsRetention = ChainsRetention{Count: 3}
	request.FullBackupsRetention = FullBackupsRetention{}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "FULL_ONLY")
}

func Test_SaveBackupConfig_PhysicalFullOnlyWithIncrementalInterval_ReturnsBadRequest(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(database)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	incrementalTime := "02:00"
	request.IncrementalBackupInterval = intervals.Interval{
		Type:      intervals.IntervalHourly,
		TimeOfDay: &incrementalTime,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "incremental cadence cannot be set")
}

func Test_GetBackupConfig_PhysicalWhenNoneExists_InitializesDefaults(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(database)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	var response PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.False(t, response.IsBackupsEnabled)
	assert.Equal(t, RetentionFullBackups, response.Retention)
	assert.Equal(t, FullBackupsRetentionPolicyLastN, response.FullBackupsRetention.Policy)
	assert.Equal(t, 7, response.FullBackupsRetention.Count)
	assert.Contains(t, response.SendNotificationsOn, NotificationChainBroken)
}
