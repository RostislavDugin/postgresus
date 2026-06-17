package api_keys

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_CreateApiKey_WhenUserIsAdmin_ReturnsTokenOnce(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := CreateApiKeyRequestDTO{Name: "CI Key", Role: users_enums.UserRoleAdmin}
	var response CreateApiKeyResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, request, http.StatusOK, &response,
	)
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(response.ID) })

	assert.Equal(t, "CI Key", response.Name)
	assert.NotEmpty(t, response.Token)
	assert.Contains(t, response.Token, TokenPrefix)
}

func Test_CreateApiKey_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter()
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := CreateApiKeyRequestDTO{Name: "Nope", Role: users_enums.UserRoleAdmin}
	var errorResponse map[string]string
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+member.Token, request, http.StatusForbidden, &errorResponse,
	)

	assert.Equal(t, ErrAdminOnly.Error(), errorResponse["error"])
}

func Test_CreateApiKey_WhenMemberRoleWithoutWorkspaces_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := CreateApiKeyRequestDTO{Name: "No grants", Role: users_enums.UserRoleMember}
	var errorResponse map[string]string
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, request, http.StatusBadRequest, &errorResponse,
	)

	assert.Equal(t, ErrWorkspacesRequired.Error(), errorResponse["error"])
}

func Test_ListAndRevokeApiKey_WhenUserIsAdmin_Works(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	createRequest := CreateApiKeyRequestDTO{Name: "Listable", Role: users_enums.UserRoleAdmin}
	var created CreateApiKeyResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, createRequest, http.StatusOK, &created,
	)
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(created.ID) })

	var list ListApiKeysResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, http.StatusOK, &list,
	)
	require.NotEmpty(t, list.ApiKeys)

	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/api-keys/"+created.ID.String(), "Bearer "+admin.Token, http.StatusOK,
	)

	revoked, err := (&ApiKeyRepository{}).FindByHashedToken(hashToken(created.Token))
	require.NoError(t, err)
	assert.NotNil(t, revoked.RevokedAt)
}

func Test_RevokeApiKey_WhenIdUnknown_ReturnsNotFound(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/api-keys/"+uuid.New().String(), "Bearer "+admin.Token, http.StatusNotFound,
	)
}

func Test_PublicTriggerBackup_WhenKeyValidAndBackupCompletes_Returns200(t *testing.T) {
	router := CreateApiKeyPublicTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace("API Key WS", admin, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(workspace, router) })

	database, _ := createDatabaseWithEnabledBackups(t, workspace, admin, router)

	token := createAdminApiKeyToken(t)

	stopMachinery := startBackupMachineryForTest(t)
	defer stopMachinery()

	request := TriggerBackupRequestDTO{DatabaseID: database.ID}
	var response TriggerBackupResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/public/backups", "Bearer "+token, request, http.StatusOK, &response,
	)

	assert.Equal(t, backups_core.BackupStatusCompleted, response.Status)
}

func Test_PublicTriggerBackup_WhenBackupStaysInProgress_Returns202(t *testing.T) {
	router := CreateApiKeyPublicTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace("API Key WS Timeout", admin, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(workspace, router) })

	database, storage := createDatabaseWithEnabledBackups(t, workspace, admin, router)

	inProgress := insertInProgressBackupForDatabase(t, database.ID, storage.ID)

	originalTimeout := config.GetEnv().ApiBackupSyncTimeout
	config.GetEnv().ApiBackupSyncTimeout = 1 * time.Second
	t.Cleanup(func() { config.GetEnv().ApiBackupSyncTimeout = originalTimeout })

	token := createAdminApiKeyToken(t)

	request := TriggerBackupRequestDTO{DatabaseID: database.ID}
	var response TriggerBackupResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/public/backups", "Bearer "+token, request, http.StatusAccepted, &response,
	)

	assert.Equal(t, inProgress.ID, response.BackupID)
	assert.Equal(t, backups_core.BackupStatusInProgress, response.Status)
}

func Test_PublicTriggerBackup_WhenTokenInvalid_Returns401(t *testing.T) {
	router := CreateApiKeyPublicTestRouter()

	request := TriggerBackupRequestDTO{DatabaseID: uuid.New()}
	test_utils.MakePostRequest(
		t, router, "/api/v1/public/backups", "Bearer dbs_invalid", request, http.StatusUnauthorized,
	)
}

func Test_PublicTriggerBackup_WhenMemberKeyLacksGrant_Returns403(t *testing.T) {
	router := CreateApiKeyPublicTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace("Ungranted WS", admin, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(workspace, router) })

	database, _ := createDatabaseWithEnabledBackups(t, workspace, admin, router)

	otherWorkspace, err := workspaces_testing.CreateTestWorkspaceDirect("Other WS "+uuid.New().String(), admin.UserID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = workspaces_testing.RemoveTestWorkspaceDirect(otherWorkspace.ID) })
	token := createMemberApiKeyToken(t, otherWorkspace.ID)

	request := TriggerBackupRequestDTO{DatabaseID: database.ID}
	test_utils.MakePostRequest(
		t, router, "/api/v1/public/backups", "Bearer "+token, request, http.StatusForbidden,
	)
}
