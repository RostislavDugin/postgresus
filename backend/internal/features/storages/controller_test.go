package storages

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	audit_logs "databasus-backend/internal/features/audit_logs"
	azure_blob_storage "databasus-backend/internal/features/storages/models/azure_blob"
	ftp_storage "databasus-backend/internal/features/storages/models/ftp"
	local_storage "databasus-backend/internal/features/storages/models/local"
	nas_storage "databasus-backend/internal/features/storages/models/nas"
	rclone_storage "databasus-backend/internal/features/storages/models/rclone"
	s3_storage "databasus-backend/internal/features/storages/models/s3"
	sftp_storage "databasus-backend/internal/features/storages/models/sftp"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
)

type mockStorageDatabaseCounter struct{}

func (m *mockStorageDatabaseCounter) GetStorageAttachedDatabasesIDs(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

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

	deleteStorage(t, router, savedStorage.ID, owner.Token)
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

	deleteStorage(t, router, updatedStorage.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_CreateSystemStorage_OnlyAdminCanCreate_MemberGetsForbidden(t *testing.T) {
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", admin, router)

	// Admin can create system storage
	systemStorage := createNewStorage(workspace.ID)
	systemStorage.IsSystem = true

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemStorage,
		http.StatusOK,
		&savedStorage,
	)

	assert.True(t, savedStorage.IsSystem)
	assert.Equal(t, systemStorage.Name, savedStorage.Name)

	// Member cannot create system storage
	memberSystemStorage := createNewStorage(workspace.ID)
	memberSystemStorage.IsSystem = true

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+member.Token,
		*memberSystemStorage,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")

	deleteStorage(t, router, savedStorage.ID, admin.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UpdateStorageIsSystem_OnlyAdminCanUpdate_MemberGetsForbidden(t *testing.T) {
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", admin, router)

	// Create a regular storage
	storage := createNewStorage(workspace.ID)
	storage.IsSystem = false

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

	assert.False(t, savedStorage.IsSystem)

	// Admin can update to system
	savedStorage.IsSystem = true
	var updatedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		savedStorage,
		http.StatusOK,
		&updatedStorage,
	)

	assert.True(t, updatedStorage.IsSystem)

	// Member cannot update system storage
	updatedStorage.Name = "Updated by member"
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+member.Token,
		updatedStorage,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")

	deleteStorage(t, router, updatedStorage.ID, admin.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UpdateSystemStorage_CannotChangeToPrivate_ReturnsBadRequest(t *testing.T) {
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", admin, router)

	// Create system storage
	storage := createNewStorage(workspace.ID)
	storage.IsSystem = true

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

	assert.True(t, savedStorage.IsSystem)

	// Attempt to change system storage to non-system (should fail)
	savedStorage.IsSystem = false
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		savedStorage,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "system storage cannot be changed to non-system")

	// Verify storage is still system
	var retrievedStorage Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&retrievedStorage,
	)
	assert.True(t, retrievedStorage.IsSystem)

	// Admin can update other fields while keeping IsSystem=true
	savedStorage.IsSystem = true
	savedStorage.Name = "Updated System Storage"
	var updatedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		savedStorage,
		http.StatusOK,
		&updatedStorage,
	)
	assert.True(t, updatedStorage.IsSystem)
	assert.Equal(t, "Updated System Storage", updatedStorage.Name)

	deleteStorage(t, router, updatedStorage.ID, admin.Token)
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

	deleteStorage(t, router, savedStorage.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_WorkspaceRolePermissions(t *testing.T) {
	tests := []struct {
		name          string
		workspaceRole *users_enums.WorkspaceRole
		isGlobalAdmin bool
		canCreate     bool
		canUpdate     bool
		canDelete     bool
	}{
		{
			name:          "owner can manage storages",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "admin can manage storages",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "member can manage storages",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "viewer can view but cannot modify storages",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin: false,
			canCreate:     false,
			canUpdate:     false,
			canDelete:     false,
		},
		{
			name:          "global admin can manage storages",
			workspaceRole: nil,
			isGlobalAdmin: true,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createRouter()
			GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					testUser,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = testUser.Token
			}

			// Owner creates initial storage for all test cases
			var ownerStorage Storage
			storage := createNewStorage(workspace.ID)
			test_utils.MakePostRequestAndUnmarshal(
				t, router, "/api/v1/storages", "Bearer "+owner.Token,
				*storage, http.StatusOK, &ownerStorage,
			)

			// Test GET storages
			var storages []Storage
			test_utils.MakeGetRequestAndUnmarshal(
				t, router,
				fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspace.ID.String()),
				"Bearer "+testUserToken, http.StatusOK, &storages,
			)
			// Count only non-system storages for this workspace
			nonSystemStorages := 0
			for _, s := range storages {
				if !s.IsSystem {
					nonSystemStorages++
				}
			}
			assert.Equal(t, 1, nonSystemStorages)

			// Test CREATE storage
			createStatusCode := http.StatusOK
			if !tt.canCreate {
				createStatusCode = http.StatusForbidden
			}
			newStorage := createNewStorage(workspace.ID)
			var savedStorage Storage
			if tt.canCreate {
				test_utils.MakePostRequestAndUnmarshal(
					t, router, "/api/v1/storages", "Bearer "+testUserToken,
					*newStorage, createStatusCode, &savedStorage,
				)
				assert.NotEmpty(t, savedStorage.ID)
			} else {
				test_utils.MakePostRequest(
					t, router, "/api/v1/storages", "Bearer "+testUserToken,
					*newStorage, createStatusCode,
				)
			}

			// Test UPDATE storage
			updateStatusCode := http.StatusOK
			if !tt.canUpdate {
				updateStatusCode = http.StatusForbidden
			}
			ownerStorage.Name = "Updated by test user"
			if tt.canUpdate {
				var updatedStorage Storage
				test_utils.MakePostRequestAndUnmarshal(
					t, router, "/api/v1/storages", "Bearer "+testUserToken,
					ownerStorage, updateStatusCode, &updatedStorage,
				)
				assert.Equal(t, "Updated by test user", updatedStorage.Name)
			} else {
				test_utils.MakePostRequest(
					t, router, "/api/v1/storages", "Bearer "+testUserToken,
					ownerStorage, updateStatusCode,
				)
			}

			// Test DELETE storage
			deleteStatusCode := http.StatusOK
			if !tt.canDelete {
				deleteStatusCode = http.StatusForbidden
			}
			test_utils.MakeDeleteRequest(
				t, router,
				fmt.Sprintf("/api/v1/storages/%s", ownerStorage.ID.String()),
				"Bearer "+testUserToken, deleteStatusCode,
			)

			// Cleanup
			if tt.canCreate {
				deleteStorage(t, router, savedStorage.ID, owner.Token)
			}
			if !tt.canDelete {
				deleteStorage(t, router, ownerStorage.ID, owner.Token)
			}
			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}

func Test_SystemStorage_AdminOnlyOperations(t *testing.T) {
	tests := []struct {
		name           string
		operation      string
		isAdmin        bool
		expectSuccess  bool
		expectedStatus int
	}{
		{
			name:           "admin can create system storage",
			operation:      "create",
			isAdmin:        true,
			expectSuccess:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "member cannot create system storage",
			operation:      "create",
			isAdmin:        false,
			expectSuccess:  false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin can update storage to make it system",
			operation:      "update_to_system",
			isAdmin:        true,
			expectSuccess:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "member cannot update storage to make it system",
			operation:      "update_to_system",
			isAdmin:        false,
			expectSuccess:  false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin can update system storage",
			operation:      "update_system",
			isAdmin:        true,
			expectSuccess:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "member cannot update system storage",
			operation:      "update_system",
			isAdmin:        false,
			expectSuccess:  false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin can delete system storage",
			operation:      "delete",
			isAdmin:        true,
			expectSuccess:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "member cannot delete system storage",
			operation:      "delete",
			isAdmin:        false,
			expectSuccess:  false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createRouter()
			GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			var testUserToken string
			if tt.isAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else {
				member := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					users_enums.WorkspaceRoleMember,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			switch tt.operation {
			case "create":
				systemStorage := &Storage{
					WorkspaceID:  workspace.ID,
					Type:         StorageTypeLocal,
					Name:         "Test System Storage " + uuid.New().String(),
					IsSystem:     true,
					LocalStorage: &local_storage.LocalStorage{},
				}

				if tt.expectSuccess {
					var savedStorage Storage
					test_utils.MakePostRequestAndUnmarshal(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						*systemStorage,
						tt.expectedStatus,
						&savedStorage,
					)
					assert.NotEmpty(t, savedStorage.ID)
					assert.True(t, savedStorage.IsSystem)
					deleteStorage(t, router, savedStorage.ID, testUserToken)
				} else {
					resp := test_utils.MakePostRequest(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						*systemStorage,
						tt.expectedStatus,
					)
					assert.Contains(t, string(resp.Body), "insufficient permissions")
				}

			case "update_to_system":
				// Owner creates private storage first
				privateStorage := createNewStorage(workspace.ID)
				var savedStorage Storage
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/storages",
					"Bearer "+owner.Token,
					*privateStorage,
					http.StatusOK,
					&savedStorage,
				)

				// Test user attempts to make it system
				savedStorage.IsSystem = true
				if tt.expectSuccess {
					var updatedStorage Storage
					test_utils.MakePostRequestAndUnmarshal(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						savedStorage,
						tt.expectedStatus,
						&updatedStorage,
					)
					assert.True(t, updatedStorage.IsSystem)
					deleteStorage(t, router, savedStorage.ID, testUserToken)
				} else {
					resp := test_utils.MakePostRequest(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						savedStorage,
						tt.expectedStatus,
					)
					assert.Contains(t, string(resp.Body), "insufficient permissions")
					deleteStorage(t, router, savedStorage.ID, owner.Token)
				}

			case "update_system":
				// Admin creates system storage first
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				systemStorage := &Storage{
					WorkspaceID:  workspace.ID,
					Type:         StorageTypeLocal,
					Name:         "Test System Storage " + uuid.New().String(),
					IsSystem:     true,
					LocalStorage: &local_storage.LocalStorage{},
				}
				var savedStorage Storage
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/storages",
					"Bearer "+admin.Token,
					*systemStorage,
					http.StatusOK,
					&savedStorage,
				)

				// Test user attempts to update system storage
				savedStorage.Name = "Updated System Storage " + uuid.New().String()
				if tt.expectSuccess {
					var updatedStorage Storage
					test_utils.MakePostRequestAndUnmarshal(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						savedStorage,
						tt.expectedStatus,
						&updatedStorage,
					)
					assert.Equal(t, savedStorage.Name, updatedStorage.Name)
					assert.True(t, updatedStorage.IsSystem)
					deleteStorage(t, router, savedStorage.ID, testUserToken)
				} else {
					resp := test_utils.MakePostRequest(
						t,
						router,
						"/api/v1/storages",
						"Bearer "+testUserToken,
						savedStorage,
						tt.expectedStatus,
					)
					assert.Contains(t, string(resp.Body), "insufficient permissions")
					deleteStorage(t, router, savedStorage.ID, admin.Token)
				}

			case "delete":
				// Admin creates system storage first
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				systemStorage := &Storage{
					WorkspaceID:  workspace.ID,
					Type:         StorageTypeLocal,
					Name:         "Test System Storage " + uuid.New().String(),
					IsSystem:     true,
					LocalStorage: &local_storage.LocalStorage{},
				}
				var savedStorage Storage
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/storages",
					"Bearer "+admin.Token,
					*systemStorage,
					http.StatusOK,
					&savedStorage,
				)

				// Test user attempts to delete system storage
				if tt.expectSuccess {
					test_utils.MakeDeleteRequest(
						t,
						router,
						fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
						"Bearer "+testUserToken,
						tt.expectedStatus,
					)
				} else {
					resp := test_utils.MakeDeleteRequest(
						t,
						router,
						fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
						"Bearer "+testUserToken,
						tt.expectedStatus,
					)
					assert.Contains(t, string(resp.Body), "insufficient permissions")
					deleteStorage(t, router, savedStorage.ID, admin.Token)
				}
			}

			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}

func Test_GetStorages_SystemStorageIncludedForAllUsers(t *testing.T) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	// Create two workspaces with different owners
	ownerA := users_testing.CreateTestUser(users_enums.UserRoleMember)
	ownerB := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspaceA := workspaces_testing.CreateTestWorkspace("Workspace A", ownerA, router)
	workspaceB := workspaces_testing.CreateTestWorkspace("Workspace B", ownerB, router)

	// Create private storage in workspace A
	privateStorageA := createNewStorage(workspaceA.ID)
	var savedPrivateStorageA Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+ownerA.Token,
		*privateStorageA,
		http.StatusOK,
		&savedPrivateStorageA,
	)

	// Admin creates system storage in workspace B
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	systemStorageB := &Storage{
		WorkspaceID:  workspaceB.ID,
		Type:         StorageTypeLocal,
		Name:         "Test System Storage B " + uuid.New().String(),
		IsSystem:     true,
		LocalStorage: &local_storage.LocalStorage{},
	}
	var savedSystemStorageB Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemStorageB,
		http.StatusOK,
		&savedSystemStorageB,
	)

	// Test: User from workspace A should see both private storage A and system storage B
	var storagesForWorkspaceA []Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspaceA.ID.String()),
		"Bearer "+ownerA.Token,
		http.StatusOK,
		&storagesForWorkspaceA,
	)

	assert.GreaterOrEqual(t, len(storagesForWorkspaceA), 2)
	foundPrivateA := false
	foundSystemB := false
	for _, s := range storagesForWorkspaceA {
		if s.ID == savedPrivateStorageA.ID {
			foundPrivateA = true
		}
		if s.ID == savedSystemStorageB.ID {
			foundSystemB = true
		}
	}
	assert.True(t, foundPrivateA, "User from workspace A should see private storage A")
	assert.True(t, foundSystemB, "User from workspace A should see system storage B")

	// Test: User from workspace B should see system storage B
	var storagesForWorkspaceB []Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspaceB.ID.String()),
		"Bearer "+ownerB.Token,
		http.StatusOK,
		&storagesForWorkspaceB,
	)

	assert.GreaterOrEqual(t, len(storagesForWorkspaceB), 1)
	foundSystemBInWorkspaceB := false
	for _, s := range storagesForWorkspaceB {
		if s.ID == savedSystemStorageB.ID {
			foundSystemBInWorkspaceB = true
		}
		// Should NOT see private storage from workspace A
		assert.NotEqual(
			t,
			savedPrivateStorageA.ID,
			s.ID,
			"User from workspace B should not see private storage from workspace A",
		)
	}
	assert.True(t, foundSystemBInWorkspaceB, "User from workspace B should see system storage B")

	// Test: Outsider (not in any workspace) cannot access storages
	outsider := users_testing.CreateTestUser(users_enums.UserRoleMember)
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspaceA.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	// Cleanup
	deleteStorage(t, router, savedPrivateStorageA.ID, ownerA.Token)
	deleteStorage(t, router, savedSystemStorageB.ID, admin.Token)
	workspaces_testing.RemoveTestWorkspace(workspaceA, router)
	workspaces_testing.RemoveTestWorkspace(workspaceB, router)
}

