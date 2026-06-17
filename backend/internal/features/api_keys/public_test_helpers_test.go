package api_keys

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backuping "databasus-backend/internal/features/backups/backups/backuping"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/postgresql"
	"databasus-backend/internal/features/storages"
	local_storage "databasus-backend/internal/features/storages/models/local"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/tools"
)

func createTestDatabaseNoStorage(
	t *testing.T,
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) *databases.Database {
	t.Helper()

	testDbName := "testdb"
	env := config.GetEnv()
	port, parseErr := strconv.Atoi(env.TestPostgres16Port)
	if parseErr != nil {
		t.Fatalf("failed to parse test postgres port: %v", parseErr)
	}

	createRequest := databases.Database{
		Name:        "API Key Test DB",
		WorkspaceID: &workspace.ID,
		Type:        databases.DatabaseTypePostgres,
		Postgresql: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env.TestLocalhost,
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
		"Bearer "+owner.Token,
		createRequest,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create database: %d %s", w.Code, w.Body.String())
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		t.Fatalf("failed to decode database: %v", err)
	}
	t.Cleanup(func() { databases.RemoveTestDatabase(&database) })

	return &database
}

func createDatabaseWithEnabledBackups(
	t *testing.T,
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*databases.Database, *storages.Storage) {
	t.Helper()

	storage := &storages.Storage{
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "API Key Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}
	savedStorage, err := (&storages.StorageRepository{}).Save(storage)
	if err != nil {
		t.Fatalf("failed to save storage: %v", err)
	}
	t.Cleanup(func() { storages.RemoveTestStorage(savedStorage.ID) })

	database := createTestDatabaseNoStorage(t, workspace, owner, router)

	configService := backups_config.GetBackupConfigService()
	backupConfig, err := configService.GetBackupConfigByDbId(database.ID)
	if err != nil {
		t.Fatalf("failed to get backup config: %v", err)
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorageID = &savedStorage.ID
	backupConfig.Storage = savedStorage
	if _, err := configService.SaveBackupConfig(backupConfig); err != nil {
		t.Fatalf("failed to save backup config: %v", err)
	}

	return database, savedStorage
}

func insertInProgressBackupForDatabase(t *testing.T, databaseID, storageID uuid.UUID) *backups_core.Backup {
	t.Helper()

	backup := &backups_core.Backup{
		ID:         uuid.New(),
		DatabaseID: databaseID,
		StorageID:  storageID,
		Status:     backups_core.BackupStatusInProgress,
		CreatedAt:  time.Now().UTC(),
	}
	backup.GenerateFilename("apikey-timeout")
	if err := backups_core.GetBackupRepository().Save(backup); err != nil {
		t.Fatalf("insert in-progress backup: %v", err)
	}
	t.Cleanup(func() { _ = backups_core.GetBackupRepository().DeleteByID(backup.ID) })

	return backup
}

func startBackupMachineryForTest(t *testing.T) func() {
	t.Helper()

	backuperNode := backuping.CreateTestBackuperNode()
	backuperCancel := backuping.StartBackuperNodeForTest(t, backuperNode)

	scheduler := backuping.CreateTestScheduler(nil)
	schedulerCancel := backuping.StartSchedulerForTest(t, scheduler)

	return func() {
		schedulerCancel()
		backuping.StopBackuperNodeForTest(t, backuperCancel, backuperNode)
	}
}

func createAdminApiKeyToken(t *testing.T) string {
	t.Helper()

	creator := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	plain, hashed, prefix, err := generateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	apiKey := &ApiKey{
		Name:            "Public Admin Key",
		HashedToken:     hashed,
		TokenPrefix:     prefix,
		Role:            users_enums.UserRoleAdmin,
		CreatedByUserID: creator.UserID,
		CreatedAt:       time.Now().UTC(),
	}
	if err := (&ApiKeyRepository{}).Create(apiKey, nil); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(apiKey.ID) })

	return plain
}

func createMemberApiKeyToken(t *testing.T, grantedWorkspaceID uuid.UUID) string {
	t.Helper()

	creator := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	plain, hashed, prefix, err := generateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	apiKey := &ApiKey{
		Name:            "Public Member Key",
		HashedToken:     hashed,
		TokenPrefix:     prefix,
		Role:            users_enums.UserRoleMember,
		CreatedByUserID: creator.UserID,
		CreatedAt:       time.Now().UTC(),
	}
	if err := (&ApiKeyRepository{}).Create(apiKey, []uuid.UUID{grantedWorkspaceID}); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(apiKey.ID) })

	return plain
}
