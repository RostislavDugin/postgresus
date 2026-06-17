package api_keys

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type ApiKeyRepository struct{}

func (r *ApiKeyRepository) Create(apiKey *ApiKey, workspaceIDs []uuid.UUID) error {
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}

	return storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Workspaces").Create(apiKey).Error; err != nil {
			return err
		}

		for _, workspaceID := range workspaceIDs {
			grant := &ApiKeyWorkspace{ApiKeyID: apiKey.ID, WorkspaceID: workspaceID}
			if err := tx.Create(grant).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ApiKeyRepository) FindByHashedToken(hashedToken string) (*ApiKey, error) {
	var apiKey ApiKey

	if err := storage.
		GetDb().
		Preload("Workspaces").
		Where("hashed_token = ?", hashedToken).
		First(&apiKey).Error; err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *ApiKeyRepository) FindAllActive() ([]*ApiKey, error) {
	var apiKeys []*ApiKey

	if err := storage.
		GetDb().
		Preload("Workspaces").
		Where("revoked_at IS NULL").
		Order("created_at DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	return apiKeys, nil
}

func (r *ApiKeyRepository) Revoke(id uuid.UUID, revokedAt time.Time) error {
	result := storage.
		GetDb().
		Model(&ApiKey{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrApiKeyNotFound
	}

	return nil
}

func (r *ApiKeyRepository) TouchLastUsed(id uuid.UUID, usedAt time.Time) error {
	return storage.
		GetDb().
		Model(&ApiKey{}).
		Where("id = ?", id).
		Update("last_used_at", usedAt).Error
}

func (r *ApiKeyRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&ApiKey{}, "id = ?", id).Error
}
