package storages

import (
	db "postgresus-backend/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StorageRepository struct{}

func (r *StorageRepository) Save(storage *Storage) (*Storage, error) {
	database := db.GetDb()

	err := database.Transaction(func(tx *gorm.DB) error {
		switch storage.Type {
		case StorageTypeLocal:
			if storage.LocalStorage != nil {
				storage.LocalStorage.StorageID = storage.ID
			}
		case StorageTypeS3:
			if storage.S3Storage != nil {
				storage.S3Storage.StorageID = storage.ID
			}
		case StorageTypeGoogleDrive:
			if storage.GoogleDriveStorage != nil {
				storage.GoogleDriveStorage.StorageID = storage.ID
			}
		case StorageTypeNAS:
			if storage.NASStorage != nil {
				storage.NASStorage.StorageID = storage.ID
			}
		case StorageTypeAzureBlob:
			if storage.AzureBlobStorage != nil {
				storage.AzureBlobStorage.StorageID = storage.ID
			}
		}

		if storage.ID == uuid.Nil {
			if err := tx.Create(storage).
				Omit("LocalStorage", "S3Storage", "GoogleDriveStorage", "NASStorage", "AzureBlobStorage").
				Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(storage).
				Omit("LocalStorage", "S3Storage", "GoogleDriveStorage", "NASStorage", "AzureBlobStorage").
				Error; err != nil {
				return err
			}
		}

		switch storage.Type {
		case StorageTypeLocal:
			if storage.LocalStorage != nil {
				storage.LocalStorage.StorageID = storage.ID // Ensure ID is set
				if err := tx.Save(storage.LocalStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeS3:
			if storage.S3Storage != nil {
				storage.S3Storage.StorageID = storage.ID // Ensure ID is set
				if err := tx.Save(storage.S3Storage).Error; err != nil {
					return err
				}
			}
		case StorageTypeGoogleDrive:
			if storage.GoogleDriveStorage != nil {
				storage.GoogleDriveStorage.StorageID = storage.ID // Ensure ID is set
				if err := tx.Save(storage.GoogleDriveStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeNAS:
			if storage.NASStorage != nil {
				storage.NASStorage.StorageID = storage.ID // Ensure ID is set
				if err := tx.Save(storage.NASStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeAzureBlob:
			if storage.AzureBlobStorage != nil {
				storage.AzureBlobStorage.StorageID = storage.ID // Ensure ID is set
				if err := tx.Save(storage.AzureBlobStorage).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (r *StorageRepository) FindByID(id uuid.UUID) (*Storage, error) {
	var s Storage

	if err := db.
		GetDb().
		Preload("LocalStorage").
		Preload("S3Storage").
		Preload("GoogleDriveStorage").
		Preload("NASStorage").
		Preload("AzureBlobStorage").
		Where("id = ?", id).
		First(&s).Error; err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *StorageRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]*Storage, error) {
	var storages []*Storage

	if err := db.
		GetDb().
		Preload("LocalStorage").
		Preload("S3Storage").
		Preload("GoogleDriveStorage").
		Preload("NASStorage").
		Preload("AzureBlobStorage").
		Where("workspace_id = ?", workspaceID).
		Order("name ASC").
		Find(&storages).Error; err != nil {
		return nil, err
	}

	return storages, nil
}

func (r *StorageRepository) Delete(s *Storage) error {
	return db.GetDb().Transaction(func(tx *gorm.DB) error {
		// Delete specific storage based on type
		switch s.Type {
		case StorageTypeLocal:
			if s.LocalStorage != nil {
				if err := tx.Delete(s.LocalStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeS3:
			if s.S3Storage != nil {
				if err := tx.Delete(s.S3Storage).Error; err != nil {
					return err
				}
			}
		case StorageTypeGoogleDrive:
			if s.GoogleDriveStorage != nil {
				if err := tx.Delete(s.GoogleDriveStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeNAS:
			if s.NASStorage != nil {
				if err := tx.Delete(s.NASStorage).Error; err != nil {
					return err
				}
			}
		case StorageTypeAzureBlob:
			if s.AzureBlobStorage != nil {
				if err := tx.Delete(s.AzureBlobStorage).Error; err != nil {
					return err
				}
			}
		}

		// Delete the main storage
		return tx.Delete(s).Error
	})
}
