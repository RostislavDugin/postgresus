package users_repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/storage"
)

type UserRepository struct{}

func (r *UserRepository) GetUsersCount() (int64, error) {
	var count int64
	if err := storage.GetDb().Model(&users_models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *UserRepository) CreateUser(user *users_models.User) error {
	return storage.GetDb().Create(user).Error
}

func (r *UserRepository) GetUserByEmail(email string) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(userID uuid.UUID) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUserPassword(userID uuid.UUID, hashedPassword string) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"hashed_password":        hashedPassword,
			"password_creation_time": time.Now().UTC(),
		}).Error
}

func (r *UserRepository) CreateInitialAdmin() error {
	admin, err := r.GetUserByEmail("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin != nil {
		return nil
	}

	admin = &users_models.User{
		ID:                   uuid.New(),
		Email:                "admin",
		Name:                 "Admin",
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleAdmin,
		Status:               users_enums.UserStatusActive,
		CreatedAt:            time.Now().UTC(),
	}

	return storage.GetDb().Create(admin).Error
}

func (r *UserRepository) GetUsers(
	limit, offset int,
	beforeCreatedAt *time.Time,
	query string,
) ([]*users_models.User, int64, error) {
	var users []*users_models.User
	var total int64

	countQuery := storage.GetDb().Model(&users_models.User{})
	if beforeCreatedAt != nil {
		countQuery = countQuery.Where("created_at < ?", *beforeCreatedAt)
	}
	if query != "" {
		searchPattern := "%" + query + "%"
		countQuery = countQuery.Where("email ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := storage.GetDb().
		Limit(limit).
		Offset(offset).
		Order("created_at DESC")

	if beforeCreatedAt != nil {
		dataQuery = dataQuery.Where("created_at < ?", *beforeCreatedAt)
	}
	if query != "" {
		searchPattern := "%" + query + "%"
		dataQuery = dataQuery.Where("email ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	if err := dataQuery.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) UpdateUserStatus(userID uuid.UUID, status users_enums.UserStatus) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"status": status,
		}).Error
}

func (r *UserRepository) UpdateUserRole(userID uuid.UUID, role users_enums.UserRole) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"role": role,
		}).Error
}

func (r *UserRepository) RenameUserEmailForTests(oldEmail, newEmail string) error {
	result := storage.GetDb().Model(&users_models.User{}).
		Where("email = ?", oldEmail).
		Update("email", newEmail)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return nil
	}

	return nil
}

func (r *UserRepository) UpdateUserInfo(userID uuid.UUID, name, email *string) error {
	updates := make(map[string]any)

	if name != nil {
		updates["name"] = *name
	}
	if email != nil {
		updates["email"] = *email
	}

	if len(updates) == 0 {
		return nil
	}

	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

func (r *UserRepository) GetUserByOAuthID(provider, oauthID string) (*users_models.User, error) {
	var user users_models.User

	err := storage.GetDb().
		Joins("JOIN user_oauth_mappings ON user_oauth_mappings.user_id = users.id").
		Where("user_oauth_mappings.provider = ? AND user_oauth_mappings.oauth_id = ?", provider, oauthID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) LinkOAuthToUser(
	userID uuid.UUID,
	provider, oauthID string,
	activateWithName *string,
) error {
	return storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if activateWithName != nil {
			if err := tx.Model(&users_models.User{}).
				Where("id = ?", userID).
				Updates(map[string]any{
					"status": users_enums.UserStatusActive,
					"name":   *activateWithName,
				}).Error; err != nil {
				return err
			}
		}

		mapping := &users_models.UserOAuthMapping{
			UserID:   userID,
			Provider: provider,
			OAuthID:  oauthID,
		}

		return tx.Create(mapping).Error
	})
}

func (r *UserRepository) CreateUserWithOAuth(
	user *users_models.User,
	provider, oauthID string,
) error {
	return storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		mapping := &users_models.UserOAuthMapping{
			UserID:   user.ID,
			Provider: provider,
			OAuthID:  oauthID,
		}

		return tx.Create(mapping).Error
	})
}
