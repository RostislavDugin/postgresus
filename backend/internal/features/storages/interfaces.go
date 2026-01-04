package storages

import (
	"context"
	"databasus-backend/internal/util/encryption"
	"io"
	"log/slog"

	"github.com/google/uuid"
)

type StorageFileSaver interface {
	SaveFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileID uuid.UUID,
		file io.Reader,
	) error

	GetFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) (io.ReadCloser, error)

	DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error

	Validate(encryptor encryption.FieldEncryptor) error

	TestConnection(encryptor encryption.FieldEncryptor) error

	HideSensitiveData()

	EncryptSensitiveData(encryptor encryption.FieldEncryptor) error
}

type StorageDatabaseCounter interface {
	GetStorageAttachedDatabasesIDs(storageID uuid.UUID) ([]uuid.UUID, error)
}
