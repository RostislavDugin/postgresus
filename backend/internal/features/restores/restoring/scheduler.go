package restoring

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config "databasus-backend/internal/features/backups/config"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	cache_utils "databasus-backend/internal/util/cache"
)

const (
	schedulerStartupDelay         = 1 * time.Minute
	schedulerTickerInterval       = 1 * time.Minute
	schedulerHealthcheckThreshold = 5 * time.Minute
)

type RestoresScheduler struct {
	restoreRepository        *restores_core.RestoreRepository
	backupService            *backups_services.BackupService
	storageService           *storages.StorageService
	backupConfigService      *backups_config.BackupConfigService
	restoreNodesRegistry     *RestoreNodesRegistry
	lastCheckTime            time.Time
	logger                   *slog.Logger
	restoreToNodeRelations   map[uuid.UUID]RestoreToNodeRelation
	restorerNode             *RestorerNode
	cacheUtil                *cache_utils.CacheUtil[RestoreDatabaseCache]
	completionSubscriptionID uuid.UUID

	hasRun atomic.Bool
}

func (s *RestoresScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.lastCheckTime = time.Now().UTC()

	if config.GetEnv().IsManyNodesMode {
		// wait other nodes to start
		time.Sleep(schedulerStartupDelay)
	}

	if err := s.failRestoresInProgress(); err != nil {
		s.logger.Error("Failed to fail restores in progress", "error", err)
		panic(err)
	}

	err := s.restoreNodesRegistry.SubscribeForRestoresCompletions(s.onRestoreCompleted)
	if err != nil {
		s.logger.Error("Failed to subscribe to restore completions", "error", err)
		panic(err)
	}

	defer func() {
		if err := s.restoreNodesRegistry.UnsubscribeForRestoresCompletions(); err != nil {
			s.logger.Error("Failed to unsubscribe from restore completions", "error", err)
		}
	}()

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
			if err := s.checkDeadNodesAndFailRestores(); err != nil {
				s.logger.Error("Failed to check dead nodes and fail restores", "error", err)
			}

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
			PostgresqlDatabase: restore.PostgresqlDatabase,
			MysqlDatabase:      restore.MysqlDatabase,
			MariadbDatabase:    restore.MariadbDatabase,
			MongodbDatabase:    restore.MongodbDatabase,
			ClickhouseDatabase: restore.ClickhouseDatabase,
		}
	}

	// Cache database credentials with 1-hour expiration
	s.cacheUtil.SetWithExpiration(restoreID.String(), dbCache, 1*time.Hour)

	leastBusyNodeID, err := s.calculateLeastBusyNode()
	if err != nil {
		s.logger.Error(
			"Failed to calculate least busy node",
			"restoreId",
			restoreID,
			"error",
			err,
		)
		return err
	}

	if err := s.restoreNodesRegistry.IncrementRestoresInProgress(*leastBusyNodeID); err != nil {
		s.logger.Error(
			"Failed to increment restores in progress",
			"nodeId",
			leastBusyNodeID,
			"restoreId",
			restoreID,
			"error",
			err,
		)
		return err
	}

	if err := s.restoreNodesRegistry.AssignRestoreToNode(*leastBusyNodeID, restoreID, false); err != nil {
		s.logger.Error(
			"Failed to submit restore",
			"nodeId",
			leastBusyNodeID,
			"restoreId",
			restoreID,
			"error",
			err,
		)
		if decrementErr := s.restoreNodesRegistry.DecrementRestoresInProgress(*leastBusyNodeID); decrementErr != nil {
			s.logger.Error(
				"Failed to decrement restores in progress after submit failure",
				"nodeId",
				leastBusyNodeID,
				"error",
				decrementErr,
			)
		}
		return err
	}

	if relation, exists := s.restoreToNodeRelations[*leastBusyNodeID]; exists {
		relation.RestoreIDs = append(relation.RestoreIDs, restoreID)
		s.restoreToNodeRelations[*leastBusyNodeID] = relation
	} else {
		s.restoreToNodeRelations[*leastBusyNodeID] = RestoreToNodeRelation{
			NodeID:     *leastBusyNodeID,
			RestoreIDs: []uuid.UUID{restoreID},
		}
	}

	s.logger.Info(
		"Successfully triggered restore",
		"restoreId",
		restoreID,
		"nodeId",
		leastBusyNodeID,
	)

	return nil
}

