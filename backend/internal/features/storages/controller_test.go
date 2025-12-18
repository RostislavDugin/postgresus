package storages

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	azure_blob_storage "postgresus-backend/internal/features/storages/models/azure_blob"
	ftp_storage "postgresus-backend/internal/features/storages/models/ftp"
	google_drive_storage "postgresus-backend/internal/features/storages/models/google_drive"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	nas_storage "postgresus-backend/internal/features/storages/models/nas"
	rclone_storage "postgresus-backend/internal/features/storages/models/rclone"
	s3_storage "postgresus-backend/internal/features/storages/models/s3"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	"postgresus-backend/internal/util/encryption"
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
