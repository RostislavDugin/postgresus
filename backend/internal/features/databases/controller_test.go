package databases

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/audit_logs"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
	"databasus-backend/internal/util/walmath"
)

func Test_CreateDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace member can create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace viewer cannot create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can create database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			request := Database{
				Name:              "Test Database",
				WorkspaceID:       &workspace.ID,
				Type:              DatabaseTypePostgresLogical,
				PostgresqlLogical: getTestPostgresConfig(),
			}

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				defer RemoveTestDatabase(&response)
				assert.Equal(t, "Test Database", response.Name)
				assert.NotEqual(t, uuid.Nil, response.ID)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CreateDatabase_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := Database{
		Name:              "Test Database",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_CreateDatabase_WithoutConnectionFields_ValidationFails(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	request := Database{
		Name:        "Test Database",
		WorkspaceID: &workspace.ID,
		Type:        DatabaseTypePostgresLogical,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			CpuCount: 1,
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "host is required")
}

func Test_UpdateDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can update database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(database)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			database.Name = "Updated Database"

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+testUserToken,
				database,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, "Updated Database", response.Name)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_UpdateDatabase_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(database)

	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
	database.Name = "Hacked Name"

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+nonMember.Token,
		database,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_UpdateDatabase_WhenDatabaseTypeChanged_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(database)

	database.Type = DatabaseTypeMysql

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "database type cannot be changed")
}

