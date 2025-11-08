package backups

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/storages"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	users_dto "postgresus-backend/internal/features/users/dto"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_models "postgresus-backend/internal/features/workspaces/models"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	test_utils "postgresus-backend/internal/util/testing"
	"postgresus-backend/internal/util/tools"
)

func Test_GetBackups_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace viewer can get backups",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get backups",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get backups",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can get backups",
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

			database, _ := createTestDatabaseWithBackups(workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(users_enums.UserRoleMember)
					workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakeGetRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups?database_id=%s", database.ID.String()),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				var response GetBackupsResponse
				err := json.Unmarshal(testResp.Body, &response)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(response.Backups), 1)
				assert.GreaterOrEqual(t, response.Total, int64(1))
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CreateBackup_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot create backup",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can create backup",
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

			database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
			enableBackupForDatabase(database.ID)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(users_enums.UserRoleMember)
					workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			request := MakeBackupRequest{DatabaseID: database.ID}
			testResp := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/backups",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "backup started successfully")
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CreateBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	enableBackupForDatabase(database.ID)

	request := MakeBackupRequest{DatabaseID: database.ID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup manually initiated") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup creation not found")
}

func Test_DeleteBackup_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace member can delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace viewer cannot delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "non-member cannot delete backup",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can delete backup",
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

			database, backup := createTestDatabaseWithBackups(workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(users_enums.UserRoleMember)
					workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakeDeleteRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups/%s", backup.ID.String()),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if !tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			} else {
				userService := users_services.GetUserService()
				ownerUser, err := userService.GetUserFromToken(owner.Token)
				assert.NoError(t, err)

				response, err := GetBackupService().GetBackups(ownerUser, database.ID, 10, 0)
				assert.NoError(t, err)
				assert.Equal(t, 0, len(response.Backups))
			}
		})
	}
}

func Test_DeleteBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database, backup := createTestDatabaseWithBackups(workspace, owner, router)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup deleted") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup deletion not found")
}

func Test_DownloadBackup_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace viewer can download backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can download backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot download backup",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can download backup",
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

			_, backup := createTestDatabaseWithBackups(workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(users_enums.UserRoleMember)
					workspaces_testing.AddMemberToWorkspace(workspace, member, *tt.workspaceRole, owner.Token, router)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakeGetRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups/%s/file", backup.ID.String()),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if !tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_DownloadBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database, backup := createTestDatabaseWithBackups(workspace, owner, router)

	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup file downloaded") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup download not found")
}

func Test_CancelBackup_InProgressBackup_SuccessfullyCancelled(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	config.IsBackupsEnabled = true
	config.StorageID = &storage.ID
	config.Storage = storage
	_, err = configService.SaveBackupConfig(config)
	assert.NoError(t, err)

	backup := &Backup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		Database:         database,
		StorageID:        storage.ID,
		Storage:          storage,
		Status:           BackupStatusInProgress,
		BackupSizeMb:     0,
		BackupDurationMs: 0,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &BackupRepository{}
	err = repo.Save(backup)
	assert.NoError(t, err)

	// Register a cancellable context for the backup
	GetBackupService().backupContextMgr.RegisterBackup(backup.ID, func() {})

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/cancel", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusNoContent,
	)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify audit log was created
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	userService := users_services.GetUserService()
	adminUser, err := userService.GetUserFromToken(admin.Token)
	assert.NoError(t, err)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetGlobalAuditLogs(
		adminUser,
		&audit_logs.GetAuditLogsRequest{Limit: 100, Offset: 0},
	)
	assert.NoError(t, err)

	foundCancelLog := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup cancelled") &&
			strings.Contains(log.Message, database.Name) {
			foundCancelLog = true
			break
		}
	}
	assert.True(t, foundCancelLog, "Cancel audit log should be created")
}

func createTestRouter() *gin.Engine {
	return CreateTestRouter()
}

func createTestDatabase(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	testDbName := "test_db"
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
			Database: &testDbName,
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

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func createTestStorage(workspaceID uuid.UUID) *storages.Storage {
	storage := &storages.Storage{
		WorkspaceID:  workspaceID,
		Type:         storages.StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}

	repo := &storages.StorageRepository{}
	storage, err := repo.Save(storage)
	if err != nil {
		panic(err)
	}

	return storage
}

func enableBackupForDatabase(databaseID uuid.UUID) {
	configService := backups_config.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		panic(err)
	}

	config.IsBackupsEnabled = true
	_, err = configService.SaveBackupConfig(config)
	if err != nil {
		panic(err)
	}
}

func createTestDatabaseWithBackups(
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*databases.Database, *Backup) {
	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	if err != nil {
		panic(err)
	}

	config.IsBackupsEnabled = true
	config.StorageID = &storage.ID
	config.Storage = storage
	_, err = configService.SaveBackupConfig(config)
	if err != nil {
		panic(err)
	}

	backup := createTestBackup(database, owner)

	return database, backup
}

func createTestBackup(
	database *databases.Database,
	owner *users_dto.SignInResponseDTO,
) *Backup {
	userService := users_services.GetUserService()
	user, err := userService.GetUserFromToken(owner.Token)
	if err != nil {
		panic(err)
	}

	storages, err := storages.GetStorageService().GetStorages(user, *database.WorkspaceID)
	if err != nil || len(storages) == 0 {
		panic("No storage found for workspace")
	}

	backup := &Backup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		Database:         database,
		StorageID:        storages[0].ID,
		Storage:          storages[0],
		Status:           BackupStatusCompleted,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	// Create a dummy backup file for testing download functionality
	dummyContent := []byte("dummy backup content for testing")
	reader := strings.NewReader(string(dummyContent))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := storages[0].SaveFile(logger, backup.ID, reader); err != nil {
		panic(fmt.Sprintf("Failed to create test backup file: %v", err))
	}

	return backup
}
