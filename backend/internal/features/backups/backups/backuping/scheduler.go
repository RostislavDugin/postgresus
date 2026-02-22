package backuping

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	files_utils "databasus-backend/internal/util/files"
)

const (
	schedulerStartupDelay         = 1 * time.Minute
	schedulerTickerInterval       = 1 * time.Minute
	schedulerHealthcheckThreshold = 5 * time.Minute
)

type BackupsScheduler struct {
	backupRepository    *backups_core.BackupRepository
	backupConfigService *backups_config.BackupConfigService
	taskCancelManager   *task_cancellation.TaskCancelManager
	backupNodesRegistry *BackupNodesRegistry
	databaseService     *databases.DatabaseService

	lastBackupTime time.Time
	logger         *slog.Logger

	backupToNodeRelations map[uuid.UUID]BackupToNodeRelation
	backuperNode          *BackuperNode

	runOnce sync.Once
	hasRun  atomic.Bool
}

func (s *BackupsScheduler) Run(ctx context.Context) {
	wasAlreadyRun := s.hasRun.Load()

	s.runOnce.Do(func() {
		s.hasRun.Store(true)

		s.lastBackupTime = time.Now().UTC()

		if config.GetEnv().IsManyNodesMode {
			// wait other nodes to start
			time.Sleep(schedulerStartupDelay)
		}

		if err := s.failBackupsInProgress(); err != nil {
			s.logger.Error("Failed to fail backups in progress", "error", err)
			panic(err)
		}

		err := s.backupNodesRegistry.SubscribeForBackupsCompletions(s.onBackupCompleted)
		if err != nil {
			s.logger.Error("Failed to subscribe to backup completions", "error", err)
			panic(err)
		}

		defer func() {
			if err := s.backupNodesRegistry.UnsubscribeForBackupsCompletions(); err != nil {
				s.logger.Error("Failed to unsubscribe from backup completions", "error", err)
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
				if err := s.checkDeadNodesAndFailBackups(); err != nil {
					s.logger.Error("Failed to check dead nodes and fail backups", "error", err)
				}

				if err := s.runPendingBackups(); err != nil {
					s.logger.Error("Failed to run pending backups", "error", err)
				}

				s.lastBackupTime = time.Now().UTC()
			}
		}
	})

	if wasAlreadyRun {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}
}

func (s *BackupsScheduler) IsSchedulerRunning() bool {
	// if last backup time is more than 5 minutes ago, return false
	return s.lastBackupTime.After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *BackupsScheduler) IsBackupNodesAvailable() bool {
	nodes, err := s.backupNodesRegistry.GetAvailableNodes()
	if err != nil {
		s.logger.Error("Failed to get available nodes for health check", "error", err)
		return false
	}

	return len(nodes) > 0
}

func (s *BackupsScheduler) StartBackup(database *databases.Database, backupConfig *backups_config.BackupConfig, isCallNotifier bool) {
	if backupConfig == nil {
		s.logger.Error("Backup config is nil", "databaseId", database.ID)
		return
	}

	if backupConfig.StorageID == nil {
		s.logger.Error("Backup config storage ID is nil", "databaseId", database.ID)
		return
	}

	// Check for existing in-progress backups
	inProgressBackups, err := s.backupRepository.FindByDatabaseIdAndStatus(
		database.ID,
		backups_core.BackupStatusInProgress,
	)
	if err != nil {
		s.logger.Error(
			"Failed to check for in-progress backups",
			"databaseId",
			database.ID,
			"error",
			err,
		)
		return
	}

	if len(inProgressBackups) > 0 {
		s.logger.Warn(
			"Backup already in progress for database, skipping new backup",
			"databaseId",
			database.ID,
			"existingBackupId",
			inProgressBackups[0].ID,
		)
		return
	}

	leastBusyNodeID, err := s.calculateLeastBusyNode()
	if err != nil {
		s.logger.Error(
			"Failed to calculate least busy node",
			"databaseId",
			backupConfig.DatabaseID,
			"error",
			err,
		)
		return
	}

	backupID := uuid.New()
	timestamp := time.Now().UTC()

	backup := &backups_core.Backup{
		ID: backupID,
		FileName: fmt.Sprintf(
			"%s-%s-%s",
			files_utils.SanitizeFilename(database.Name),
			timestamp.Format("20060102-150405"),
			backupID.String(),
		),
		DatabaseID:     backupConfig.DatabaseID,
		BackupConfigID: &backupConfig.ID,
		StorageID:      *backupConfig.StorageID,
		Status:         backups_core.BackupStatusInProgress,
		BackupSizeMb:   0,
		CreatedAt:      timestamp,
	}

	if err := s.backupRepository.Save(backup); err != nil {
		s.logger.Error(
			"Failed to save backup",
			"databaseId",
			backupConfig.DatabaseID,
			"error",
			err,
		)
		return
	}

	if err := s.backupNodesRegistry.IncrementBackupsInProgress(*leastBusyNodeID); err != nil {
		s.logger.Error(
			"Failed to increment backups in progress",
			"nodeId",
			leastBusyNodeID,
			"backupId",
			backup.ID,
			"error",
			err,
		)
		return
	}

	if err := s.backupNodesRegistry.AssignBackupToNode(*leastBusyNodeID, backup.ID, isCallNotifier); err != nil {
		s.logger.Error(
			"Failed to submit backup",
			"nodeId",
			leastBusyNodeID,
			"backupId",
			backup.ID,
			"error",
			err,
		)
		if decrementErr := s.backupNodesRegistry.DecrementBackupsInProgress(*leastBusyNodeID); decrementErr != nil {
			s.logger.Error(
				"Failed to decrement backups in progress after submit failure",
				"nodeId",
				leastBusyNodeID,
				"error",
				decrementErr,
			)
		}
		return
	}

	if relation, exists := s.backupToNodeRelations[*leastBusyNodeID]; exists {
		relation.BackupsIDs = append(relation.BackupsIDs, backup.ID)
		s.backupToNodeRelations[*leastBusyNodeID] = relation
	} else {
		s.backupToNodeRelations[*leastBusyNodeID] = BackupToNodeRelation{
			*leastBusyNodeID,
			[]uuid.UUID{backup.ID},
		}
	}

	s.logger.Info(
		"Successfully triggered scheduled backup",
		"databaseId",
		backupConfig.DatabaseID,
		"backupId",
		backup.ID,
		"nodeId",
		leastBusyNodeID,
	)
}

