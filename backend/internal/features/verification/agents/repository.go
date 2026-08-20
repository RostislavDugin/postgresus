package verification_agents

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type AgentRepository struct{}

func (r *AgentRepository) Create(agent *Agent) error {
	if agent.ID == uuid.Nil {
		agent.ID = uuid.New()
	}

	return storage.GetDb().Create(agent).Error
}

func (r *AgentRepository) FindAll(ctx context.Context) ([]*Agent, error) {
	agents := make([]*Agent, 0)

	err := storage.GetDb().WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&agents).Error

	return agents, err
}

func (r *AgentRepository) CountLive(ctx context.Context) (int64, error) {
	var count int64

	err := storage.GetDb().WithContext(ctx).
		Model(&Agent{}).
		Where("deleted_at IS NULL").
		Count(&count).Error

	return count, err
}

func (r *AgentRepository) FindByID(id uuid.UUID) (*Agent, error) {
	var agent Agent

	err := storage.GetDb().
		Where("id = ?", id).
		First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &agent, nil
}

func (r *AgentRepository) UpdateTokenHash(id uuid.UUID, tokenHash string) error {
	result := storage.GetDb().
		Model(&Agent{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("token_hash", tokenHash)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrAgentNotFound
	}

	return nil
}

func (r *AgentRepository) UpdateCapacityAndLastSeen(
	id uuid.UUID,
	capacity AgentCapacity,
	seenAt time.Time,
) error {
	result := storage.GetDb().
		Model(&Agent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"max_cpu":             capacity.MaxCPU,
			"max_ram_gb":          capacity.MaxRAMGb,
			"max_disk_gb":         capacity.MaxDiskGb,
			"max_concurrent_jobs": capacity.MaxConcurrentJobs,
			"last_seen_at":        seenAt,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrAgentNotFound
	}

	return nil
}

func (r *AgentRepository) SoftDelete(id uuid.UUID, deletedAt time.Time) error {
	result := storage.GetDb().
		Model(&Agent{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", deletedAt)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrAgentNotFound
	}

	return nil
}

func (r *AgentRepository) FindStale(staleBefore time.Time) ([]*Agent, error) {
	agents := make([]*Agent, 0)

	err := storage.GetDb().
		Where("deleted_at IS NULL AND COALESCE(last_seen_at, created_at) < ?", staleBefore).
		Order("created_at DESC").
		Find(&agents).Error

	return agents, err
}
