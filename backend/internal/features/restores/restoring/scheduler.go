package restoring

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	restores_core "databasus-backend/internal/features/restores/core"
	cache_utils "databasus-backend/internal/util/cache"
)

const (
	schedulerTickerInterval       = 1 * time.Minute
	schedulerHealthcheckThreshold = 5 * time.Minute
)

type RestoresScheduler struct {
	restoreRepository *restores_core.RestoreRepository
	lastCheckTime     time.Time
	logger            *slog.Logger
	restorer          *Restorer
	cacheUtil         *cache_utils.CacheUtil[RestoreDatabaseCache]

	hasRun atomic.Bool
}

func (s *RestoresScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.lastCheckTime = time.Now().UTC()

	if err := s.failRestoresInProgress(); err != nil {
		s.logger.Error("Failed to fail restores in progress", "error", err)
		panic(err)
	}

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(schedulerTickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.lastCheckTime = time.Now().UTC()
		}
	}
}

func (s *RestoresScheduler) IsSchedulerRunning() bool {
	return s.lastCheckTime.After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *RestoresScheduler) StartRestore(restoreID uuid.UUID, dbCache *RestoreDatabaseCache) error {
	// If dbCache not provided, try to fetch from DB (for backward compatibility/testing)
	if dbCache == nil {
		restore, err := s.restoreRepository.FindByID(restoreID)
		if err != nil {
			s.logger.Error(
				"Failed to find restore by ID",
				"restoreId",
				restoreID,
				"error",
				err,
			)
			return err
		}

		// Create cache DTO from restore (may be nil if not in DB)
		dbCache = &RestoreDatabaseCache{
			PostgresqlLogicalDatabase: restore.PostgresqlLogicalDatabase,
			MysqlDatabase:             restore.MysqlDatabase,
			MariadbDatabase:           restore.MariadbDatabase,
			MongodbDatabase:           restore.MongodbDatabase,
		}
	}

	// Cache database credentials with 1-hour expiration
	s.cacheUtil.SetWithExpiration(restoreID.String(), dbCache, 1*time.Hour)

	go s.restorer.MakeRestore(restoreID)

	s.logger.Info("Successfully triggered restore", "restoreId", restoreID)

	return nil
}

func (s *RestoresScheduler) failRestoresInProgress() error {
	restoresInProgress, err := s.restoreRepository.FindByStatus(
		restores_core.RestoreStatusInProgress,
	)
	if err != nil {
		return err
	}

	for _, restore := range restoresInProgress {
		failMessage := "Restore failed due to application restart"
		restore.FailMessage = &failMessage
		restore.Status = restores_core.RestoreStatusFailed

		if err := s.restoreRepository.Save(restore); err != nil {
			return err
		}
	}

	return nil
}