func Test_GetSystemStorage_SensitiveDataHiddenForNonAdmin(t *testing.T) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", member, router)

	// Admin creates system S3 storage with credentials
	systemS3Storage := &Storage{
		WorkspaceID: workspace.ID,
		Type:        StorageTypeS3,
		Name:        "Test System S3 Storage " + uuid.New().String(),
		IsSystem:    true,
		S3Storage: &s3_storage.S3Storage{
			S3Bucket:    "test-system-bucket",
			S3Region:    "us-east-1",
			S3AccessKey: "test-access-key-123",
			S3SecretKey: "test-secret-key-456",
			S3Endpoint:  "https://s3.amazonaws.com",
		},
	}

	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemS3Storage,
		http.StatusOK,
		&savedStorage,
	)

	assert.NotEmpty(t, savedStorage.ID)
	assert.True(t, savedStorage.IsSystem)

	// Test: Admin retrieves system storage - should see S3Storage object with hidden sensitive fields
	var adminView Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&adminView,
	)

	assert.NotNil(t, adminView.S3Storage, "Admin should see S3Storage object")
	assert.Equal(t, "test-system-bucket", adminView.S3Storage.S3Bucket)
	assert.Equal(t, "us-east-1", adminView.S3Storage.S3Region)
	// Sensitive fields should be hidden (empty strings)
	assert.Equal(
		t,
		"",
		adminView.S3Storage.S3AccessKey,
		"Admin should see hidden (empty) access key",
	)
	assert.Equal(
		t,
		"",
		adminView.S3Storage.S3SecretKey,
		"Admin should see hidden (empty) secret key",
	)

	// Test: Member retrieves system storage - should see storage but all specific data hidden
	var memberView Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
		"Bearer "+member.Token,
		http.StatusOK,
		&memberView,
	)

	assert.Equal(t, savedStorage.ID, memberView.ID)
	assert.Equal(t, savedStorage.Name, memberView.Name)
	assert.True(t, memberView.IsSystem)

	// All storage type objects should be nil for non-admin viewing system storage
	assert.Nil(t, memberView.S3Storage, "Non-admin should not see S3Storage object")
	assert.Nil(t, memberView.LocalStorage, "Non-admin should not see LocalStorage object")
	assert.Nil(
		t,
		memberView.GoogleDriveStorage,
		"Non-admin should not see GoogleDriveStorage object",
	)
	assert.Nil(t, memberView.NASStorage, "Non-admin should not see NASStorage object")
	assert.Nil(t, memberView.AzureBlobStorage, "Non-admin should not see AzureBlobStorage object")
	assert.Nil(t, memberView.FTPStorage, "Non-admin should not see FTPStorage object")
	assert.Nil(t, memberView.SFTPStorage, "Non-admin should not see SFTPStorage object")
	assert.Nil(t, memberView.RcloneStorage, "Non-admin should not see RcloneStorage object")

	// Test: Member can also see system storage in GetStorages list
	var storages []Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages?workspace_id=%s", workspace.ID.String()),
		"Bearer "+member.Token,
		http.StatusOK,
		&storages,
	)

	foundSystemStorage := false
	for _, s := range storages {
		if s.ID == savedStorage.ID {
			foundSystemStorage = true
			assert.True(t, s.IsSystem)
			assert.Nil(t, s.S3Storage, "Non-admin should not see S3Storage in list")
		}
	}
	assert.True(t, foundSystemStorage, "System storage should be in list")

	// Cleanup
	deleteStorage(t, router, savedStorage.ID, admin.Token)
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

	deleteStorage(t, router, savedStorage.ID, owner.Token)
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

	deleteStorage(t, router, savedStorage.ID, owner1.Token)
	workspaces_testing.RemoveTestWorkspace(workspace1, router)
	workspaces_testing.RemoveTestWorkspace(workspace2, router)
}

