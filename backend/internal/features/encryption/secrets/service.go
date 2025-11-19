package secrets

import (
	"errors"
	"fmt"
	"os"

	"postgresus-backend/internal/config"
	user_models "postgresus-backend/internal/features/users/models"
	"postgresus-backend/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SecretKeyService struct {
	cachedKey *string
}

func (s *SecretKeyService) MigrateKeyFromDbToFileIfExist() error {
	var secretKey user_models.SecretKey

	err := storage.GetDb().First(&secretKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to check for secret key in database: %w", err)
	}

	if secretKey.Secret == "" {
		return nil
	}

	secretKeyPath := config.GetEnv().SecretKeyPath
	if err := os.WriteFile(secretKeyPath, []byte(secretKey.Secret), 0600); err != nil {
		return fmt.Errorf("failed to write secret key to file: %w", err)
	}

	if err := storage.GetDb().Exec("DELETE FROM secret_keys").Error; err != nil {
		return fmt.Errorf("failed to delete secret key from database: %w", err)
	}

	return nil
}

func (s *SecretKeyService) GetSecretKey() (string, error) {
	if s.cachedKey != nil {
		return *s.cachedKey, nil
	}

	secretKeyPath := config.GetEnv().SecretKeyPath
	data, err := os.ReadFile(secretKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			newKey := s.generateNewSecretKey()
			if err := os.WriteFile(secretKeyPath, []byte(newKey), 0600); err != nil {
				return "", fmt.Errorf("failed to write new secret key: %w", err)
			}
			s.cachedKey = &newKey
			return newKey, nil
		}
		return "", fmt.Errorf("failed to read secret key file: %w", err)
	}

	key := string(data)
	s.cachedKey = &key
	return key, nil
}

func (s *SecretKeyService) generateNewSecretKey() string {
	return uuid.New().String() + uuid.New().String()
}
