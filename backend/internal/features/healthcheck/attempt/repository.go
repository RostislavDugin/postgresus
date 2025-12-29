package healthcheck_attempt

import (
	"databasus-backend/internal/storage"
	"time"

	"github.com/google/uuid"
)

type HealthcheckAttemptRepository struct{}

func (r *HealthcheckAttemptRepository) FindByDatabaseIdOrderByCreatedAtDesc(
	databaseID uuid.UUID,
	afterDate time.Time,
) ([]*HealthcheckAttempt, error) {
	var attempts []*HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Where("created_at > ?", afterDate).
		Order("created_at DESC").
		Find(&attempts).Error; err != nil {
		return nil, err
	}

	return attempts, nil
}

func (r *HealthcheckAttemptRepository) FindLastByDatabaseID(
	databaseID uuid.UUID,
) (*HealthcheckAttempt, error) {
	var attempt HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&attempt).Error; err != nil {
		return nil, err
	}

	return &attempt, nil
}

func (r *HealthcheckAttemptRepository) DeleteOlderThan(
	databaseID uuid.UUID,
	olderThan time.Time,
) error {
	return storage.
		GetDb().
		Where("database_id = ? AND created_at < ?", databaseID, olderThan).
		Delete(&HealthcheckAttempt{}).Error
}

func (r *HealthcheckAttemptRepository) Create(
	attempt *HealthcheckAttempt,
) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}

	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(attempt).Error
}

func (r *HealthcheckAttemptRepository) Insert(
	attempt *HealthcheckAttempt,
) error {
	return r.Create(attempt)
}

func (r *HealthcheckAttemptRepository) FindByDatabaseIDWithLimit(
	databaseID uuid.UUID,
	limit int,
) ([]*HealthcheckAttempt, error) {
	var attempts []*HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&attempts).Error; err != nil {
		return nil, err
	}

	return attempts, nil
}

func (r *HealthcheckAttemptRepository) CountByDatabaseID(
	databaseID uuid.UUID,
) (int64, error) {
	var count int64

	if err := storage.
		GetDb().
		Model(&HealthcheckAttempt{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
