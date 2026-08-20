package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
)

// slotCleanupDeadline caps how long we wait on the source PG when a database
// is being removed. If the source is unreachable we log + skip so the
// metadata-level delete still goes through; RunStartupCleanup will not
// recover the WAL streamer slot (which lives until that source disappears),
// but the per-backup slot is recoverable on the next run that touches that
// source. The budget covers terminating a still-attached pg_receivewal and
// waiting for the WAL streamer slot to detach before dropping it, so it is
// larger than a plain query timeout.
const slotCleanupDeadline = 20 * time.Second

// PhysicalSlotCleanupListener drops the per-backup and WAL streamer slots
// owned by a physical PostgreSQL database before the database row is
// removed from the metadata DB. Wired in physical/di.go via
// DatabaseService.AddDbRemoveListener so DeleteDatabase / DeleteForTest
// fire it automatically.
//
// Without this hook, slots would be orphaned on the source PG every time a
// database is removed — pinning WAL forever. RunStartupCleanup catches
// per-backup orphans only for databases that still exist; once the metadata
// row is gone there is no record of which source to scan.
type PhysicalSlotCleanupListener struct {
	databaseService *databases.DatabaseService
	fieldEncryptor  encryption.FieldEncryptor
	logger          *slog.Logger
}

func NewPhysicalSlotCleanupListener(
	databaseService *databases.DatabaseService,
	fieldEncryptor encryption.FieldEncryptor,
	logger *slog.Logger,
) *PhysicalSlotCleanupListener {
	return &PhysicalSlotCleanupListener{
		databaseService: databaseService,
		fieldEncryptor:  fieldEncryptor,
		logger:          logger,
	}
}

func (l *PhysicalSlotCleanupListener) OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error {
	database, err := l.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil
	}

	if database.Type != databases.DatabaseTypePostgresPhysical || database.PostgresqlPhysical == nil {
		return nil
	}

	logger := l.logger.With("database_id", databaseID)

	// The database row is being deleted, so the slot drop must finish even if the caller goes
	// away: RunStartupCleanup can never recover a slot whose database row is gone.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotCleanupDeadline)
	defer cancel()

	// The per-backup slot is keyed by the PostgresqlPhysical ID (see
	// WithBackupSlot / RunStartupCleanup), NOT the parent Database ID — the two are
	// distinct UUIDs. Using databaseID here would drop a name that never existed and
	// leave the real slot orphaned forever (the DB row is about to be deleted, so
	// RunStartupCleanup can never recover it).
	slotName := SlotName(database.PostgresqlPhysical.ID)
	walSlotName := database.PostgresqlPhysical.ReplicationSlotName

	tunneledDatabase, err := postgresql_physical.OpenTunnel(cleanupCtx, postgresql_physical.OpenTunnelSpec{
		Database:  database.PostgresqlPhysical,
		Logger:    logger,
		Encryptor: l.fieldEncryptor,
	})
	if err != nil {
		// Both names go into the message: the row is about to disappear, and with it any record of
		// which cluster to go looking on, so this line is the operator's only handle on them.
		logger.Warn("physical slot cleanup: bastion unreachable, leaving slots orphaned",
			"slot_name", slotName,
			"wal_slot_name", walSlotName,
			"error", err,
		)

		return nil
	}

	defer tunneledDatabase.Close()

	sourceDatabase := tunneledDatabase.GetDatabaseThroughTunnel()

	conn, err := sourceDatabase.OpenInspectionConn(cleanupCtx, l.fieldEncryptor)
	if err != nil {
		logger.Warn("physical slot cleanup: source PG unreachable, leaving slots orphaned",
			"slot_name", slotName,
			"wal_slot_name", walSlotName,
			"error", err,
		)

		return nil
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if dropErr := dropBackupSlotIfExists(cleanupCtx, conn, slotName); dropErr != nil {
		logger.Warn("physical slot cleanup: per-backup slot drop failed", "slot_name", slotName, "error", dropErr)
	}

	if dropErr := sourceDatabase.DropWalSlotForRemoval(cleanupCtx, logger, l.fieldEncryptor); dropErr != nil {
		logger.Warn("physical slot cleanup: WAL streamer slot drop failed",
			"slot_name", walSlotName,
			"error", dropErr,
		)
	}

	return nil
}