func Test_DeleteDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace member can delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace viewer cannot delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "global admin can delete database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			testResp := test_utils.MakeDeleteRequest(
				t,
				router,
				"/api/v1/databases/"+database.ID.String(),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if !tt.expectSuccess {
				defer RemoveTestDatabase(database)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabase_PermissionsEnforced(t *testing.T) {
	memberRole := users_enums.WorkspaceRoleViewer
	tests := []struct {
		name               string
		userRole           *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace member can get database",
			userRole:           &memberRole,
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get database",
			userRole:           nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can get database",
			userRole:           nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(database)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.userRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.userRole,
					owner.Token,
					router,
				)
				testUser = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			var response Database
			testResp := test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+database.ID.String(),
				"Bearer "+testUser,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.ID)
				assert.Equal(t, "Test Database", response.Name)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabasesByWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		isMember           bool
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace member can list databases",
			isMember:           true,
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot list databases",
			isMember:           false,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can list databases",
			isMember:           false,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			db1 := createTestDatabaseViaAPI("Database 1", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(db1)
			db2 := createTestDatabaseViaAPI("Database 2", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(db2)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.isMember {
				testUser = owner.Token
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			if tt.expectSuccess {
				var response []Database
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/databases?workspace_id="+workspace.ID.String(),
					"Bearer "+testUser,
					tt.expectedStatusCode,
					&response,
				)
				assert.GreaterOrEqual(t, len(response), 2)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/databases?workspace_id="+workspace.ID.String(),
					"Bearer "+testUser,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabasesByWorkspace_WhenMultipleDatabasesExist_ReturnsCorrectCount(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	db1 := createTestDatabaseViaAPI("Database 1", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(db1)
	db2 := createTestDatabaseViaAPI("Database 2", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(db2)
	db3 := createTestDatabaseViaAPI("Database 3", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(db3)

	var response []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, 3, len(response))
}

func Test_GetDatabasesByWorkspace_WithPhysicalBackups_ReturnsNewestBackupTimeAcrossTypes(t *testing.T) {
	const walSegmentBytes = 16 * 1024 * 1024

	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Phys Last Backup "+uuid.NewString(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	storage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(storage.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	backedUp := CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	defer func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, backedUp.ID)
		RemoveTestDatabase(backedUp)
	}()
	noBackups := CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	defer RemoveTestDatabase(noBackups)

	base := time.Now().UTC().Add(-time.Hour)

	fullBackup := physical_testing.NewTestCompletedFullBackup(
		backedUp.ID, storage.ID, 1, walmath.LSN(0), walmath.LSN(walSegmentBytes))
	fullBackup.CreatedAt = base
	fullBackup.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, fullBackup)

	incrementalBackup := physical_testing.NewTestCompletedIncrementalBackup(
		backedUp.ID, storage.ID, fullBackup.ID, nil, 1, walmath.LSN(walSegmentBytes), walmath.LSN(2*walSegmentBytes))
	incrementalBackup.CreatedAt = base.Add(10 * time.Minute)
	physical_testing.CreateTestIncrementalBackup(t, incrementalBackup)

	// Newest of the three - the value the card must show.
	walSegment := physical_testing.NewTestWalSegment(
		backedUp.ID, storage.ID, 1, "000000010000000000000001",
		walmath.LSN(2*walSegmentBytes), walmath.LSN(3*walSegmentBytes))
	walSegment.ReceivedAt = base.Add(20 * time.Minute)
	physical_testing.CreateTestWalSegment(t, walSegment)

	var response []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	backedUpListed := findDatabaseByID(t, response, backedUp.ID)
	require.NotNil(t, backedUpListed.LastBackupTime, "physical database with backups must expose a last backup time")
	assert.WithinDuration(t, walSegment.ReceivedAt, *backedUpListed.LastBackupTime, time.Second)

	noBackupsListed := findDatabaseByID(t, response, noBackups.ID)
	assert.Nil(t, noBackupsListed.LastBackupTime, "physical database without backups must have no last backup time")
}

func findDatabaseByID(t *testing.T, databases []Database, databaseID uuid.UUID) Database {
	t.Helper()

	for _, database := range databases {
		if database.ID == databaseID {
			return database
		}
	}

	t.Fatalf("database %s not found in response", databaseID)

	return Database{}
}

func Test_GetDatabasesByWorkspace_EnsuresCrossWorkspaceIsolation(t *testing.T) {
	router := createTestRouter()
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace("Workspace 1", owner1, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace1, router)

	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace("Workspace 2", owner2, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace2, router)

	workspace1Db1 := createTestDatabaseViaAPI("Workspace1 DB1", workspace1.ID, owner1.Token, router)
	defer RemoveTestDatabase(workspace1Db1)
	workspace1Db2 := createTestDatabaseViaAPI("Workspace1 DB2", workspace1.ID, owner1.Token, router)
	defer RemoveTestDatabase(workspace1Db2)

	workspace2Db1 := createTestDatabaseViaAPI("Workspace2 DB1", workspace2.ID, owner2.Token, router)
	defer RemoveTestDatabase(workspace2Db1)

	var workspace1Dbs []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace1.ID.String(),
		"Bearer "+owner1.Token,
		http.StatusOK,
		&workspace1Dbs,
	)

	var workspace2Dbs []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace2.ID.String(),
		"Bearer "+owner2.Token,
		http.StatusOK,
		&workspace2Dbs,
	)

	assert.Equal(t, 2, len(workspace1Dbs))
	assert.Equal(t, 1, len(workspace2Dbs))

	for _, db := range workspace1Dbs {
		assert.Equal(t, workspace1.ID, *db.WorkspaceID)
	}

	for _, db := range workspace2Dbs {
		assert.Equal(t, workspace2.ID, *db.WorkspaceID)
	}
}

func Test_CopyDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace member can copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace viewer cannot copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can copy database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(database)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+database.ID.String()+"/copy",
				"Bearer "+testUserToken,
				nil,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				defer RemoveTestDatabase(&response)
				assert.NotEqual(t, database.ID, response.ID)
				assert.Contains(t, response.Name, "(Copy)")
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CopyDatabase_CopyStaysInSameWorkspace(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(database)

	var response Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String()+"/copy",
		"Bearer "+owner.Token,
		nil,
		http.StatusCreated,
		&response,
	)

	defer RemoveTestDatabase(&response)

	assert.NotEqual(t, database.ID, response.ID)
	assert.Equal(t, "Test Database (Copy)", response.Name)
	assert.Equal(t, workspace.ID, *response.WorkspaceID)
	assert.Equal(t, database.Type, response.Type)
}

func Test_CreateDatabase_PasswordIsEncryptedInDB(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	pgConfig := getTestPostgresConfig()
	plainPassword := "testpassword"
	pgConfig.Password = plainPassword
	request := Database{
		Name:              "Test Database",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: pgConfig,
	}

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusCreated,
		&createdDatabase,
	)

	repository := &DatabaseRepository{}
	databaseFromDB, err := repository.FindByID(createdDatabase.ID)
	assert.NoError(t, err)
	assert.NotNil(t, databaseFromDB)
	assert.NotNil(t, databaseFromDB.PostgresqlLogical)

	assert.True(
		t,
		strings.HasPrefix(databaseFromDB.PostgresqlLogical.Password, "enc:"),
		"Password should be encrypted in database with 'enc:' prefix, got: %s",
		databaseFromDB.PostgresqlLogical.Password,
	)

	encryptor := encryption.GetFieldEncryptor()
	decryptedPassword, err := encryptor.Decrypt(databaseFromDB.PostgresqlLogical.Password)
	assert.NoError(t, err)
	assert.Equal(t, plainPassword, decryptedPassword,
		"Decrypted password should match original plaintext password")

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+createdDatabase.ID.String(),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_DatabaseSensitiveDataLifecycle_AllTypes(t *testing.T) {
	testCases := []struct {
		name                string
		databaseType        DatabaseType
		createDatabase      func(t *testing.T, workspaceID uuid.UUID) *Database
		updateDatabase      func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database
		verifySensitiveData func(t *testing.T, database *Database)
		verifyHiddenData    func(t *testing.T, database *Database)
	}{
		{
			name:         "PostgreSQL Database",
			databaseType: DatabaseTypePostgresLogical,
			createDatabase: func(_ *testing.T, workspaceID uuid.UUID) *Database {
				pgConfig := getTestPostgresConfig()
				return &Database{
					WorkspaceID:       &workspaceID,
					Name:              "Test PostgreSQL Database",
					Type:              DatabaseTypePostgresLogical,
					PostgresqlLogical: pgConfig,
				}
			},
			updateDatabase: func(_ *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				pgConfig := getTestPostgresConfig()
				pgConfig.Password = ""
				return &Database{
					ID:                databaseID,
					WorkspaceID:       &workspaceID,
					Name:              "Updated PostgreSQL Database",
					Type:              DatabaseTypePostgresLogical,
					PostgresqlLogical: pgConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.PostgresqlLogical.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.PostgresqlLogical.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.PostgresqlLogical.Password)
			},
		},
		{
			name:         "MariaDB Database",
			databaseType: DatabaseTypeMariadb,
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) *Database {
				mariaConfig := getTestMariadbConfig(t)
				return &Database{
					WorkspaceID: &workspaceID,
					Name:        "Test MariaDB Database",
					Type:        DatabaseTypeMariadb,
					Mariadb:     mariaConfig,
				}
			},
			updateDatabase: func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				mariaConfig := getTestMariadbConfig(t)
				mariaConfig.Password = ""
				return &Database{
					ID:          databaseID,
					WorkspaceID: &workspaceID,
					Name:        "Updated MariaDB Database",
					Type:        DatabaseTypeMariadb,
					Mariadb:     mariaConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.Mariadb.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.Mariadb.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.Mariadb.Password)
			},
		},
		{
			name:         "MongoDB Database",
			databaseType: DatabaseTypeMongodb,
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) *Database {
				mongoConfig := getTestMongodbConfig(t)
				return &Database{
					WorkspaceID: &workspaceID,
					Name:        "Test MongoDB Database",
					Type:        DatabaseTypeMongodb,
					Mongodb:     mongoConfig,
				}
			},
			updateDatabase: func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				mongoConfig := getTestMongodbConfig(t)
				mongoConfig.Password = ""
				return &Database{
					ID:          databaseID,
					WorkspaceID: &workspaceID,
					Name:        "Updated MongoDB Database",
					Type:        DatabaseTypeMongodb,
					Mongodb:     mongoConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.Mongodb.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.Mongodb.Password)
				assert.NoError(t, err)
				assert.Equal(t, "rootpassword", decrypted)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.Mongodb.Password)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			// Phase 1: Create database with sensitive data
			initialDatabase := tc.createDatabase(t, workspace.ID)
			var createdDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+owner.Token,
				*initialDatabase,
				http.StatusCreated,
				&createdDatabase,
			)
			assert.NotEmpty(t, createdDatabase.ID)
			assert.Equal(t, initialDatabase.Name, createdDatabase.Name)

			// Phase 2: Read via service - sensitive data should be hidden
			var retrievedDatabase Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&retrievedDatabase,
			)
			tc.verifyHiddenData(t, &retrievedDatabase)
			assert.Equal(t, initialDatabase.Name, retrievedDatabase.Name)

			// Phase 3: Update with non-sensitive changes only (sensitive fields empty)
			updatedDatabase := tc.updateDatabase(t, workspace.ID, createdDatabase.ID)
			var updateResponse Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+owner.Token,
				*updatedDatabase,
				http.StatusOK,
				&updateResponse,
			)

			// Phase 4: Retrieve directly from repository to verify sensitive data preservation
			repository := &DatabaseRepository{}
			databaseFromDB, err := repository.FindByID(createdDatabase.ID)
			assert.NoError(t, err)

			// Verify original sensitive data is still present in DB
			tc.verifySensitiveData(t, databaseFromDB)

			// Verify non-sensitive fields were updated in DB
			assert.Equal(t, updatedDatabase.Name, databaseFromDB.Name)

			// Phase 5: Additional verification - Check via GET that data is still hidden
			var finalRetrieved Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&finalRetrieved,
			)
			tc.verifyHiddenData(t, &finalRetrieved)

			// Phase 6: Verify GetDatabasesByWorkspace also hides sensitive data
			var workspaceDatabases []Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases?workspace_id=%s", workspace.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&workspaceDatabases,
			)
			var foundDatabase *Database
			for i := range workspaceDatabases {
				if workspaceDatabases[i].ID == createdDatabase.ID {
					foundDatabase = &workspaceDatabases[i]
					break
				}
			}
			assert.NotNil(t, foundDatabase, "Database should be found in workspace databases list")
			tc.verifyHiddenData(t, foundDatabase)

			// Clean up: Delete database before removing workspace
			test_utils.MakeDeleteRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusNoContent,
			)

			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}