// GetRemainedBackupTryCount returns the number of remaining backup tries for a given backup.
// If the backup is not failed or the backup config does not allow retries, it returns 0.
// If the backup is failed and the backup config allows retries, it returns the number of remaining tries.
// If the backup is failed and the backup config does not allow retries, it returns 0.
func (s *BackupsScheduler) GetRemainedBackupTryCount(lastBackup *backups_core.Backup) int {
	if lastBackup == nil {
		return 0
	}

	if lastBackup.Status != backups_core.BackupStatusFailed {
		return 0
	}

	if lastBackup.IsSkipRetry {
		return 0
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByID(lastBackup.DatabaseID)
	if err != nil {
		s.logger.Error("Failed to get backup config by database ID", "error", err)
		return 0
	}

	if !backupConfig.IsRetryIfFailed {
		return 0
	}

	maxFailedTriesCount := backupConfig.MaxFailedTriesCount

	lastBackups, err := s.backupRepository.FindByDatabaseIDWithLimit(
		lastBackup.DatabaseID,
		maxFailedTriesCount,
	)
	if err != nil {
		s.logger.Error("Failed to find last backups by database ID", "error", err)
		return 0
	}

	lastFailedBackups := make([]*backups_core.Backup, 0)

	for _, backup := range lastBackups {
		if backup.Status == backups_core.BackupStatusFailed {
			lastFailedBackups = append(lastFailedBackups, backup)
		}
	}

	return maxFailedTriesCount - len(lastFailedBackups)
}

func (s *BackupsScheduler) runPendingBackups() error {
	enabledBackupConfigs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		if backupConfig.BackupInterval == nil {
			continue
		}

		lastBackup, err := s.backupRepository.FindLastByConfigID(backupConfig.ID)
		if err != nil {
			s.logger.Error(
				"Failed to get last backup for database",
				"databaseId",
				backupConfig.DatabaseID,
				"error",
				err,
			)
			continue
		}

		var lastBackupTime *time.Time
		if lastBackup != nil {
			lastBackupTime = &lastBackup.CreatedAt
		}

		remainedBackupTryCount := s.GetRemainedBackupTryCount(lastBackup)

		if backupConfig.BackupInterval.ShouldTriggerBackup(time.Now().UTC(), lastBackupTime) ||
			remainedBackupTryCount > 0 {
			s.logger.Info(
				"Triggering scheduled backup",
				"databaseId",
				backupConfig.DatabaseID,
				"intervalType",
				backupConfig.BackupInterval.Interval,
			)

			database, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
			if err != nil {
				s.logger.Error("Failed to get database by ID", "error", err)
				continue
			}

			s.StartBackup(database, backupConfig, remainedBackupTryCount == 1)
			continue
		}
	}

	return nil
}

