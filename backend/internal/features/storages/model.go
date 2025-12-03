package storages

import (
	"context"
	"errors"
	"io"
	"log/slog"
	azure_blob_storage "postgresus-backend/internal/features/storages/models/azure_blob"
	google_drive_storage "postgresus-backend/internal/features/storages/models/google_drive"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	nas_storage "postgresus-backend/internal/features/storages/models/nas"
	s3_storage "postgresus-backend/internal/features/storages/models/s3"
	"postgresus-backend/internal/util/encryption"

	"github.com/google/uuid"
)

type Storage struct {
	ID            uuid.UUID   `json:"id"            gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	WorkspaceID   uuid.UUID   `json:"workspaceId"   gorm:"column:workspace_id;not null;type:uuid;index"`
	Type          StorageType `json:"type"          gorm:"column:type;not null;type:text"`
	Name          string      `json:"name"          gorm:"column:name;not null;type:text"`
	LastSaveError *string     `json:"lastSaveError" gorm:"column:last_save_error;type:text"`

	// specific storage
	LocalStorage       *local_storage.LocalStorage              `json:"localStorage"       gorm:"foreignKey:StorageID"`
	S3Storage          *s3_storage.S3Storage                    `json:"s3Storage"          gorm:"foreignKey:StorageID"`
	GoogleDriveStorage *google_drive_storage.GoogleDriveStorage `json:"googleDriveStorage" gorm:"foreignKey:StorageID"`
	NASStorage         *nas_storage.NASStorage                  `json:"nasStorage"         gorm:"foreignKey:StorageID"`
	AzureBlobStorage   *azure_blob_storage.AzureBlobStorage     `json:"azureBlobStorage"   gorm:"foreignKey:StorageID"`
}

func (s *Storage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	err := s.getSpecificStorage().SaveFile(ctx, encryptor, logger, fileID, file)
	if err != nil {
		lastSaveError := err.Error()
		s.LastSaveError = &lastSaveError
		return err
	}

	s.LastSaveError = nil

	return nil
}

func (s *Storage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	return s.getSpecificStorage().GetFile(encryptor, fileID)
}

func (s *Storage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	return s.getSpecificStorage().DeleteFile(encryptor, fileID)
}

func (s *Storage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.Type == "" {
		return errors.New("storage type is required")
	}

	if s.Name == "" {
		return errors.New("storage name is required")
	}

	return s.getSpecificStorage().Validate(encryptor)
}

func (s *Storage) TestConnection(encryptor encryption.FieldEncryptor) error {
	return s.getSpecificStorage().TestConnection(encryptor)
}

func (s *Storage) HideSensitiveData() {
	s.getSpecificStorage().HideSensitiveData()
}

func (s *Storage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	return s.getSpecificStorage().EncryptSensitiveData(encryptor)
}

func (s *Storage) Update(incoming *Storage) {
	s.Name = incoming.Name
	s.Type = incoming.Type

	switch s.Type {
	case StorageTypeLocal:
		if s.LocalStorage != nil && incoming.LocalStorage != nil {
			s.LocalStorage.Update(incoming.LocalStorage)
		}
	case StorageTypeS3:
		if s.S3Storage != nil && incoming.S3Storage != nil {
			s.S3Storage.Update(incoming.S3Storage)
		}
	case StorageTypeGoogleDrive:
		if s.GoogleDriveStorage != nil && incoming.GoogleDriveStorage != nil {
			s.GoogleDriveStorage.Update(incoming.GoogleDriveStorage)
		}
	case StorageTypeNAS:
		if s.NASStorage != nil && incoming.NASStorage != nil {
			s.NASStorage.Update(incoming.NASStorage)
		}
	case StorageTypeAzureBlob:
		if s.AzureBlobStorage != nil && incoming.AzureBlobStorage != nil {
			s.AzureBlobStorage.Update(incoming.AzureBlobStorage)
		}
	}
}

func (s *Storage) getSpecificStorage() StorageFileSaver {
	switch s.Type {
	case StorageTypeLocal:
		return s.LocalStorage
	case StorageTypeS3:
		return s.S3Storage
	case StorageTypeGoogleDrive:
		return s.GoogleDriveStorage
	case StorageTypeNAS:
		return s.NASStorage
	case StorageTypeAzureBlob:
		return s.AzureBlobStorage
	default:
		panic("invalid storage type: " + string(s.Type))
	}
}
