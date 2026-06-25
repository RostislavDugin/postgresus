package verification_runs

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
)

const jobName = "verification_scheduler"

var (
	schedulerTickInterval = 15 * time.Second
	maxPendingDuration    = 24 * time.Hour
)

type VerificationScheduler struct {
	repo                     *VerificationRepository
	service                  *VerificationService
	verificaionConfigService *verification_config.VerificationConfigService
	agentService             *verification_agents.AgentService
	backupService            *backups_services.LogicalBackupService
	databaseService          *databases.DatabaseService
	logger                   *slog.Logger

	hasRun atomic.Bool
}

func (s *VerificationScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	ticker := time.NewTicker(schedulerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickLogger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

			if err := s.createScheduledRuns(); err != nil {
				tickLogger.Error("createScheduledRuns failed", "error", err)
			}

			if err := s.reapStaleRuns(); err != nil {
				tickLogger.Error("reapStaleRuns failed", "error", err)
			}

			if err := s.sweepCanceledByDisabledConfig(); err != nil {
				tickLogger.Error("sweepCanceledByDisabledConfig failed", "error", err)
			}
		}
	}
}

func (s *VerificationScheduler) createScheduledRuns() error {
	configsWithEnabledVerifications, err := s.verificaionConfigService.ListEnabled()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, config := range configsWithEnabledVerifications {
		if err := s.createScheduledRunForConfig(config, now); err != nil {
			s.logger.Error(
				"failed to evaluate scheduled run for database",
				"error", err,
				"database_id", config.DatabaseID,
			)
		}
	}

	return nil
}

func (s *VerificationScheduler) createScheduledRunForConfig(
	config *verification_config.BackupVerificationConfig,
	now time.Time,
) error {
	// After-backup configs are driven by the backup-completion listener
	if config.ScheduleType == verification_config.VerificationScheduleAfterBackup {
		return nil
	}

	existing, err := s.repo.FindByDatabaseID(config.DatabaseID)
	if err != nil {
		return err
	}

	for _, row := range existing {
		if row.Trigger != VerificationTriggerScheduled {
			continue
		}

		if row.Status == VerificationStatusPending || row.Status == VerificationStatusRunning {
			return nil
		}
	}

	lastFinishedAt, err := s.repo.FindLatestFinishedAt(config.DatabaseID)
	if err != nil {
		return err
	}

	if !config.VerificationInterval.ShouldTriggerBackup(now, lastFinishedAt) {
		return nil
	}

	backup, err := s.backupService.GetLatestVerifiableBackup(config.DatabaseID)
	if err != nil {
		return err
	}

	if backup == nil {
		s.logger.Info(
			"skipping scheduled verification: no verifiable backup yet",
			"database_id", config.DatabaseID,
		)

		return nil
	}

	database, err := s.databaseService.GetDatabaseByID(config.DatabaseID)
	if err != nil {
		return err
	}

	if err := validateDatabaseIsVerifiable(database); err != nil {
		s.logger.Warn(
			"skipping scheduled verification: database not verifiable",
			"database_id", config.DatabaseID,
			"error", err,
		)

		return nil
	}

	verification := &RestoreVerification{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		BackupID:     backup.ID,
		Trigger:      VerificationTriggerScheduled,
		Status:       VerificationStatusPending,
		AttemptCount: 1,
		CreatedAt:    now,
	}

	return s.repo.Create(verification)
}

func (s *VerificationScheduler) reapStaleRuns() error {
	runningRows, err := s.repo.FindAllRunning()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	staleBefore := now.Add(-StaleAgentThreshold)

	for _, row := range runningRows {
		if row.AgentID == nil {
			continue
		}

		agent, lookupErr := s.agentService.GetAgentByID(*row.AgentID)
		if lookupErr != nil {
			s.logger.Error(
				"failed to load agent during reap",
				"error", lookupErr,
				"verification_id", row.ID,
				"agent_id", row.AgentID,
			)

			continue
		}

		if agent == nil || agent.IsDeleted() {
			// The owning agent is gone, but Requeue clears agent_id so a
			// different agent can pick the run up. If none does, the stale
			// PENDING reaper retires it after maxPendingDuration.
			if failErr := s.service.RequeueOrFail(row, FailureReasonAgentRemoved, nil); failErr != nil {
				s.logger.Error(
					"failed to requeue or fail RUNNING verification for removed agent",
					"error", failErr,
					"verification_id", row.ID,
				)
			}

			continue
		}

		if agent.LastSeenAt == nil || agent.LastSeenAt.Before(staleBefore) {
			if failErr := s.service.RequeueOrFail(row, FailureReasonAgentLostContact, nil); failErr != nil {
				s.logger.Error(
					"failed to requeue or fail stale RUNNING verification",
					"error", failErr,
					"verification_id", row.ID,
				)
			}
		}
	}

	stalePendingBefore := now.Add(-maxPendingDuration)

	stalePending, err := s.repo.FindStalePending(stalePendingBefore)
	if err != nil {
		return err
	}

	for _, row := range stalePending {
		if failErr := s.service.RequeueOrFail(row, FailureReasonUnclaimedTooLong, nil); failErr != nil {
			s.logger.Error(
				"failed to fail stale PENDING verification",
				"error", failErr,
				"verification_id", row.ID,
			)
		}
	}

	return nil
}

// sweepCanceledByDisabledConfig sends no notification
// — disable is user-initiated, not a failure.
func (s *VerificationScheduler) sweepCanceledByDisabledConfig() error {
	rows, err := s.repo.FindNonTerminalForDisabledConfigs()
	if err != nil {
		return err
	}

	for _, row := range rows {
		if cancelErr := s.repo.MarkTerminal(nil, row.ID, VerificationStatusCanceled, map[string]any{
			"fail_message": cancelMessageScheduleDisabled,
		}); cancelErr != nil {
			s.logger.Error(
				"failed to mark verification CANCELED",
				"error", cancelErr,
				"verification_id", row.ID,
			)
		}
	}

	return nil
}
