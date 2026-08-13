package postgresql_logical

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

const unreachableBastionTimeout = 2 * time.Second

func enabledTunnelDatabase() *PostgresqlLogicalDatabase {
	databaseName := "testdb"

	return &PostgresqlLogicalDatabase{
		Host:     "db.internal",
		Port:     5432,
		Username: "testuser",
		Password: "testpassword",
		Database: &databaseName,
		CpuCount: 1,
		SshTunnel: sshtunnel.Config{
			IsEnabled: true,
			Host:      "bastion.example.com",
			Port:      22,
			Username:  "tunneluser",
			AuthType:  sshtunnel.AuthTypePassword,
			Password:  "tunnelpassword",
		},
	}
}

func bastionedTunnelDatabase(t *testing.T) *PostgresqlLogicalDatabase {
	t.Helper()

	bastion := containers.StartSshBastion(t)

	database := enabledTunnelDatabase()
	database.SshTunnel.Host = bastion.Host
	database.SshTunnel.Port = bastion.Port
	database.SshTunnel.Username = containers.SshBastionUsername
	database.SshTunnel.Password = containers.SshBastionPassword

	return database
}

func Test_OpenTunnel_WhenTheTunnelIsDisabled_HandsBackTheSameDatabase(t *testing.T) {
	database := enabledTunnelDatabase()
	database.SshTunnel.IsEnabled = false

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	assert.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.Same(t, database, tunneledDatabase.GetDatabaseThroughTunnel())
}

func Test_OpenTunnel_WhenTheBastionIsUnreachable_ReturnsAnError(t *testing.T) {
	database := enabledTunnelDatabase()
	// RFC 5737 TEST-NET-1: routable nowhere, so this fails to connect rather than resolving.
	database.SshTunnel.Host = "192.0.2.1"

	// A blackholing network would otherwise burn the forwarder's full 30s dial timeout here.
	ctx, cancel := context.WithTimeout(t.Context(), unreachableBastionTimeout)
	defer cancel()

	_, err := OpenTunnel(ctx, OpenTunnelSpec{Database: database})

	assert.Error(t, err)
}

// PostgreSQL is the one engine whose copy keeps the real host: sslmode=verify-full matches the
// certificate against it, and libpq looks up .pgpass by host and port.
func Test_OpenTunnel_WhenTunneled_KeepsTheRealHostAndCarriesTheForwardedPort(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	assert.Equal(t, "db.internal", databaseThroughTunnel.Host)
	require.NotNil(t, databaseThroughTunnel.LocalTunnelEndpoint)
	assert.Equal(t, "127.0.0.1", databaseThroughTunnel.LocalTunnelEndpoint.Host)
	assert.Equal(t, databaseThroughTunnel.LocalTunnelEndpoint.Port, databaseThroughTunnel.Port)
	assert.NotEqual(t, database.Port, databaseThroughTunnel.Port)
}

func Test_OpenTunnel_WhenTunneled_DisablesTheTunnelFlagOnTheCopy(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.False(t, tunneledDatabase.GetDatabaseThroughTunnel().SshTunnel.IsEnabled,
		"a re-read of the copy would otherwise open a second tunnel into the first one")
	assert.True(t, database.SshTunnel.IsEnabled, "the original must keep the stored configuration")
	assert.Equal(t, 5432, database.Port, "the original address must not be rewritten")
	assert.Nil(t, database.LocalTunnelEndpoint)
}