func (s *BackupsScheduler) failBackupsInProgress() error {
	backupsInProgress, err := s.backupRepository.FindByStatus(backups_core.BackupStatusInProgress)
	if err != nil {
		return err
	}

	for _, backup := range backupsInProgress {
		if err := s.taskCancelManager.CancelTask(backup.ID); err != nil {
			s.logger.Error(
				"Failed to cancel backup via task cancel manager",
				"backupId",
				backup.ID,
				"error",
				err,
			)
		}

		backupConfig, err := s.backupConfigService.GetBackupConfigByID(backup.DatabaseID)
		if err != nil {
			s.logger.Error("Failed to get backup config by database ID", "error", err)
			continue
		}

		failMessage := "Backup failed due to application restart"
		backup.FailMessage = &failMessage
		backup.Status = backups_core.BackupStatusFailed
		backup.BackupSizeMb = 0

		s.backuperNode.SendBackupNotification(
			backupConfig,
			backup,
			backups_config.NotificationBackupFailed,
			&failMessage,
		)

		if err := s.backupRepository.Save(backup); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupsScheduler) calculateLeastBusyNode() (*uuid.UUID, error) {
	nodes, err := s.backupNodesRegistry.GetAvailableNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	stats, err := s.backupNodesRegistry.GetBackupNodesStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup nodes stats: %w", err)
	}

	statsMap := make(map[uuid.UUID]int)
	for _, stat := range stats {
		statsMap[stat.ID] = stat.ActiveBackups
	}

	var bestNode *BackupNode
	var bestScore float64 = -1

	for i := range nodes {
		node := &nodes[i]

		activeBackups := statsMap[node.ID]

		var score float64
		if node.ThroughputMBs > 0 {
			score = float64(activeBackups) / float64(node.ThroughputMBs)
		} else {
			score = float64(activeBackups) * 1000
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

func (s *BackupsScheduler) onBackupCompleted(nodeID uuid.UUID, backupID uuid.UUID) {
	// Verify this task is actually a backup (registry contains multiple task types)
	_, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		// Not a backup task, ignore it
		return
	}

	relation, exists := s.backupToNodeRelations[nodeID]
	if !exists {
		s.logger.Warn(
			"Received completion for unknown node",
			"nodeId",
			nodeID,
			"backupId",
			backupID,
		)
		return
	}

	newBackupIDs := make([]uuid.UUID, 0)
	found := false
	for _, id := range relation.BackupsIDs {
		if id == backupID {
			found = true
			continue
		}
		newBackupIDs = append(newBackupIDs, id)
	}

	if !found {
		s.logger.Warn(
			"Backup not found in node's backup list",
			"nodeId",
			nodeID,
			"backupId",
			backupID,
		)
		return
	}

	if len(newBackupIDs) == 0 {
		delete(s.backupToNodeRelations, nodeID)
	} else {
		relation.BackupsIDs = newBackupIDs
		s.backupToNodeRelations[nodeID] = relation
	}

	if err := s.backupNodesRegistry.DecrementBackupsInProgress(nodeID); err != nil {
		s.logger.Error(
			"Failed to decrement backups in progress",
			"nodeId",
			nodeID,
			"backupId",
			backupID,
			"error",
			err,
		)
	}
}

func (s *BackupsScheduler) checkDeadNodesAndFailBackups() error {
	nodes, err := s.backupNodesRegistry.GetAvailableNodes()
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	aliveNodeIDs := make(map[uuid.UUID]bool)
	for _, node := range nodes {
		aliveNodeIDs[node.ID] = true
	}

	for nodeID, relation := range s.backupToNodeRelations {
		if aliveNodeIDs[nodeID] {
			continue
		}

		s.logger.Warn(
			"Node is dead, failing its backups",
			"nodeId",
			nodeID,
			"backupCount",
			len(relation.BackupsIDs),
		)

		for _, backupID := range relation.BackupsIDs {
			backup, err := s.backupRepository.FindByID(backupID)
			if err != nil {
				s.logger.Error(
					"Failed to find backup for dead node",
					"nodeId",
					nodeID,
					"backupId",
					backupID,
					"error",
					err,
				)
				continue
			}

			failMessage := "Backup failed due to node unavailability"
			backup.FailMessage = &failMessage
			backup.Status = backups_core.BackupStatusFailed
			backup.BackupSizeMb = 0

			if err := s.backupRepository.Save(backup); err != nil {
				s.logger.Error(
					"Failed to save failed backup for dead node",
					"nodeId",
					nodeID,
					"backupId",
					backupID,
					"error",
					err,
				)
				continue
			}

			if err := s.backupNodesRegistry.DecrementBackupsInProgress(nodeID); err != nil {
				s.logger.Error(
					"Failed to decrement backups in progress for dead node",
					"nodeId",
					nodeID,
					"backupId",
					backupID,
					"error",
					err,
				)
			}

			s.logger.Info(
				"Failed backup due to dead node",
				"nodeId",
				nodeID,
				"backupId",
				backupID,
			)
		}

		delete(s.backupToNodeRelations, nodeID)
	}

	return nil
}
