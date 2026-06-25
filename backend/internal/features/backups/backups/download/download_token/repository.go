package download_token

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type Repository struct{}

func (r *Repository) Create(token *Token) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	return storage.GetDb().Create(token).Error
}

func (r *Repository) FindByToken(token string) (*Token, error) {
	var downloadToken Token

	err := storage.GetDb().
		Where("token = ?", token).
		First(&downloadToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &downloadToken, nil
}

func (r *Repository) Update(token *Token) error {
	return storage.GetDb().Save(token).Error
}

func (r *Repository) DeleteExpired(before time.Time) error {
	return storage.GetDb().
		Where("expires_at < ?", before).
		Delete(&Token{}).Error
}
