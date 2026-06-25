package storages

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	azure_blob_storage "databasus-backend/internal/features/storages/models/azure_blob"
	ftp_storage "databasus-backend/internal/features/storages/models/ftp"
	local_storage "databasus-backend/internal/features/storages/models/local"
	nas_storage "databasus-backend/internal/features/storages/models/nas"
	rclone_storage "databasus-backend/internal/features/storages/models/rclone"
	s3_storage "databasus-backend/internal/features/storages/models/s3"
	sftp_storage "databasus-backend/internal/features/storages/models/sftp"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
)

type S3Container struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	region     string
}

type AzuriteContainer struct {
	endpoint         string
	accountName      string
	accountKey       string
	containerNameKey string
	containerNameStr string
	connectionString string
}

func Test_Storage_BasicOperations(t *testing.T) {
	ctx := t.Context()

	minioEndpoint := containers.StartMinio(t)
	azuriteEndpoint := containers.StartAzurite(t)
	nasEndpoint := containers.StartSamba(t)
	ftpEndpoint := containers.StartFtp(t)
	sftpEndpoint := containers.StartSftp(t)

	s3Container, err := setupS3Container(ctx, addressOf(minioEndpoint))
	require.NoError(t, err, "Failed to setup S3 container")

	azuriteContainer, err := setupAzuriteContainer(ctx, addressOf(azuriteEndpoint))
	require.NoError(t, err, "Failed to setup Azurite container")

	testFilePath, err := setupTestFile()
	require.NoError(t, err, "Failed to setup test file")
	defer os.Remove(testFilePath)

	testCases := []struct {
		name    string
		storage StorageFileSaver
	}{
		{
			name:    "LocalStorage",
			storage: &local_storage.LocalStorage{StorageID: uuid.New()},
		},
		{
			name: "S3Storage",
			storage: &s3_storage.S3Storage{
				StorageID:   uuid.New(),
				S3Bucket:    s3Container.bucketName,
				S3Region:    s3Container.region,
				S3AccessKey: s3Container.accessKey,
				S3SecretKey: s3Container.secretKey,
				S3Endpoint:  "http://" + s3Container.endpoint,
			},
		},
		{
			name: "S3Storage_WithStorageClass",
			storage: &s3_storage.S3Storage{
				StorageID:      uuid.New(),
				S3Bucket:       s3Container.bucketName,
				S3Region:       s3Container.region,
				S3AccessKey:    s3Container.accessKey,
				S3SecretKey:    s3Container.secretKey,
				S3Endpoint:     "http://" + s3Container.endpoint,
				S3StorageClass: s3_storage.S3StorageClassStandard,
			},
		},
		{
			name: "NASStorage",
			storage: &nas_storage.NASStorage{
				StorageID: uuid.New(),
				Host:      nasEndpoint.Host,
				Port:      nasEndpoint.Port,
				Share:     containers.SambaShare,
				Username:  containers.SambaUsername,
				Password:  containers.SambaPassword,
				UseSSL:    false,
				Domain:    "",
				Path:      "test-files",
			},
		},
		{
			name: "AzureBlobStorage_AccountKey",
			storage: &azure_blob_storage.AzureBlobStorage{
				StorageID:     uuid.New(),
				AuthMethod:    azure_blob_storage.AuthMethodAccountKey,
				AccountName:   azuriteContainer.accountName,
				AccountKey:    azuriteContainer.accountKey,
				ContainerName: azuriteContainer.containerNameKey,
				Endpoint:      azuriteContainer.endpoint,
			},
		},
		{
			name: "AzureBlobStorage_ConnectionString",
			storage: &azure_blob_storage.AzureBlobStorage{
				StorageID:        uuid.New(),
				AuthMethod:       azure_blob_storage.AuthMethodConnectionString,
				ConnectionString: azuriteContainer.connectionString,
				ContainerName:    azuriteContainer.containerNameStr,
			},
		},
		{
			name: "FTPStorage",
			storage: &ftp_storage.FTPStorage{
				StorageID: uuid.New(),
				Host:      ftpEndpoint.Host,
				Port:      ftpEndpoint.Port,
				Username:  containers.FtpUsername,
				Password:  containers.FtpPassword,
				UseSSL:    false,
				Path:      "test-files",
			},
		},
		{
			name: "SFTPStorage",
			storage: &sftp_storage.SFTPStorage{
				StorageID:         uuid.New(),
				Host:              sftpEndpoint.Host,
				Port:              sftpEndpoint.Port,
				Username:          containers.SftpUsername,
				Password:          containers.SftpPassword,
				SkipHostKeyVerify: true,
				Path:              containers.SftpUploadDir,
			},
		},
		{
			name: "RcloneStorage",
			storage: &rclone_storage.RcloneStorage{
				StorageID: uuid.New(),
				ConfigContent: fmt.Sprintf(`[minio]
type = s3
provider = Other
access_key_id = %s
secret_access_key = %s
endpoint = http://%s
acl = private`, s3Container.accessKey, s3Container.secretKey, s3Container.endpoint),
				RemotePath: s3Container.bucketName,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encryptor := encryption.GetFieldEncryptor()

			t.Run("Test_TestConnection_ConnectionSucceeds", func(t *testing.T) {
				err := tc.storage.TestConnection(encryptor)
				assert.NoError(t, err, "TestConnection should succeed")
			})

			t.Run("Test_TestValidation_ValidationSucceeds", func(t *testing.T) {
				err := tc.storage.Validate(encryptor)
				assert.NoError(t, err, "Validate should succeed")
			})

			t.Run("Test_TestSaveAndGetFile_ReturnsCorrectContent", func(t *testing.T) {
				fileData, err := os.ReadFile(testFilePath)
				require.NoError(t, err, "Should be able to read test file")

				fileID := uuid.New()

				err = tc.storage.SaveFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
					bytes.NewReader(fileData),
				)
				require.NoError(t, err, "SaveFile should succeed")

				file, err := tc.storage.GetFile(encryptor, fileID.String())
				assert.NoError(t, err, "GetFile should succeed")
				defer file.Close()

				content, err := io.ReadAll(file)
				assert.NoError(t, err, "Should be able to read file")
				assert.Equal(t, fileData, content, "File content should match the original")
			})

			t.Run("Test_TestDeleteFile_RemovesFileFromDisk", func(t *testing.T) {
				fileData, err := os.ReadFile(testFilePath)
				require.NoError(t, err, "Should be able to read test file")

				fileID := uuid.New()
				err = tc.storage.SaveFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
					bytes.NewReader(fileData),
				)
				require.NoError(t, err, "SaveFile should succeed")

				err = tc.storage.DeleteFile(encryptor, fileID.String())
				assert.NoError(t, err, "DeleteFile should succeed")

				file, err := tc.storage.GetFile(encryptor, fileID.String())
				assert.Error(t, err, "GetFile should fail for non-existent file")
				if file != nil {
					file.Close()
				}
			})

			t.Run("Test_TestDeleteNonExistentFile_DoesNotError", func(t *testing.T) {
				nonExistentID := uuid.New()
				err := tc.storage.DeleteFile(encryptor, nonExistentID.String())
				assert.NoError(t, err, "DeleteFile should not error for non-existent file")
			})
		})
	}
}

