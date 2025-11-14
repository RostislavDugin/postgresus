package restores

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
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/restores/models"
	"postgresus-backend/internal/features/storages"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	users_dto "postgresus-backend/internal/features/users/dto"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
	workspaces_models "postgresus-backend/internal/features/workspaces/models"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	test_utils "postgresus-backend/internal/util/testing"
	"postgresus-backend/internal/util/tools"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config.GetBackupConfigController(),
		backups.GetBackupController(),
		GetRestoreController(),
	)
	return router
}

func Test_GetRestores_WhenUserIsWorkspaceMember_RestoresReturned(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	var restores []*models.Restore
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&restores,
	)

	assert.NotNil(t, restores)
	assert.Equal(t, 0, len(restores))
	assert.NotNil(t, database)
}

func Test_GetRestores_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	_, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+nonMember.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_GetRestores_WhenUserIsGlobalAdmin_RestoresReturned(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	_, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	var restores []*models.Restore
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&restores,
	)

	assert.NotNil(t, restores)
}

func Test_RestoreBackup_WhenUserIsWorkspaceMember_RestoreInitiated(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	_, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	request := RestoreBackupRequest{
		PostgresqlDatabase: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "restore started successfully")
}

func Test_RestoreBackup_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	_, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := RestoreBackupRequest{
		PostgresqlDatabase: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_RestoreBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	database, backup := createTestDatabaseWithBackupForRestore(workspace, owner, router)

	request := RestoreBackupRequest{
		PostgresqlDatabase: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
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
		if strings.Contains(log.Message, "Database restored from backup") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for restore not found")
}

func createTestDatabaseWithBackupForRestore(
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*databases.Database, *backups.Backup) {
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

func createTestDatabase(
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

func createTestBackup(
	database *databases.Database,
	owner *users_dto.SignInResponseDTO,
) *backups.Backup {
	userService := users_services.GetUserService()
	user, err := userService.GetUserFromToken(owner.Token)
	if err != nil {
		panic(err)
	}

	storages, err := storages.GetStorageService().GetStorages(user, *database.WorkspaceID)
	if err != nil || len(storages) == 0 {
		panic("No storage found for workspace")
	}

	backup := &backups.Backup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		StorageID:        storages[0].ID,
		Status:           backups.BackupStatusCompleted,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &backups.BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	dummyContent := []byte("dummy backup content for testing")
	reader := strings.NewReader(string(dummyContent))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := storages[0].SaveFile(logger, backup.ID, reader); err != nil {
		panic(fmt.Sprintf("Failed to create test backup file: %v", err))
	}

	return backup
}