func Test_TestConnection_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name                    string
		isMember                bool
		isGlobalAdmin           bool
		expectAccessGranted     bool
		expectedStatusCodeOnErr int
	}{
		{
			name:                    "workspace member can test connection",
			isMember:                true,
			isGlobalAdmin:           false,
			expectAccessGranted:     true,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
		{
			name:                    "non-member cannot test connection",
			isMember:                false,
			isGlobalAdmin:           false,
			expectAccessGranted:     false,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
		{
			name:                    "global admin can test connection",
			isMember:                false,
			isGlobalAdmin:           true,
			expectAccessGranted:     true,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(database)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.isMember {
				testUser = owner.Token
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			w := workspaces_testing.MakeAPIRequest(
				router,
				"POST",
				"/api/v1/databases/"+database.ID.String()+"/test-connection",
				"Bearer "+testUser,
				nil,
			)

			body := w.Body.String()

			if tt.expectAccessGranted {
				assert.True(
					t,
					w.Code == http.StatusOK ||
						(w.Code == http.StatusBadRequest && strings.Contains(body, "connect")),
					"Expected 200 OK or 400 with connection error, got %d: %s",
					w.Code,
					body,
				)
			} else {
				assert.Equal(t, tt.expectedStatusCodeOnErr, w.Code)
				assert.Contains(t, body, "insufficient permissions")
			}
		})
	}
}

func createTestDatabaseViaAPI(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *Database {
	env := config.GetEnv()
	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	testDbName := "testdb"
	request := Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        DatabaseTypePostgresLogical,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     config.GetEnv().TestLocalhost,
			Port:     port,
			Username: "testuser",
			Password: "testpassword",
			Database: &testDbName,
			CpuCount: 1,
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
		panic(
			fmt.Sprintf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String()),
		)
	}

	var database Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected.(*gin.RouterGroup))
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetDatabaseController().RegisterRoutes(protected.(*gin.RouterGroup))

	GetDatabaseController().RegisterPublicRoutes(v1)

	audit_logs.SetupDependencies()

	return router
}

