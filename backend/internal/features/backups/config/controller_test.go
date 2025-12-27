package backups_config

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/postgresql"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/period"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetBackupConfigController(),
	)
	return router
}

func Test_SaveBackupConfig_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can save backup config",
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

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
				testUserToken = member.Token
			}

			timeOfDay := "04:00"
			request := BackupConfig{
				DatabaseID:       database.ID,
				IsBackupsEnabled: true,
				StorePeriod:      period.PeriodWeek,
				BackupInterval: &intervals.Interval{
					Interval:  intervals.IntervalDaily,
					TimeOfDay: &timeOfDay,
				},
				SendNotificationsOn: []BackupNotificationType{
					NotificationBackupFailed,
				},
				IsRetryIfFailed:     true,
				MaxFailedTriesCount: 3,
			}

			var response BackupConfig
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/save",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.True(t, response.IsBackupsEnabled)
				assert.Equal(t, period.PeriodWeek, response.StorePeriod)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_SaveBackupConfig_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	timeOfDay := "04:00"
	request := BackupConfig{
		DatabaseID:       database.ID,
		IsBackupsEnabled: true,
		StorePeriod:      period.PeriodWeek,
		BackupInterval: &intervals.Interval{
			Interval:  intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_GetBackupConfigByDbID_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can get backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			var response BackupConfig
			testResp := test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/database/"+database.ID.String(),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.NotNil(t, response.BackupInterval)
			} else {
				assert.Contains(t, string(testResp.Body), "backup configuration not found")
			}
		})
	}
}

func Test_GetBackupConfigByDbID_ReturnsDefaultConfigForNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	var response BackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.False(t, response.IsBackupsEnabled)
	assert.Equal(t, period.PeriodWeek, response.StorePeriod)
	assert.True(t, response.IsRetryIfFailed)
	assert.Equal(t, 3, response.MaxFailedTriesCount)
	assert.NotNil(t, response.BackupInterval)
}

func Test_IsStorageUsing_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		isStorageOwner     bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "storage owner can check storage usage",
			isStorageOwner:     true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-storage-owner cannot check storage usage",
			isStorageOwner:     false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			storageOwner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(
				"Test Workspace",
				storageOwner,
				router,
			)
			storage := createTestStorage(workspace.ID)

			var testUserToken string
			if tt.isStorageOwner {
				testUserToken = storageOwner.Token
			} else {
				otherUser := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = otherUser.Token
			}

			if tt.expectSuccess {
				var response map[string]bool
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/backup-configs/storage/"+storage.ID.String()+"/is-using",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				isUsing, exists := response["isUsing"]
				assert.True(t, exists)
				assert.False(t, isUsing)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/backup-configs/storage/"+storage.ID.String()+"/is-using",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "error")
			}

			// Cleanup
			storages.RemoveTestStorage(storage.ID)
			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}

func Test_SaveBackupConfig_WithEncryptionNone_ConfigSaved(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	timeOfDay := "04:00"
	request := BackupConfig{
		DatabaseID:       database.ID,
		IsBackupsEnabled: true,
		StorePeriod:      period.PeriodWeek,
		BackupInterval: &intervals.Interval{
			Interval:  intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          BackupEncryptionNone,
	}

	var response BackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.Equal(t, BackupEncryptionNone, response.Encryption)
}

func Test_SaveBackupConfig_WithEncryptionEncrypted_ConfigSaved(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	timeOfDay := "04:00"
	request := BackupConfig{
		DatabaseID:       database.ID,
		IsBackupsEnabled: true,
		StorePeriod:      period.PeriodWeek,
		BackupInterval: &intervals.Interval{
			Interval:  intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          BackupEncryptionEncrypted,
	}

	var response BackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.Equal(t, BackupEncryptionEncrypted, response.Encryption)
}

func createTestDatabaseViaAPI(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	testDbName := "test_db"
	request := databases.Database{
		WorkspaceID: &workspaceID,
		Name:        name,
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
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
		panic("Failed to create database")
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}
	return &database
}

func createTestStorage(workspaceID uuid.UUID) *storages.Storage {
	return storages.CreateTestStorage(workspaceID)
}