func addressOf(endpoint containers.Endpoint) string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func setupTestFile() (string, error) {
	tempDir := os.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	testData := []byte("This is test data for storage testing")

	err := os.WriteFile(testFilePath, testData, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to create test file: %w", err)
	}

	return testFilePath, nil
}

// setupS3Container creates the test bucket on the MinIO server at address (host:port).
func setupS3Container(ctx context.Context, address string) (*S3Container, error) {
	accessKey := containers.MinioRootUser
	secretKey := containers.MinioRootPassword
	bucketName := "test-bucket"
	region := containers.MinioRegion

	minioClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		if err := minioClient.MakeBucket(
			ctx,
			bucketName,
			minio.MakeBucketOptions{Region: region},
		); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &S3Container{
		endpoint:   address,
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucketName: bucketName,
		region:     region,
	}, nil
}

// setupAzuriteContainer creates the two blob containers on the Azurite server at address (host:port).
func setupAzuriteContainer(ctx context.Context, address string) (*AzuriteContainer, error) {
	accountName := containers.AzuriteAccountName
	accountKey := containers.AzuriteAccountKey
	serviceURL := fmt.Sprintf("http://%s/%s", address, accountName)
	containerNameKey := "test-container-key"
	containerNameStr := "test-container-connstr"

	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s/%s",
		accountName,
		accountKey,
		address,
		accountName,
	)

	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azblob client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// The containers may already exist on a reused server; an AlreadyExists error is harmless here.
	_, _ = client.CreateContainer(ctx, containerNameKey, nil)
	_, _ = client.CreateContainer(ctx, containerNameStr, nil)

	return &AzuriteContainer{
		endpoint:         serviceURL,
		accountName:      accountName,
		accountKey:       accountKey,
		containerNameKey: containerNameKey,
		containerNameStr: containerNameStr,
		connectionString: connectionString,
	}, nil
}

