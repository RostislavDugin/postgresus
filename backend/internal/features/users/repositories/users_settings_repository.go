package users_repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	user_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/storage"
)

type UsersSettingsRepository struct{}

func (r *UsersSettingsRepository) GetSettings(ctx context.Context) (*user_models.UsersSettings, error) {
	var settings user_models.UsersSettings

	if err := storage.GetDb().WithContext(ctx).First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			defaultSettings := &user_models.UsersSettings{
				ID:                                uuid.New(),
				IsAllowExternalRegistrations:      true,
				IsAllowMemberInvitations:          true,
				IsMemberAllowedToCreateWorkspaces: true,
			}

			if createErr := storage.GetDb().WithContext(ctx).Create(defaultSettings).Error; createErr != nil {
				return nil, createErr
			}

			return defaultSettings, nil
		}
		return nil, err
	}

	return &settings, nil
}

func (r *UsersSettingsRepository) UpdateSettings(ctx context.Context, settings *user_models.UsersSettings) error {
	existingSettings, err := r.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.ID = existingSettings.ID

	return storage.GetDb().WithContext(ctx).Save(settings).Error
}
