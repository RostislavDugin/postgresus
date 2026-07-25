package usecases_physical_postgresql

import (
	"strings"
	"syscall"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

func Test_NewReceivewalCommand_SetsParentDeathSignalAndApplicationName(t *testing.T) {
	databaseID := uuid.New()
	sourceDB := &postgresql_physical.PostgresqlPhysicalDatabase{
		DatabaseID:          &databaseID,
		Host:                "localhost",
		Port:                5432,
		Username:            "replicator",
		ReplicationSlotName: "slot",
	}

	cmd, err := newReceivewalCommand(t.Context(), receivewalCommandSpec{
		PgBin:    "sh",
		SourceDB: sourceDB,
		Creds:    &postgresql_shared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		WatchDir: t.TempDir(),
		SlotName: "slot",
	})
	require.NoError(t, err)
	require.NotNil(t, cmd.SysProcAttr)
	require.Equal(t, syscall.SIGTERM, cmd.SysProcAttr.Pdeathsig)

	applicationName := "PGAPPNAME=" + receivewalApplicationNamePrefix + databaseID.String()
	require.True(t, strings.Contains(strings.Join(cmd.Env, "\n"), applicationName))
}

func Test_IsFatalReceivewalError_ClassifiesNonRetryableStderr(t *testing.T) {
	fatal := []string{
		"pg_receivewal: error: could not write 16777216 bytes to WAL file: No space left on device",
		`pg_receivewal: error: connection failed: FATAL:  password authentication failed for user "repl"`,
		"FATAL:  no pg_hba.conf entry for replication connection from host 10.0.0.1",
		"could not create archive status file: Permission denied",
		`ERROR:  replication slot "db_slot" is active for PID 4242`,
	}
	for _, stderr := range fatal {
		require.True(t, isFatalReceivewalError([]byte(stderr)), stderr)
	}

	transientStderrs := []string{
		"pg_receivewal: error: could not receive data from WAL stream: server closed the connection unexpectedly",
		"pg_receivewal: error: connection to server failed: Connection refused",
		"",
	}
	for _, stderr := range transientStderrs {
		require.False(t, isFatalReceivewalError([]byte(stderr)), stderr)
	}
}

func Test_IsResumeMismatchError_WhenSegmentAlreadyRemoved_ReturnsTrue(t *testing.T) {
	stderr := "pg_receivewal: error: could not send replication command \"START_REPLICATION\": " +
		"ERROR:  requested WAL segment 0000000100000000E000002F has already been removed"

	require.True(t, isResumeMismatchError([]byte(stderr)))
	require.False(t, isFatalReceivewalError([]byte(stderr)),
		"a realign fixes this, so it must not escalate the streamer to FAILED")
}

func Test_IsResumeMismatchError_WhenTransientStderr_ReturnsFalse(t *testing.T) {
	transientStderrs := []string{
		"pg_receivewal: error: could not receive data from WAL stream: server closed the connection unexpectedly",
		"pg_receivewal: error: connection to server failed: Connection refused",
		"",
	}

	for _, stderr := range transientStderrs {
		require.False(t, isResumeMismatchError([]byte(stderr)), stderr)
	}
}
