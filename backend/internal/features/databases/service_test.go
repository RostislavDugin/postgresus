package databases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
)

// The healthcheck path runs without an end-user principal, so the embedded-target refusal inside
// OpenTunnel is the only thing keeping a persisted row from probing Databasus's own cluster.
func Test_ProbeDatabaseLiveness_WithEmbeddedPhysicalTarget_ReturnsError(t *testing.T) {
	database := &Database{
		Type: DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Host:     "/tmp",
			Port:     5437,
			Username: "postgres",
			Password: "stored-password",
		},
	}

	err := GetDatabaseService().ProbeDatabaseLiveness(t.Context(), database)

	require.ErrorContains(t, err, "backing up Databasus internal PostgreSQL cluster is not allowed")
}

// The source cluster publishes no port, so a passing probe proves it went through the bastion.
func Test_ProbeDatabaseLiveness_WhenSourceIsBehindSshBastion_Succeeds(t *testing.T) {
	topology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")

	physicalConfig := GetTestPhysicalPostgresConfigWithType(
		topology.Database.Host, topology.Database.Port, "17", postgresql_physical.BackupTypeFullOnly,
	)
	physicalConfig.SshTunnel = bastion.GetTunnelConfig(topology)

	database := &Database{
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: physicalConfig,
	}

	err := GetDatabaseService().ProbeDatabaseLiveness(t.Context(), database)

	assert.NoError(t, err)
}
