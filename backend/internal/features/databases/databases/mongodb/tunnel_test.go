package mongodb

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

func enabledTunnelDatabase() *MongodbDatabase {
	return &MongodbDatabase{
		Host:               "db.internal",
		Port:               new(27017),
		Username:           "testuser",
		Password:           "testpassword",
		Database:           "testdb",
		AuthDatabase:       "admin",
		CpuCount:           1,
		IsDirectConnection: false,
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

func bastionedTunnelDatabase(t *testing.T) *MongodbDatabase {
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

func Test_OpenTunnel_WhenTunneledWithoutAPort_ReturnsAnError(t *testing.T) {
	database := enabledTunnelDatabase()
	database.Port = nil

	_, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})

	assert.ErrorContains(t, err, "port")
}

func Test_OpenTunnel_WhenTunneled_DisablesTheTunnelFlagOnTheCopy(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.False(t, tunneledDatabase.GetDatabaseThroughTunnel().SshTunnel.IsEnabled,
		"a re-read of the copy would otherwise open a second tunnel into the first one")
	assert.True(t, database.SshTunnel.IsEnabled, "the original must keep the stored configuration")
}

func Test_OpenTunnel_WhenTunneled_GivesTheCopyItsOwnPortPointer(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.Equal(t, 27017, *database.Port, "the persisted model owns its own port")
	assert.NotSame(t, database.Port, tunneledDatabase.GetDatabaseThroughTunnel().Port)
	assert.Equal(t, "db.internal", database.Host)
}

func Test_OpenTunnel_WhenTunneled_ForcesDirectConnectionOnTheCopyOnly(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.True(t, tunneledDatabase.GetDatabaseThroughTunnel().IsDirectConnection,
		"topology discovery would re-dial the advertised member addresses and bypass the forward")
	assert.False(t, database.IsDirectConnection,
		"removing the tunnel must restore the failover behaviour the user configured")
}
