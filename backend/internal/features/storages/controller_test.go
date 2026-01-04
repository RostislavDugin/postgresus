package storages

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	audit_logs "databasus-backend/internal/features/audit_logs"
	azure_blob_storage "databasus-backend/internal/features/storages/models/azure_blob"
	ftp_storage "databasus-backend/internal/features/storages/models/ftp"
	google_drive_storage "databasus-backend/internal/features/storages/models/google_drive"
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
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
			assert.Len(t, storages, 1)

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
				deleteStorage(t, router, savedStorage.ID, workspace.ID, owner.Token)
			}
			if !tt.canDelete {
				deleteStorage(t, router, ownerStorage.ID, workspace.ID, owner.Token)
			}
			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
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

func Test_StorageSensitiveDataLifecycle_AllTypes(t *testing.T) {
	testCases := []struct {
		name                string
		storageType         StorageType
		createStorage       func(workspaceID uuid.UUID) *Storage
		updateStorage       func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			name:        "Google Drive Storage",
			storageType: StorageTypeGoogleDrive,
			createStorage: func(workspaceID uuid.UUID) *Storage {
				return &Storage{
					WorkspaceID: workspaceID,
					Type:        StorageTypeGoogleDrive,
					Name:        "Test Google Drive Storage",
					GoogleDriveStorage: &google_drive_storage.GoogleDriveStorage{
						ClientID:     "original-client-id",
						ClientSecret: "original-client-secret",
						TokenJSON:    `{"access_token":"ya29.test-access-token","token_type":"Bearer","expiry":"2030-12-31T23:59:59Z","refresh_token":"1//test-refresh-token"}`,
					},
				}
			},
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
				return &Storage{
					ID:          storageID,
					WorkspaceID: workspaceID,
					Type:        StorageTypeGoogleDrive,
					Name:        "Updated Google Drive Storage",
					GoogleDriveStorage: &google_drive_storage.GoogleDriveStorage{
						ClientID:     "updated-client-id",
						ClientSecret: "",
						TokenJSON:    "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, storage *Storage) {
				assert.True(t, strings.HasPrefix(storage.GoogleDriveStorage.ClientSecret, "enc:"),
					"ClientSecret should be encrypted with 'enc:' prefix")
				assert.True(t, strings.HasPrefix(storage.GoogleDriveStorage.TokenJSON, "enc:"),
					"TokenJSON should be encrypted with 'enc:' prefix")

				encryptor := encryption.GetFieldEncryptor()
				clientSecret, err := encryptor.Decrypt(
					storage.ID,
					storage.GoogleDriveStorage.ClientSecret,
				)
				assert.NoError(t, err)
				assert.Equal(t, "original-client-secret", clientSecret)

				tokenJSON, err := encryptor.Decrypt(
					storage.ID,
					storage.GoogleDriveStorage.TokenJSON,
				)
				assert.NoError(t, err)
				assert.Equal(
					t,
					`{"access_token":"ya29.test-access-token","token_type":"Bearer","expiry":"2030-12-31T23:59:59Z","refresh_token":"1//test-refresh-token"}`,
					tokenJSON,
				)
			},
			verifyHiddenData: func(t *testing.T, storage *Storage) {
				assert.Equal(t, "", storage.GoogleDriveStorage.ClientSecret)
				assert.Equal(t, "", storage.GoogleDriveStorage.TokenJSON)
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
			updateStorage: func(workspaceID uuid.UUID, storageID uuid.UUID) *Storage {
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
				workspaces_testing.AddMemberToWorkspace(sourceWorkspace, testUser, *tt.sourceRole, sourceOwner.Token, router)
				workspaces_testing.AddMemberToWorkspace(targetWorkspace, testUser, *tt.targetRole, targetOwner.Token, router)
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

				deleteStorage(t, router, savedStorage.ID, targetWorkspace.ID, targetOwner.Token)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
				deleteStorage(t, router, savedStorage.ID, sourceWorkspace.ID, sourceOwner.Token)
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

	deleteStorage(t, router, savedStorage.ID, workspace1.ID, userA.Token)
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
