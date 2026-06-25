package usecases_physical_postgresql_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

// Test_OnBeforeDatabaseRemove_DropsBothPerBackupAndStreamerSlots is the only
// coverage for the database-removal drop site — the sole place the long-lived
// streamer slot is removed. It also pins that the per-backup slot is keyed by the
// PostgresqlPhysical ID (not the parent Database ID): the two are distinct UUIDs,
// and computing the wrong one would silently orphan the slot forever, since the DB
// row is deleted right after and RunStartupCleanup only scans existing rows.
func Test_OnBeforeDatabaseRemove_DropsBothPerBackupAndStreamerSlots(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	perBackupSlot := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	streamerSlot := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	require.NotEmpty(t, streamerSlot, "streamer slot name should be populated by BeforeCreate")

	require.NoError(t, fixture.DB.PostgresqlPhysical.VerifyWalSlot(
		t.Context(), logger.GetLogger(), encryption.GetFieldEncryptor(),
	))

	// Simulate an orphan per-backup slot a crashed/interrupted backup left behind
	// that neither the next run nor startup cleanup recovered before removal.
	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", perBackupSlot)
	require.NoError(t, err, "pre-create orphan per-backup slot")

	// Defensive: if the listener fails to drop either slot, don't leak it into the
	// shared cluster (max_replication_slots is finite).
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = ANY($1)`,
			[]string{perBackupSlot, streamerSlot})
	})

	require.True(t, postgresql_executor.SlotExists(t, adminConn, perBackupSlot),
		"per-backup slot must exist before removal")
	require.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"streamer slot must exist before removal")

	listener := postgresql_executor.NewPhysicalSlotCleanupListener(
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
	)

	require.NoError(t, listener.OnBeforeDatabaseRemove(fixture.DB.ID))

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, perBackupSlot),
		"per-backup slot (keyed by PostgresqlPhysical.ID) must be dropped on DB removal")
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"WAL streamer slot must be dropped on DB removal")
}

// Test_OnBeforeDatabaseRemove_WhenSourceUnreachable_DoesNotBlockRemoval pins the
// documented degraded path: an unreachable source must not fail the metadata
// delete (the listener logs + returns nil). This is the known G5 limitation — the
// slots stay orphaned on the unreachable source — captured so the behavior is
// intentional, not accidental.
func Test_OnBeforeDatabaseRemove_WhenSourceUnreachable_DoesNotBlockRemoval(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	fixture.DB.PostgresqlPhysical.Host = "127.0.0.1"
	fixture.DB.PostgresqlPhysical.Port = 1

	listener := postgresql_executor.NewPhysicalSlotCleanupListener(
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
	)

	assert.NoError(t, listener.OnBeforeDatabaseRemove(fixture.DB.ID),
		"unreachable source must be logged + skipped, never block the metadata delete")
}

// Test_DropWalSlot_WhenStreamerSlotActive_RefusesAndKeepsSlot pins the safety
// guard the removal listener relies on: DropWalSlot must refuse to drop a slot a
// consumer (our pg_receivewal) still holds, rather than terminate the session —
// dropping an active slot would crash the live streamer.
func Test_DropWalSlot_WhenStreamerSlotActive_RefusesAndKeepsSlot(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	streamerSlot := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	stop := postgresql_executor.StartWalStreamerForTest(t, fixture, fixture.Storage)
	defer stop()

	waitUntilSlotActive(t, adminConn, streamerSlot, 30*time.Second)

	err := fixture.DB.PostgresqlPhysical.DropWalSlot(
		context.Background(), logger.GetLogger(), encryption.GetFieldEncryptor(),
	)
	require.Error(t, err, "DropWalSlot must refuse a slot held by an active consumer")
	assert.Contains(t, err.Error(), "is active")

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"a refused drop must leave the active slot intact")
}

func waitUntilSlotActive(t *testing.T, conn *pgx.Conn, slotName string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		var isActive bool
		err := conn.QueryRow(context.Background(),
			"SELECT COALESCE(active, false) FROM pg_replication_slots WHERE slot_name = $1",
			slotName,
		).Scan(&isActive)
		if err == nil && isActive {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("slot %s did not become active within %s", slotName, timeout)
}
