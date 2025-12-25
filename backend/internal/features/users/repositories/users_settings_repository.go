package users_repositories

import (
	user_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersSettingsRepository struct{}

func (r *UsersSettingsRepository) GetSettings() (*user_models.UsersSettings, error) {
	var settings user_models.UsersSettings

	if err := storage.GetDb().First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings if none exist
			defaultSettings := &user_models.UsersSettings{
				ID:                                uuid.New(),
				IsAllowExternalRegistrations:      true,
				IsAllowMemberInvitations:          true,
				IsMemberAllowedToCreateWorkspaces: true,
			}

			if createErr := storage.GetDb().Create(defaultSettings).Error; createErr != nil {
				return nil, createErr
			}

			return defaultSettings, nil
		}
		return nil, err
	}

	return &settings, nil
}

func (r *UsersSettingsRepository) UpdateSettings(settings *user_models.UsersSettings) error {
	existingSettings, err := r.GetSettings()
	if err != nil {
		return err
	}

	settings.ID = existingSettings.ID

	return storage.GetDb().Save(settings).Error
}