func Test_StorageSensitiveDataLifecycle_AllTypes(t *testing.T) {
	testCases := []struct {
		name                string
		storageType         StorageType
		createStorage       func(workspaceID uuid.UUID) *Storage
		updateStorage       func(workspaceID, storageID uuid.UUID) *Storage
		verifySensitiveData func(t *testing.T, storage *Storage)
		verifyHiddenData    func(t *testing.T, storage *Storage)
	}{
		{
			name:        "S3 Storage",
			storageType: StorageTypeS3,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeS3,
					Name:        "Test S3 Storage",
					S3Storage: &s3_storage.S3Storage{
						S3Bucket:    "test-bucket",
						S3Region:    "us-east-1",
						S3AccessKey: "original-access-key",
						S3SecretKey: "original-secret-key",
						S3Endpoint:  "https://s3.amazonaws.com",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeS3,
					Name:        "Updated S3 Storage",
					S3Storage: &s3_storage.S3Storage{
						S3Bucket:    "updated-bucket",
						S3Region:    "us-west-2",
						S3AccessKey: "",
						S3SecretKey: "",
						S3Endpoint:  "https://s3.us-west-2.amazonaws.com",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.S3Storage.S3AccessKey, "enc:"),
					"S3AccessKey should be encrypted with 'enc:' prefix")
				assert.True(t, strings.HasPrefix(storage.S3Storage.S3SecretKey, "enc:"),
					"S3SecretKey should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				accessKey, err := encryptor.Decrypt(storage.ID, storage.S3Storage.S3AccessKey)
				assert.NoError(t, err)
				assert.Equal(t, "original-access-key", accessKey)

				secretKey, err := encryptor.Decrypt(storage.ID, storage.S3Storage.S3SecretKey)
				assert.NoError(t, err)
				assert.Equal(t, "original-secret-key", secretKey)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.S3Storage.S3AccessKey)
				assert.Equal(t, "", storage.S3Storage.S3SecretKey)
			},
		},
		{
			name:        "Local Storage",
			storageType: StorageTypeLocal,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID:  workspaceID,
					Type:         StorageTypeLocal,
					Name:         "Test Local Storage",
					LocalStorage: &local_storage.LocalStorage{},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:           storageID,
					WorkspaceID:  workspaceID,
					Type:         StorageTypeLocal,
					Name:         "Updated Local Storage",
					LocalStorage: &local_storage.LocalStorage{},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
			},
		},
		{
			name:        "NAS Storage",
			storageType: StorageTypeNAS,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeNAS,
					Name:        "Test NAS Storage",
					NASStorage: &nas_storage.NASStorage{
						Host:     "nas.example.com",
						Port:     445,
						Share:    "backups",
						Username: "testuser",
						Password: "original-password",
						UseSSL:   false,
						Domain:   "WORKGROUP",
						Path:     "/test",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeNAS,
					Name:        "Updated NAS Storage",
					NASStorage: &nas_storage.NASStorage{
						Host:     "nas2.example.com",
						Port:     445,
						Share:    "backups2",
						Username: "testuser2",
						Password: "",
						UseSSL:   true,
						Domain:   "WORKGROUP2",
						Path:     "/test2",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.NASStorage.Password, "enc:"),
					"Password should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				password, err := encryptor.Decrypt(storage.ID, storage.NASStorage.Password)
				assert.NoError(t, err)
				assert.Equal(t, "original-password", password)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.NASStorage.Password)
			},
		},
		{
			name:        "Azure Blob Storage (Connection String)",
			storageType: StorageTypeAzureBlob,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeAzureBlob,
					Name:        "Test Azure Blob Storage",
					AzureBlobStorage: &azure_blob_storage.AzureBlobStorage{
						AuthMethod:       azure_blob_storage.AuthMethodConnectionString,
						ConnectionString: "original-connection-string",
						ContainerName:    "test-container",
						Endpoint:         "",
						Prefix:           "backups/",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeAzureBlob,
					Name:        "Updated Azure Blob Storage",
					AzureBlobStorage: &azure_blob_storage.AzureBlobStorage{
						AuthMethod:       azure_blob_storage.AuthMethodConnectionString,
						ConnectionString: "",
						ContainerName:    "updated-container",
						Endpoint:         "https://custom.blob.core.windows.net",
						Prefix:           "backups2/",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.AzureBlobStorage.ConnectionString, "enc:"),
					"ConnectionString should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				connectionString, err := encryptor.Decrypt(
					storage.ID,
					storage.AzureBlobStorage.ConnectionString,
				)
				assert.NoError(t, err)
				assert.Equal(t, "original-connection-string", connectionString)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.AzureBlobStorage.ConnectionString)
				assert.Equal(t, "", storage.AzureBlobStorage.AccountKey)
			},
		},
		{
			name:        "Azure Blob Storage (Account Key)",
			storageType: StorageTypeAzureBlob,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeAzureBlob,
					Name:        "Test Azure Blob with Account Key",
					AzureBlobStorage: &azure_blob_storage.AzureBlobStorage{
						AuthMethod:    azure_blob_storage.AuthMethodAccountKey,
						AccountName:   "testaccount",
						AccountKey:    "original-account-key",
						ContainerName: "test-container",
						Endpoint:      "",
						Prefix:        "backups/",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeAzureBlob,
					Name:        "Updated Azure Blob with Account Key",
					AzureBlobStorage: &azure_blob_storage.AzureBlobStorage{
						AuthMethod:    azure_blob_storage.AuthMethodAccountKey,
						AccountName:   "updatedaccount",
						AccountKey:    "",
						ContainerName: "updated-container",
						Endpoint:      "https://custom.blob.core.windows.net",
						Prefix:        "backups2/",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.AzureBlobStorage.AccountKey, "enc:"),
					"AccountKey should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				accountKey, err := encryptor.Decrypt(
					storage.ID,
					storage.AzureBlobStorage.AccountKey,
				)
				assert.NoError(t, err)
				assert.Equal(t, "original-account-key", accountKey)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.AzureBlobStorage.ConnectionString)
				assert.Equal(t, "", storage.AzureBlobStorage.AccountKey)
			},
		},
		{
			name:        "FTP Storage",
			storageType: StorageTypeFTP,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeFTP,
					Name:        "Test FTP Storage",
					FTPStorage: &ftp_storage.FTPStorage{
						Host:     "ftp.example.com",
						Port:     21,
						Username: "testuser",
						Password: "original-password",
						UseSSL:   false,
						Path:     "/backups",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeFTP,
					Name:        "Updated FTP Storage",
					FTPStorage: &ftp_storage.FTPStorage{
						Host:     "ftp2.example.com",
						Port:     2121,
						Username: "testuser2",
						Password: "",
						UseSSL:   true,
						Path:     "/backups2",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.FTPStorage.Password, "enc:"),
					"Password should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				password, err := encryptor.Decrypt(storage.ID, storage.FTPStorage.Password)
				assert.NoError(t, err)
				assert.Equal(t, "original-password", password)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.FTPStorage.Password)
			},
		},
		{
			name:        "SFTP Storage",
			storageType: StorageTypeSFTP,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeSFTP,
					Name:        "Test SFTP Storage",
					SFTPStorage: &sftp_storage.SFTPStorage{
						Host:              "sftp.example.com",
						Port:              22,
						Username:          "testuser",
						Password:          "original-password",
						PrivateKey:        "original-private-key",
						SkipHostKeyVerify: false,
						Path:              "/backups",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeSFTP,
					Name:        "Updated SFTP Storage",
					SFTPStorage: &sftp_storage.SFTPStorage{
						Host:              "sftp2.example.com",
						Port:              2222,
						Username:          "testuser2",
						Password:          "",
						PrivateKey:        "",
						SkipHostKeyVerify: true,
						Path:              "/backups2",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.SFTPStorage.Password, "enc:"),
					"Password should be encrypted with 'enc:' prefix")
				assert.True(t, strings.HasPrefix(storage.SFTPStorage.PrivateKey, "enc:"),
					"PrivateKey should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				password, err := encryptor.Decrypt(storage.ID, storage.SFTPStorage.Password)
				assert.NoError(t, err)
				assert.Equal(t, "original-password", password)

				privateKey, err := encryptor.Decrypt(storage.ID, storage.SFTPStorage.PrivateKey)
				assert.NoError(t, err)
				assert.Equal(t, "original-private-key", privateKey)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.SFTPStorage.Password)
				assert.Equal(t, "", storage.SFTPStorage.PrivateKey)
			},
		},
		{
			name:        "Rclone Storage",
			storageType: StorageTypeRclone,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeRclone,
					Name:        "Test Rclone Storage",
					RcloneStorage: &rclone_storage.RcloneStorage{
						ConfigContent: "[myremote]\ntype = s3\nprovider = AWS\naccess_key_id = test\nsecret_access_key = secret\n",
						RemotePath:    "/backups",
					},
				}
			},
			updateStorage: func(workspaceID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeRclone,
					Name:        "Updated Rclone Storage",
					RcloneStorage: &rclone_storage.RcloneStorage{
						ConfigContent: "",
						RemotePath:    "/backups2",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.RcloneStorage.ConfigContent, "enc:"),
					"ConfigContent should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				configContent, err := encryptor.Decrypt(
					storage.ID,
					storage.RcloneStorage.ConfigContent,
				)
				assert.NoError(t, err)
				assert.Equal(
					t,
					"[myremote]\ntype = s3\nprovider = AWS\naccess_key_id = test\nsecret_access_key = secret\n",
					configContent,
				)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.RcloneStorage.ConfigContent)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			router := createRouter()
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			// Phase 1: Create storage with sensitive data
			initialStorage := tc.createStorage(workspace.ID)
			var createdStorage Storage
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/storages",
				"Bearer "+owner.Token,
				*initialStorage,
				http.StatusOK,
				&createdStorage,
			)

			assert.NotEmpty(t, createdStorage.ID)
			assert.Equal(t, initialStorage.Name, createdStorage.Name)

			// Phase 2: Verify sensitive data is encrypted in repository after creation
			repository := &StorageRepository{}
			storageFromDBAfterCreate, err := repository.FindByID(createdStorage.ID)
			assert.NoError(t, err)
			tc.verifySensitiveData(t, storageFromDBAfterCreate)

			// Phase 3: Read via service - sensitive data should be hidden
			var retrievedStorage Storage
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/storages/%s", createdStorage.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&retrievedStorage,
			)

			tc.verifyHiddenData(t, &retrievedStorage)
			assert.Equal(t, initialStorage.Name, retrievedStorage.Name)

			// Phase 4: Update with non-sensitive changes only (sensitive fields empty)
			updatedStorage := tc.updateStorage(workspace.ID, createdStorage.ID)
			var updateResponse Storage
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/storages",
				"Bearer "+owner.Token,
				*updatedStorage,
				http.StatusOK,
				&updateResponse,
			)

			// Verify non-sensitive fields were updated
			assert.Equal(t, updatedStorage.Name, updateResponse.Name)

			// Phase 5: Retrieve directly from repository to verify sensitive data preservation
			storageFromDB, err := repository.FindByID(createdStorage.ID)
			assert.NoError(t, err)

			// Verify original sensitive data is still present in DB
			tc.verifySensitiveData(t, storageFromDB)

			// Verify non-sensitive fields were updated in DB
			assert.Equal(t, updatedStorage.Name, storageFromDB.Name)

			// Additional verification: Check via GET that data is still hidden
			var finalRetrieved Storage
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/storages/%s", createdStorage.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&finalRetrieved,
			)
			tc.verifyHiddenData(t, &finalRetrieved)

			// Cleanup
			deleteStorage(t, router, createdStorage.ID, owner.Token)
			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}

