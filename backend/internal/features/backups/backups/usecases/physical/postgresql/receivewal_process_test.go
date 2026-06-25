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

	cmd, err := newReceivewalCommand(
		t.Context(),
		"sh",
		sourceDB,
		&postgresql_shared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		t.TempDir(),
		"slot",
	)
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

	transient := []string{
		"pg_receivewal: error: could not receive data from WAL stream: server closed the connection unexpectedly",
		"pg_receivewal: error: connection to server failed: Connection refused",
		"",
	}
	for _, stderr := range transient {
		require.False(t, isFatalReceivewalError([]byte(stderr)), stderr)
	}
}