func Test_RcloneStorage_DeleteFile_WhenAuthFailsOnLookup_ReturnsErrorAndDoesNotDeleteObject(t *testing.T) {
	ctx := t.Context()

	minioEndpoint := containers.StartMinio(t)

	s3Container, err := setupS3Container(ctx, addressOf(minioEndpoint))
	require.NoError(t, err)

	minioClient, err := minio.New(s3Container.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Container.accessKey, s3Container.secretKey, ""),
		Secure: false,
		Region: s3Container.region,
	})
	require.NoError(t, err)

	fileID := uuid.New().String()
	testData := []byte("rclone delete bug repro")

	_, err = minioClient.PutObject(
		ctx,
		s3Container.bucketName,
		fileID,
		bytes.NewReader(testData),
		int64(len(testData)),
		minio.PutObjectOptions{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = minioClient.RemoveObject(
			context.Background(),
			s3Container.bucketName,
			fileID,
			minio.RemoveObjectOptions{},
		)
	})

	rcloneStorage := &rclone_storage.RcloneStorage{
		StorageID: uuid.New(),
		ConfigContent: fmt.Sprintf(`[minio]
type = s3
provider = Other
access_key_id = %s
secret_access_key = totally-wrong-secret-key
endpoint = http://%s
acl = private`, s3Container.accessKey, s3Container.endpoint),
		RemotePath: s3Container.bucketName,
	}

	encryptor := encryption.GetFieldEncryptor()

	err = rcloneStorage.DeleteFile(encryptor, fileID)
	require.Error(t, err)

	_, statErr := minioClient.StatObject(
		ctx,
		s3Container.bucketName,
		fileID,
		minio.StatObjectOptions{},
	)
	assert.NoError(t, statErr)
}

func Test_StorageUpdate_WhenExistingStorageHasNilS3_AssignsIncomingS3(t *testing.T) {
	storageID := uuid.New()

	existing := &Storage{
		ID:        storageID,
		Type:      StorageTypeS3,
		Name:      "old name",
		S3Storage: nil,
	}

	incoming := &Storage{
		ID:   storageID,
		Type: StorageTypeS3,
		Name: "new name",
		S3Storage: &s3_storage.S3Storage{
			StorageID:   storageID,
			S3Bucket:    "my-bucket",
			S3Region:    "us-east-1",
			S3AccessKey: "access",
			S3SecretKey: "secret",
		},
	}

	existing.Update(incoming)

	assert.Equal(t, "new name", existing.Name)
	assert.NotNil(t, existing.S3Storage)
	assert.Equal(t, "my-bucket", existing.S3Storage.S3Bucket)
	assert.Equal(t, "us-east-1", existing.S3Storage.S3Region)
}

func Test_StorageUpdate_WhenExistingS3IsNil_ValidateDoesNotPanic(t *testing.T) {
	storageID := uuid.New()
	encryptor := encryption.GetFieldEncryptor()

	existing := &Storage{
		ID:        storageID,
		Type:      StorageTypeS3,
		Name:      "test",
		S3Storage: nil,
	}

	incoming := &Storage{
		ID:   storageID,
		Type: StorageTypeS3,
		Name: "test",
		S3Storage: &s3_storage.S3Storage{
			StorageID:   storageID,
			S3Bucket:    "my-bucket",
			S3Region:    "us-east-1",
			S3AccessKey: "access",
			S3SecretKey: "secret",
		},
	}

	existing.Update(incoming)

	assert.NotPanics(t, func() {
		_ = existing.Validate(encryptor)
	})
}