func Test_TransferStorage_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		sourceRole         *users_enums.WorkspaceRole
		targetRole         *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "owner in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "admin in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "member in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "viewer in both workspaces cannot transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "global admin can transfer",
			sourceRole:         nil,
			targetRole:         nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createRouter()
			GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

			sourceOwner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			targetOwner := users_testing.CreateTestUser(users_enums.UserRoleMember)

			sourceWorkspace := workspaces_testing.CreateTestWorkspace(
				"Source Workspace",
				sourceOwner,
				router,
			)
			targetWorkspace := workspaces_testing.CreateTestWorkspace(
				"Target Workspace",
				targetOwner,
				router,
			)

			storage := createNewStorage(sourceWorkspace.ID)
			var savedStorage Storage
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/storages",
				"Bearer "+sourceOwner.Token,
				*storage,
				http.StatusOK,
				&savedStorage,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.sourceRole != nil {
				testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					sourceWorkspace,
					testUser,
					*tt.sourceRole,
					sourceOwner.Token,
					router,
				)
				workspaces_testing.AddMemberToWorkspace(
					targetWorkspace,
					testUser,
					*tt.targetRole,
					targetOwner.Token,
					router,
				)
				testUserToken = testUser.Token
			}

			request := TransferStorageRequest{
				TargetWorkspaceID: targetWorkspace.ID,
			}

			testResp := test_utils.MakePostRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/storages/%s/transfer", savedStorage.ID.String()),
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "transferred successfully")

				var retrievedStorage Storage
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					fmt.Sprintf("/api/v1/storages/%s", savedStorage.ID.String()),
					"Bearer "+targetOwner.Token,
					http.StatusOK,
					&retrievedStorage,
				)
				assert.Equal(t, targetWorkspace.ID, retrievedStorage.WorkspaceID)

				deleteStorage(t, router, savedStorage.ID, targetOwner.Token)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
				deleteStorage(t, router, savedStorage.ID, sourceOwner.Token)
			}

			workspaces_testing.RemoveTestWorkspace(sourceWorkspace, router)
			workspaces_testing.RemoveTestWorkspace(targetWorkspace, router)
		})
	}
}