func getTestPostgresConfig() *postgresql_logical.PostgresqlLogicalDatabase {
	env := config.GetEnv()
	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	testDbName := "testdb"
	return &postgresql_logical.PostgresqlLogicalDatabase{
		Version:  tools.PostgresqlVersion16,
		Host:     config.GetEnv().TestLocalhost,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &testDbName,
		CpuCount: 1,
	}
}

func Test_CreateDatabase_WhenUserIsNotReadOnly_DatabaseCreated(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Non-Cloud", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(workspace, router)

	request := Database{
		Name:              "Non-Cloud DB",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	var response Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusCreated,
		&response,
	)
	defer RemoveTestDatabase(&response)

	assert.Equal(t, "Non-Cloud DB", response.Name)
	assert.NotEqual(t, uuid.Nil, response.ID)
}

func getTestMariadbConfig(t *testing.T) *mariadb.MariadbDatabase {
	endpoint := containers.StartMariadb(t, "mariadb:10.11")

	return GetTestMariadbConfig(endpoint.Host, endpoint.Port)
}

func getTestMongodbConfig(t *testing.T) *mongodb.MongodbDatabase {
	endpoint := containers.StartMongodb(t, "mongo:7.0")

	return GetTestMongodbConfig(endpoint.Host, endpoint.Port)
}

