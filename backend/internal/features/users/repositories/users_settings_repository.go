package users_repositories

import (
	user_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/storage"
)

type UsersSettingsRepository struct{}

func (r *UsersSettingsRepository) GetSettings() (*user_models.UsersSettings, error) {
	var settings user_models.UsersSettings

	err := storage.GetDb().First(&settings).Error
	return &settings, err
}

func (r *UsersSettingsRepository) CreateSettings(settings *user_models.UsersSettings) error {
	return storage.GetDb().Create(settings).Error
}

func (r *UsersSettingsRepository) UpdateSettings(settings *user_models.UsersSettings) error {
	existingSettings, err := r.GetSettings()
	if err != nil {
		return err
	}

	settings.ID = existingSettings.ID

	return storage.GetDb().Save(settings).Error
}