func Test_TransferStorageNotManagableWorkspace_TransferFailed(t *testing.T) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	userA := users_testing.CreateTestUser(users_enums.UserRoleMember)
	userB := users_testing.CreateTestUser(users_enums.UserRoleMember)

	workspace1 := workspaces_testing.CreateTestWorkspace("Workspace 1", userA, router)
	workspace2 := workspaces_testing.CreateTestWorkspace("Workspace 2", userB, router)

	storage := createNewStorage(workspace1.ID)
	var savedStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+userA.Token,
		*storage,
		http.StatusOK,
		&savedStorage,
	)

	request := TransferStorageRequest{
		TargetWorkspaceID: workspace2.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s/transfer", savedStorage.ID.String()),
		"Bearer "+userA.Token,
		request,
		http.StatusForbidden,
	)

	assert.Contains(
		t,
		string(testResp.Body),
		"insufficient permissions to manage storage in target workspace",
	)

	deleteStorage(t, router, savedStorage.ID, userA.Token)
	workspaces_testing.RemoveTestWorkspace(workspace1, router)
	workspaces_testing.RemoveTestWorkspace(workspace2, router)
}

func Test_TransferSystemStorage_TransferBlocked(t *testing.T) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspaceA := workspaces_testing.CreateTestWorkspace("Workspace A", admin, router)
	workspaceB := workspaces_testing.CreateTestWorkspace("Workspace B", admin, router)

	// Admin creates system storage in workspace A
	systemStorage := &Storage{
		WorkspaceID:  workspaceA.ID,
		Type:         StorageTypeLocal,
		Name:         "Test System Storage " + uuid.New().String(),
		IsSystem:     true,
		LocalStorage: &local_storage.LocalStorage{},
	}
	var savedSystemStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemStorage,
		http.StatusOK,
		&savedSystemStorage,
	)

	// Admin attempts to transfer system storage to workspace B - should be blocked
	transferRequest := TransferStorageRequest{
		TargetWorkspaceID: workspaceB.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s/transfer", savedSystemStorage.ID.String()),
		"Bearer "+admin.Token,
		transferRequest,
		http.StatusBadRequest,
	)

	assert.Contains(
		t,
		string(testResp.Body),
		"system storage cannot be transferred",
		"Transfer should fail with appropriate error message",
	)

	// Verify storage is still in workspace A
	var retrievedStorage Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedSystemStorage.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&retrievedStorage,
	)
	assert.Equal(
		t,
		workspaceA.ID,
		retrievedStorage.WorkspaceID,
		"Storage should remain in workspace A",
	)

	// Test regression: Non-system storage can still be transferred
	privateStorage := createNewStorage(workspaceA.ID)
	var savedPrivateStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*privateStorage,
		http.StatusOK,
		&savedPrivateStorage,
	)

	privateTransferResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s/transfer", savedPrivateStorage.ID.String()),
		"Bearer "+admin.Token,
		transferRequest,
		http.StatusOK,
	)

	assert.Contains(
		t,
		string(privateTransferResp.Body),
		"transferred successfully",
		"Private storage should be transferable",
	)

	// Verify private storage was transferred to workspace B
	var transferredStorage Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/storages/%s", savedPrivateStorage.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&transferredStorage,
	)
	assert.Equal(
		t,
		workspaceB.ID,
		transferredStorage.WorkspaceID,
		"Private storage should be in workspace B",
	)

	// Cleanup
	deleteStorage(t, router, savedSystemStorage.ID, admin.Token)
	deleteStorage(t, router, savedPrivateStorage.ID, admin.Token)
	workspaces_testing.RemoveTestWorkspace(workspaceA, router)
	workspaces_testing.RemoveTestWorkspace(workspaceB, router)
}

