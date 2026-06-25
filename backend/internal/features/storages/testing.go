package storages

import (
	"github.com/google/uuid"

	local_storage "databasus-backend/internal/features/storages/models/local"
	s3_storage "databasus-backend/internal/features/storages/models/s3"
	"databasus-backend/internal/util/encryption"
)

func SetStorageDatabaseCountersForTest(counters ...StorageDatabaseCounter) {
	storageService.storageDatabaseCounters = counters
}

func CreateTestStorage(workspaceID uuid.UUID) *Storage {
	storage := &Storage{
		WorkspaceID:  workspaceID,
		Type:         StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}

	storage, err := storageRepository.Save(storage)
	if err != nil {
		panic(err)
	}

	return storage
}

func CreateTestFlakyS3Storage(workspaceID uuid.UUID, endpoint string) *Storage {
	encryptor := encryption.GetFieldEncryptor()

	accessKey, err := encryptor.Encrypt("issue-582-access")
	if err != nil {
		panic(err)
	}

	secretKey, err := encryptor.Encrypt("issue-582-secret")
	if err != nil {
		panic(err)
	}

	storage := &Storage{
		WorkspaceID: workspaceID,
		Type:        StorageTypeS3,
		Name:        "Flaky S3 " + uuid.New().String(),
		S3Storage: &s3_storage.S3Storage{
			S3Bucket:    "issue-582-no-such-bucket-" + uuid.New().String(),
			S3Region:    "us-east-1",
			S3AccessKey: accessKey,
			S3SecretKey: secretKey,
			S3Endpoint:  endpoint,
		},
	}

	saved, err := storageRepository.Save(storage)
	if err != nil {
		panic(err)
	}

	return saved
}

func RemoveTestStorage(id uuid.UUID) {
	storage, err := storageRepository.FindByID(id)
	if err != nil {
		panic(err)
	}

	err = storageRepository.Delete(storage)
	if err != nil {
		panic(err)
	}
}