func (s *RestoresScheduler) calculateLeastBusyNode() (*uuid.UUID, error) {
	nodes, err := s.restoreNodesRegistry.GetAvailableNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	stats, err := s.restoreNodesRegistry.GetRestoreNodesStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get restore nodes stats: %w", err)
	}

	statsMap := make(map[uuid.UUID]int)
	for _, stat := range stats {
		statsMap[stat.ID] = stat.ActiveRestores
	}

	var bestNode *RestoreNode
	var bestScore float64 = -1

	for i := range nodes {
		node := &nodes[i]

		activeRestores := statsMap[node.ID]

		var score float64
		if node.ThroughputMBs > 0 {
			score = float64(activeRestores) / float64(node.ThroughputMBs)
		} else {
			score = float64(activeRestores) * 1000
		}

		if bestNode == nil || score < bestScore {
			bestNode = node
			bestScore = score
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("no suitable nodes available")
	}

	return &bestNode.ID, nil
}

func (s *RestoresScheduler) onRestoreCompleted(nodeID, restoreID uuid.UUID) {
	// Verify this task is actually a restore (registry contains multiple task types)
	_, err := s.restoreRepository.FindByID(restoreID)
	if err != nil {
		// Not a restore task, ignore it
		return
	}

	relation, exists := s.restoreToNodeRelations[nodeID]
	if !exists {
		s.logger.Warn(
			"Received completion for unknown node",
			"nodeId",
			nodeID,
			"restoreId",
			restoreID,
		)
		return
	}

	newRestoreIDs := make([]uuid.UUID, 0)
	found := false
	for _, id := range relation.RestoreIDs {
		if id == restoreID {
			found = true
			continue
		}
		newRestoreIDs = append(newRestoreIDs, id)
	}

	if !found {
		s.logger.Warn(
			"Restore not found in node's restore list",
			"nodeId",
			nodeID,
			"restoreId",
			restoreID,
		)
		return
	}

	if len(newRestoreIDs) == 0 {
		delete(s.restoreToNodeRelations, nodeID)
	} else {
		relation.RestoreIDs = newRestoreIDs
		s.restoreToNodeRelations[nodeID] = relation
	}

	if err := s.restoreNodesRegistry.DecrementRestoresInProgress(nodeID); err != nil {
		s.logger.Error(
			"Failed to decrement restores in progress",
			"nodeId",
			nodeID,
			"restoreId",
			restoreID,
			"error",
			err,
		)
	}
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

func (s *RestoresScheduler) checkDeadNodesAndFailRestores() error {
	nodes, err := s.restoreNodesRegistry.GetAvailableNodes()
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	aliveNodeIDs := make(map[uuid.UUID]bool)
	for _, node := range nodes {
		aliveNodeIDs[node.ID] = true
	}

	for nodeID, relation := range s.restoreToNodeRelations {
		if aliveNodeIDs[nodeID] {
			continue
		}

		s.logger.Warn(
			"Node is dead, failing its restores",
			"nodeId",
			nodeID,
			"restoreCount",
			len(relation.RestoreIDs),
		)

		for _, restoreID := range relation.RestoreIDs {
			restore, err := s.restoreRepository.FindByID(restoreID)
			if err != nil {
				s.logger.Error(
					"Failed to find restore for dead node",
					"nodeId",
					nodeID,
					"restoreId",
					restoreID,
					"error",
					err,
				)
				continue
			}

			failMessage := "Restore failed due to node unavailability"
			restore.FailMessage = &failMessage
			restore.Status = restores_core.RestoreStatusFailed

			if err := s.restoreRepository.Save(restore); err != nil {
				s.logger.Error(
					"Failed to save failed restore for dead node",
					"nodeId",
					nodeID,
					"restoreId",
					restoreID,
					"error",
					err,
				)
				continue
			}

			if err := s.restoreNodesRegistry.DecrementRestoresInProgress(nodeID); err != nil {
				s.logger.Error(
					"Failed to decrement restores in progress for dead node",
					"nodeId",
					nodeID,
					"restoreId",
					restoreID,
					"error",
					err,
				)
			}

			s.logger.Info(
				"Failed restore due to dead node",
				"nodeId",
				nodeID,
				"restoreId",
				restoreID,
			)
		}

		delete(s.restoreToNodeRelations, nodeID)
	}

	return nil
}