func Test_DeleteWorkspace_SystemStoragesFromAnotherWorkspaceNotRemovedAndWorkspaceDeletedSuccessfully(
	t *testing.T,
) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspaceA := workspaces_testing.CreateTestWorkspace("Workspace A", admin, router)
	workspaceD := workspaces_testing.CreateTestWorkspace("Workspace D", admin, router)

	// Create a system storage in workspace A
	systemStorage := &Storage{
		WorkspaceID:  workspaceA.ID,
		Type:         StorageTypeLocal,
		Name:         "Test System Storage " + uuid.New().String(),
		IsSystem:     true,
		LocalStorage: &local_storage.LocalStorage{},
	}
	var savedSystemStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemStorage,
		http.StatusOK,
		&savedSystemStorage,
	)
	assert.True(t, savedSystemStorage.IsSystem)
	assert.Equal(t, workspaceA.ID, savedSystemStorage.WorkspaceID)

	// Create a regular storage in workspace D
	regularStorage := createNewStorage(workspaceD.ID)
	var savedRegularStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*regularStorage,
		http.StatusOK,
		&savedRegularStorage,
	)
	assert.False(t, savedRegularStorage.IsSystem)
	assert.Equal(t, workspaceD.ID, savedRegularStorage.WorkspaceID)

	// Delete workspace D
	workspaces_testing.DeleteWorkspace(workspaceD, admin.Token, router)

	// Verify system storage from workspace A still exists
	repository := &StorageRepository{}
	systemStorageAfterDeletion, err := repository.FindByID(savedSystemStorage.ID)
	assert.NoError(t, err, "System storage should still exist after workspace D deletion")
	assert.NotNil(t, systemStorageAfterDeletion)
	assert.Equal(t, savedSystemStorage.ID, systemStorageAfterDeletion.ID)
	assert.True(t, systemStorageAfterDeletion.IsSystem)
	assert.Equal(t, workspaceA.ID, systemStorageAfterDeletion.WorkspaceID)

	// Verify regular storage from workspace D was deleted
	regularStorageAfterDeletion, err := repository.FindByID(savedRegularStorage.ID)
	assert.Error(t, err, "Regular storage should be deleted with workspace D")
	assert.Nil(t, regularStorageAfterDeletion)

	// Cleanup
	deleteStorage(t, router, savedSystemStorage.ID, admin.Token)
	workspaces_testing.RemoveTestWorkspace(workspaceA, router)
}

