package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
)

const (
	// slotNamePrefix identifies per-backup slots so RunStartupCleanup can
	// drop them without touching the long-lived streamer slot (which lives
	// under "databasus_slot_<uuid>" — see model.go's BeforeCreate).
	slotNamePrefix = "databasus_basebackup_"

	// startupCleanupTimeout caps how long a single DB's cleanup may block
	// startup. Source PG might be unreachable; we log + skip rather than
	// hang the boot.
	startupCleanupTimeout = 5 * time.Second

	// startupCleanupConcurrency bounds parallel cleanup. With 100+ DBs
	// configured, unbounded fan-out would saturate Databasus's connection
	// pool. 10 keeps total boot delay near the slowest 10% of sources.
	startupCleanupConcurrency = 10
)

// SlotName derives the deterministic per-backup slot name for a database.
func SlotName(dbID uuid.UUID) string {
	return slotNamePrefix + strings.ReplaceAll(dbID.String(), "-", "")
}

// WithBackupSlot creates a fresh per-backup replication slot on the source
// PG, runs fn, then drops the slot. Drop-if-exists at the start picks up
// any orphan from a previous crashed backup; defer drop covers the happy
// path and most failure paths.
func WithBackupSlot(
	ctx context.Context,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fn func() error,
) error {
	conn, err := sourceDB.OpenInspectionConn(ctx, encryptor)
	if err != nil {
		return fmt.Errorf("open conn for backup slot: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	slotName := SlotName(sourceDB.ID)

	if err := dropBackupSlotIfExists(ctx, conn, slotName); err != nil {
		return fmt.Errorf("pre-create drop of backup slot %q: %w", slotName, err)
	}

	if _, err := conn.Exec(ctx,
		"SELECT pg_create_physical_replication_slot($1, true)",
		slotName,
	); err != nil {
		return fmt.Errorf("create backup slot %q: %w", slotName, err)
	}

	logger.Debug("per-backup slot created", "slot_name", slotName)

	defer func() {
		// Background context so defer runs even when ctx is cancelled.
		if dropErr := dropBackupSlotIfExists(context.Background(), conn, slotName); dropErr != nil {
			logger.Warn(
				"post-backup slot drop failed; will be recovered by next backup or startup cleanup",
				"slot_name", slotName,
				"error", dropErr,
			)
		}
	}()

	return fn()
}

func dropBackupSlotIfExists(ctx context.Context, conn *pgx.Conn, slotName string) error {
	_, err := conn.Exec(ctx,
		`SELECT pg_drop_replication_slot(slot_name)
		   FROM pg_replication_slots WHERE slot_name = $1`,
		slotName,
	)

	return err
}

// RunStartupCleanup iterates every physical PG database and drops the per-backup
// replication slot left behind by a crash, EXCEPT where isSlotProtected reports
// the slot is still in use by an in-flight backup — a blind drop would unpin the
// WAL a running pg_basebackup still needs. Runs once at boot.
//
// Cleanup never blocks boot beyond startupCleanupTimeout per DB; unreachable
// sources and protected slots are logged and skipped.
func RunStartupCleanup(
	ctx context.Context,
	logger *slog.Logger,
	isSlotProtected func(databaseID uuid.UUID) (bool, error),
) error {
	dbs, err := databases.GetDatabaseService().GetAllDatabases()
	if err != nil {
		return fmt.Errorf("list databases for slot cleanup: %w", err)
	}

	encryptor := encryption.GetFieldEncryptor()

	sem := make(chan struct{}, startupCleanupConcurrency)

	var (
		wg           sync.WaitGroup
		droppedCount sync.Map
		skippedCount sync.Map
		failureCount sync.Map
	)

	for _, db := range dbs {
		if db.Type != databases.DatabaseTypePostgresPhysical {
			continue
		}

		if db.PostgresqlPhysical == nil {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		// The slot is keyed by the PostgresqlPhysical ID; the in-flight claim is
		// keyed by the Database ID — both are threaded in so the protection check
		// looks up the right claim.
		go func(d *postgresql_physical.PostgresqlPhysicalDatabase, databaseID uuid.UUID) {
			defer wg.Done()
			defer func() { <-sem }()

			slotName := SlotName(d.ID)
			scopedLogger := logger.With("database_id", databaseID, "slot_name", slotName)

			// Decided from the in-flight claim, not the source PG, so it runs
			// before we open a connection.
			protected, protErr := isSlotProtected(databaseID)
			if protErr != nil {
				scopedLogger.Warn("startup slot cleanup: skip, cannot determine owner liveness", "error", protErr)
				skippedCount.Store(databaseID, struct{}{})
				return
			}

			if protected {
				scopedLogger.Info("startup slot cleanup: skip, backup still in flight")
				skippedCount.Store(databaseID, struct{}{})
				return
			}

			cleanupCtx, cancel := context.WithTimeout(ctx, startupCleanupTimeout)
			defer cancel()

			conn, err := d.OpenInspectionConn(cleanupCtx, encryptor)
			if err != nil {
				scopedLogger.Warn("startup slot cleanup: skip unreachable source", "error", err)
				skippedCount.Store(databaseID, struct{}{})
				return
			}
			defer func() { _ = conn.Close(context.Background()) }()

			if err := dropBackupSlotIfExists(cleanupCtx, conn, slotName); err != nil {
				scopedLogger.Warn("startup slot cleanup: drop failed", "error", err)
				failureCount.Store(databaseID, struct{}{})
				return
			}

			droppedCount.Store(databaseID, struct{}{})
		}(db.PostgresqlPhysical, db.ID)
	}

	wg.Wait()

	var dropped, skipped, failed int
	droppedCount.Range(func(_, _ any) bool { dropped++; return true })
	skippedCount.Range(func(_, _ any) bool { skipped++; return true })
	failureCount.Range(func(_, _ any) bool { failed++; return true })

	logger.Info(fmt.Sprintf(
		"startup physical backup slot cleanup complete: %d dropped, %d skipped (unreachable), %d failed",
		dropped, skipped, failed,
	))

	if failed > 0 {
		return errors.New("one or more databases failed startup slot cleanup; see logs")
	}

	return nil
}
