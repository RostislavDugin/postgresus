package download_token

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID `json:"id"        gorm:"column:id;primaryKey"`
	Token     string    `json:"token"     gorm:"column:token;uniqueIndex;not null"`
	BackupID  uuid.UUID `json:"backupId"  gorm:"column:backup_id;not null"`
	UserID    uuid.UUID `json:"userId"    gorm:"column:user_id;not null"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"column:expires_at;not null"`
	Used      bool      `json:"used"      gorm:"column:used;not null;default:false"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;not null"`
}

func (Token) TableName() string {
	return "download_tokens"
}