func Test_DeleteWorkspace_WithOwnSystemStorage_ReturnsForbidden(t *testing.T) {
	router := createRouter()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspaceA := workspaces_testing.CreateTestWorkspace("Workspace A", admin, router)

	// Create a system storage assigned to workspace A
	systemStorage := &Storage{
		WorkspaceID:  workspaceA.ID,
		Type:         StorageTypeLocal,
		Name:         "System Storage in A " + uuid.New().String(),
		IsSystem:     true,
		LocalStorage: &local_storage.LocalStorage{},
	}

	var savedSystemStorage Storage
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages",
		"Bearer "+admin.Token,
		*systemStorage,
		http.StatusOK,
		&savedSystemStorage,
	)
	assert.True(t, savedSystemStorage.IsSystem)
	assert.Equal(t, workspaceA.ID, savedSystemStorage.WorkspaceID)

	// Attempt to delete workspace A - should fail because it has a system storage
	resp := workspaces_testing.MakeAPIRequest(
		router,
		"DELETE",
		"/api/v1/workspaces/"+workspaceA.ID.String(),
		"Bearer "+admin.Token,
		nil,
	)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "Workspace deletion should fail")
	assert.Contains(
		t,
		resp.Body.String(),
		"system storage cannot be deleted due to workspace deletion",
		"Error message should indicate system storage prevents deletion",
	)

	// Verify workspace still exists
	workspaceRepo := &workspaces_repositories.WorkspaceRepository{}
	workspaceAfterFailedDeletion, err := workspaceRepo.GetWorkspaceByID(workspaceA.ID)
	assert.NoError(t, err, "Workspace should still exist after failed deletion")
	assert.NotNil(t, workspaceAfterFailedDeletion)
	assert.Equal(t, workspaceA.ID, workspaceAfterFailedDeletion.ID)

	// Verify system storage still exists
	repository := &StorageRepository{}
	storageAfterFailedDeletion, err := repository.FindByID(savedSystemStorage.ID)
	assert.NoError(t, err, "System storage should still exist after failed deletion")
	assert.NotNil(t, storageAfterFailedDeletion)
	assert.Equal(t, savedSystemStorage.ID, storageAfterFailedDeletion.ID)
	assert.True(t, storageAfterFailedDeletion.IsSystem)

	// Cleanup: Delete system storage first, then workspace can be deleted
	deleteStorage(t, router, savedSystemStorage.ID, admin.Token)
	workspaces_testing.DeleteWorkspace(workspaceA, admin.Token, router)

	// Verify workspace was successfully deleted after storage removal
	_, err = workspaceRepo.GetWorkspaceByID(workspaceA.ID)
	assert.Error(t, err, "Workspace should be deleted after storage was removed")
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
	SetupDependencies()
	GetStorageService().SetStorageDatabaseCounter(&mockStorageDatabaseCounter{})

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

func verifyStorageData(t *testing.T, expected, actual *Storage) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.IsSystem, actual.IsSystem)
}

func deleteStorage(
	t *testing.T,
	router *gin.Engine,
	storageID uuid.UUID,
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
