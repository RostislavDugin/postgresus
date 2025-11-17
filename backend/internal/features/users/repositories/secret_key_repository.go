package users_repositories

import (
	"errors"
	user_models "postgresus-backend/internal/features/users/models"
	"postgresus-backend/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SecretKeyRepository struct{}

func (r *SecretKeyRepository) GetSecretKey() (string, error) {
	var secretKey user_models.SecretKey

	if err := storage.GetDb().First(&secretKey).Error; err != nil {
		// create a new secret key if not found
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newSecretKey := user_models.SecretKey{
				Secret: uuid.New().String() + uuid.New().String(),
			}
			if err := storage.GetDb().Create(&newSecretKey).Error; err != nil {
				return "", errors.New("failed to create new secret key")
			}

			return newSecretKey.Secret, nil
		}

		return "", err
	}

	return secretKey.Secret, nil
}
