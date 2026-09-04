package databases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
)

// Close and CopyDiscoveredMetadataToOriginal delegate through an interface, and a passthrough leaves
// it nil rather than holding a nil-safe pointer, so the guard is the only thing between a disabled
// tunnel and a panic on every deferred Close.
func Test_CloseTunneledDatabase_WhenTheTunnelIsDisabled_DoesNotPanic(t *testing.T) {
	database := &Database{
		Type:  DatabaseTypeMysql,
		Mysql: &mysql.MysqlDatabase{Host: "db.internal", Port: 3306},
	}

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	assert.NotPanics(t, tunneledDatabase.Close)
	assert.NotPanics(t, tunneledDatabase.CopyDiscoveredMetadataToOriginal)
	assert.Same(t, database, tunneledDatabase.GetDatabaseThroughTunnel())
}

func Test_CloseTunneledDatabase_WhenTheDatabaseIsNil_DoesNotPanic(t *testing.T) {
	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{})
	require.NoError(t, err)

	assert.NotPanics(t, tunneledDatabase.Close)
	assert.Nil(t, tunneledDatabase.GetDatabaseThroughTunnel())
}

func Test_OpenTunnel_WithPersistedEmbeddedLogicalTarget_ReturnsError(t *testing.T) {
	testCases := []struct {
		name string
		host string
	}{
		{name: "Unix socket in libpq host list", host: "remote.invalid,/tmp"},
		{name: "decimal loopback", host: "2130706433"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			databaseName := "databasus"
			database := &Database{
				Type: DatabaseTypePostgresLogical,
				PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
					Host:     testCase.host,
					Port:     5437,
					Username: "postgres",
					Password: "stored-password",
					Database: &databaseName,
					CpuCount: 1,
				},
			}

			_, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})

			require.ErrorContains(t, err, "backing up Databasus internal database is not allowed")
		})
	}
}
