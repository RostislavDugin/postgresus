package backuping_physical

import (
	"log/slog"

	"github.com/google/uuid"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
)

// PhysicalBackupCanceller stands a database's in-flight physical backup down: it
// cancels the running backup task (the executor unwinds on the cancelled
// context) and releases the cross-table single-in-flight claim. It is the one
// place that knows how to stop a running FULL/INCR, shared by the config-change
// listener, the database-remove listener, and the user-facing cancel/delete
// endpoints.
type PhysicalBackupCanceller struct {
	inFlightRepo      *physical_repositories.PhysicalInFlightBackupRepository
	taskCancelManager *tasks_cancellation.TaskCancelManager
	logger            *slog.Logger
}

func NewPhysicalBackupCanceller(
	inFlightRepo *physical_repositories.PhysicalInFlightBackupRepository,
	taskCancelManager *tasks_cancellation.TaskCancelManager,
	logger *slog.Logger,
) *PhysicalBackupCanceller {
	return &PhysicalBackupCanceller{inFlightRepo, taskCancelManager, logger}
}

// CancelInFlightForDatabase cancels whatever backup the database currently holds
// in flight, whichever it is. Use it for teardown (config disable, db removal)
// where any running backup must stop. A no claim is a no-op.
func (c *PhysicalBackupCanceller) CancelInFlightForDatabase(databaseID uuid.UUID) {
	logger := c.logger.With("database_id", databaseID)

	claim, err := c.inFlightRepo.FindByDatabaseID(databaseID)
	if err != nil {
		logger.Error("failed to look up in-flight backup for cancel", "error", err)

		return
	}

	if claim == nil {
		return
	}

	c.cancelClaim(logger, databaseID, claim.BackupID)
}

// CancelInFlightBackup cancels the database's in-flight backup only when the
// claim still names backupID. It returns whether a matching claim was found and
// cancelled, so a delete path can tell "I stopped the running backup" from
// "nothing was running for this row". Scoping by backupID avoids stopping a
// newer backup that took the claim after the targeted one finished.
func (c *PhysicalBackupCanceller) CancelInFlightBackup(databaseID, backupID uuid.UUID) (bool, error) {
	claim, err := c.inFlightRepo.FindByDatabaseID(databaseID)
	if err != nil {
		return false, err
	}

	if claim == nil || claim.BackupID != backupID {
		return false, nil
	}

	c.cancelClaim(c.logger.With("database_id", databaseID), databaseID, backupID)

	return true, nil
}

func (c *PhysicalBackupCanceller) cancelClaim(logger *slog.Logger, databaseID, backupID uuid.UUID) {
	if err := c.taskCancelManager.CancelTask(backupID); err != nil {
		logger.Error("failed to cancel in-flight backup task", "backup_id", backupID, "error", err)
	}

	if err := c.inFlightRepo.ReleaseOwned(databaseID, backupID); err != nil {
		logger.Error("failed to release in-flight claim", "backup_id", backupID, "error", err)
	}
}