// physicalNoSummaryVersion pairs a version tag with its image for the summarize_wal=off tests,
// which boot a throwaway no-summary source per version via containers.StartPhysicalPostgres.
type physicalNoSummaryVersion struct {
	tag   string
	image string
}

var physicalNoSummaryVersions = []physicalNoSummaryVersion{
	{"17", "postgres:17"},
	{"18", "postgres:18"},
}

func Test_CreateDatabase_FailsForPhysicalIncrementalWhenSummarizeWalOff(t *testing.T) {
	incrementalBackupTypes := []postgresql_physical.BackupType{
		postgresql_physical.BackupTypeFullAndIncremental,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
	}

	for _, dbVersion := range physicalNoSummaryVersions {
		for _, backupType := range incrementalBackupTypes {
			t.Run(fmt.Sprintf("pg%s_%s", dbVersion.tag, backupType), func(t *testing.T) {
				router := createTestRouter()
				owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
				defer workspaces_testing.RemoveTestWorkspace(workspace, router)

				source := containers.StartPhysicalPostgres(t, dbVersion.image, containers.WithoutSummarizer())
				physicalConfig := GetTestPhysicalPostgresConfigNoSummary(source.Host, source.Port, dbVersion.tag)
				physicalConfig.BackupType = backupType

				request := Database{
					Name:               "Physical Incremental NoSummary",
					WorkspaceID:        &workspace.ID,
					Type:               DatabaseTypePostgresPhysical,
					PostgresqlPhysical: physicalConfig,
				}

				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/databases/create",
					"Bearer "+owner.Token,
					request,
					http.StatusBadRequest,
				)

				assert.Contains(t, string(resp.Body), string(postgresql_shared.ConnErrWalSummaryDisabled))
			})
		}
	}
}

func Test_UpdateDatabase_FailsForSwitchToPhysicalIncrementalWhenSummarizeWalOff(t *testing.T) {
	for _, dbVersion := range physicalNoSummaryVersions {
		t.Run("pg"+dbVersion.tag, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(workspace, router)

			source := containers.StartPhysicalPostgres(t, dbVersion.image, containers.WithoutSummarizer())
			createRequest := Database{
				Name:               "Physical Full Only NoSummary",
				WorkspaceID:        &workspace.ID,
				Type:               DatabaseTypePostgresPhysical,
				PostgresqlPhysical: GetTestPhysicalPostgresConfigNoSummary(source.Host, source.Port, dbVersion.tag),
			}

			var createdDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+owner.Token,
				createRequest,
				http.StatusCreated,
				&createdDatabase,
			)
			defer RemoveTestDatabase(&createdDatabase)

			createdDatabase.PostgresqlPhysical.BackupType = postgresql_physical.BackupTypeFullAndIncremental

			updateResponse := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+owner.Token,
				createdDatabase,
				http.StatusBadRequest,
			)

			assert.Contains(t, string(updateResponse.Body), string(postgresql_shared.ConnErrWalSummaryDisabled))

			var refetchedDatabase Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+createdDatabase.ID.String(),
				"Bearer "+owner.Token,
				http.StatusOK,
				&refetchedDatabase,
			)

			assert.Equal(
				t,
				postgresql_physical.BackupTypeFullOnly,
				refetchedDatabase.PostgresqlPhysical.BackupType,
			)
		})
	}
}
