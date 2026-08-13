package databases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/databases/databases/mysql"
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
