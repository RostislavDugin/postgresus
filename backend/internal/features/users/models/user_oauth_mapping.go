package users_models

import (
	"time"

	"github.com/google/uuid"
)

type UserOAuthMapping struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	Provider  string    `gorm:"column:provider"`
	OAuthID   string    `gorm:"column:oauth_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (UserOAuthMapping) TableName() string {
	return "user_oauth_mappings"
}
