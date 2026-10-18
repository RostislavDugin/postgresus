package usecases_physical_postgresql_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/util/walmath"
)

func Test_InspectSlot_WithAndWithoutSlotWalLimit_ReportsAllowanceAndCurrentPosition(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	slotName := createReservedTestSlot(t, adminConn)

	var currentLSNBeforeInspect string
	require.NoError(
		t,
		adminConn.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&currentLSNBeforeInspect),
	)

	unlimitedSlotState, err := postgresql_executor.InspectSlot(t.Context(), adminConn, slotName)
	require.NoError(t, err)
	require.NotNil(t, unlimitedSlotState)

	assert.Nil(t, unlimitedSlotState.SafeWalSizeBytes, "max_slot_wal_keep_size = -1 means no limit")

	positionBeforeInspect, err := walmath.ParseLSN(currentLSNBeforeInspect)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, unlimitedSlotState.CurrentLSN, positionBeforeInspect,
		"the position InspectSlot reads must not be below one read before it")

	setSlotWalLimit(t, adminConn, "100GB")

	require.Eventually(t, func() bool {
		limitedSlotState, err := postgresql_executor.InspectSlot(t.Context(), adminConn, slotName)
		if err != nil || limitedSlotState == nil {
			return false
		}

		return limitedSlotState.SafeWalSizeBytes != nil && *limitedSlotState.SafeWalSizeBytes > 0
	}, 15*time.Second, 200*time.Millisecond, "a source with a slot WAL limit reports how much WAL is left")
}

func createReservedTestSlot(t *testing.T, adminConn *pgx.Conn) string {
	t.Helper()

	slotName := "databasus_inspect_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	_, err := adminConn.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(), "SELECT pg_drop_replication_slot($1)", slotName)
	})

	return slotName
}

// The setting is reloadable, so the test restores the default instead of
// leaving a limit behind for the tests that share this source.
func setSlotWalLimit(t *testing.T, adminConn *pgx.Conn, limit string) {
	t.Helper()

	_, err := adminConn.Exec(t.Context(), "ALTER SYSTEM SET max_slot_wal_keep_size = '"+limit+"'")
	require.NoError(t, err)

	_, err = adminConn.Exec(t.Context(), "SELECT pg_reload_conf()")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(), "ALTER SYSTEM RESET max_slot_wal_keep_size")
		_, _ = adminConn.Exec(context.Background(), "SELECT pg_reload_conf()")
	})
}
