package storages

import (
	"fmt"
	"net/http"
	"testing"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	test_utils "postgresus-backend/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SaveNewStorage_StorageReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	verifyStorageData(t, storage, &savedStorage)
	assert.NotEmpty(t, savedStorage.ID)

	// Verify storage is returned via GET
	var retrievedStorage Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedStorage,
	)

	verifyStorageData(t, &savedStorage, &retrievedStorage)

	// Verify storage is returned via GET all storages
	var storages []Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspace.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&storages,
	)

	assert.Contains(t, storages, savedStorage)

	deleteStorage(t, router, savedStorage.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UpdateExistingStorage_UpdatedStorageReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	updatedName := "Updated Storage " + uuid.New().String()
	savedStorage.Name = updatedName

	var updatedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		savedStorage,
		http.StatusOK,
		&updatedStorage,
	)

	assert.Equal(t, updatedName, updatedStorage.Name)
	assert.Equal(t, savedStorage.ID, updatedStorage.ID)

	deleteStorage(t, router, updatedStorage.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_DeleteStorage_StorageNotReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
	)

	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "error")
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_TestDirectStorageConnection_ConnectionEstablished(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)
	response := test_utils.MakePostRequest(
		t, router, "/api/v1/storages/direct-test", "Bearer "+owner.Token, *storage, http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_TestExistingStorageConnection_ConnectionEstablished(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	response := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s/test", savedStorage.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")

	deleteStorage(t, router, savedStorage.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_ViewerCanViewStorages_ButCannotModify(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	viewer := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		viewer,
		users_enums.WorkspaceRoleViewer,
		owner.Token,
		router,
	)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	// Viewer can GET storages
	var storages []Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspace.ID.String()),
		"Bearer "+viewer.Token,
		http.StatusOK,
		&storages,
	)
	assert.Len(t, storages, 1)

	// Viewer cannot CREATE storage
	newStorage := createNewStorage(workspace.ID)
	test_utils.MakePostRequest(
		t, router, "/api/v1/storages", "Bearer "+viewer.Token, *newStorage, http.StatusForbidden,
	)

	// Viewer cannot UPDATE storage
	savedStorage.Name = "Updated by viewer"
	test_utils.MakePostRequest(
		t, router, "/api/v1/storages", "Bearer "+viewer.Token, savedStorage, http.StatusForbidden,
	)

	// Viewer cannot DELETE storage
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+viewer.Token,
		http.StatusForbidden,
	)

	deleteStorage(t, router, savedStorage.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_MemberCanManageStorages(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		member,
		users_enums.WorkspaceRoleMember,
		owner.Token,
		router,
	)
	storage := createNewStorage(workspace.ID)

	// Member can CREATE storage
	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+member.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)
	assert.NotEmpty(t, savedStorage.ID)

	// Member can UPDATE storage
	savedStorage.Name = "Updated by member"
	var updatedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+member.Token,
		savedStorage,
		http.StatusOK,
		&updatedStorage,
	)
	assert.Equal(t, "Updated by member", updatedStorage.Name)

	// Member can DELETE storage
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+member.Token,
		http.StatusOK,
	)

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_AdminCanManageStorages(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		admin,
		users_enums.WorkspaceRoleAdmin,
		owner.Token,
		router,
	)
	storage := createNewStorage(workspace.ID)

	// Admin can CREATE, UPDATE, DELETE
	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	savedStorage.Name = "Updated by admin"
	test_utils.MakePostRequest(
		t, router, "/api/v1/storages", "Bearer "+admin.Token, savedStorage, http.StatusOK,
	)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
	)

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UserNotInWorkspace_CannotAccessStorages(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	outsider := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	storage := createNewStorage(workspace.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	// Outsider cannot GET storages
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspace.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	// Outsider cannot CREATE storage
	test_utils.MakePostRequest(
		t, router, "/api/v1/storages", "Bearer "+outsider.Token, *storage, http.StatusForbidden,
	)

	// Outsider cannot UPDATE storage
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+outsider.Token,
		savedStorage,
		http.StatusForbidden,
	)

	// Outsider cannot DELETE storage
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	deleteStorage(t, router, savedStorage.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_CrossWorkspaceSecurity_CannotAccessStorageFromAnotherWorkspace(t *testing.T) {
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace1 := workspaces_testing.CreateTestWorkspace("Workspace 1", owner1, router)
	workspace2 := workspaces_testing.CreateTestWorkspace("Workspace 2", owner2, router)
	storage1 := createNewStorage(workspace1.ID)

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+owner1.Token,
		*storage1,
		http.StatusOK,
		&savedStorage,
	)

	// Try to access workspace1's storage with owner2 from workspace2
	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+owner2.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(response.Body), "insufficient permissions")

	deleteStorage(t, router, savedStorage.ID, workspace1.ID, owner1.Token)
	workspaces_testing.RemoveTestWorkspace(workspace1, router)
	workspaces_testing.RemoveTestWorkspace(workspace2, router)
}

func createRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		GetStorageController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	audit_logs.SetupDependencies()

	return router
}

func createNewStorage(workspaceID uuid.UUID) *Storage {
	return &Storage{
		WorkspaceID:  workspaceID,
		Type:         StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}
}

func verifyStorageData(t *testing.T, expected *Storage, actual *Storage) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
}

func deleteStorage(
	t *testing.T,
	router *gin.Engine,
	storageID, workspaceID uuid.UUID,
	token string,
) {
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", storageID.String()),
		"Bearer "+token,
		http.StatusOK,
	)
}
