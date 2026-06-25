package container

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-verification-agent/internal/testutil"
)

func Test_BuildSpec_AppliesHardeningControls(t *testing.T) {
	// nil engine: buildSpec is pure and never calls the engine.
	containerManager := NewManager(nil, "agent-1", "databasus/verification-postgres", testutil.DiscardLogger())

	labels := map[string]string{LabelAgentID: "agent-1"}
	spec := containerManager.buildSpec(spawnPlan{
		verificationID: uuid.New(),
		image:          "postgres@sha256:x",
		password:       "pw",
		cpuPerJob:      2,
		ramMbPerJob:    1024,
		networkID:      "net-id",
		labels:         labels,
	})

	assert.True(t, spec.NoNewPrivileges)
	assert.True(t, spec.CapDropAll)
	assert.Equal(t, minimalCaps, spec.CapAdd)
	assert.EqualValues(t, containerPidsLimit, spec.PidsLimit)
	assert.Equal(t, "net-id", spec.NetworkID)
	assert.Equal(t, int64(2)*1_000_000_000, spec.NanoCPUs)
	assert.Equal(t, int64(1024)*1024*1024, spec.MemoryBytes)
	assert.Equal(t, []string{"POSTGRES_PASSWORD=pw"}, spec.Env)
}

func Test_BuildSpec_SetsRestoreTunedPostgresCmd(t *testing.T) {
	containerManager := NewManager(nil, "agent-1", "databasus/verification-postgres", testutil.DiscardLogger())

	spec := containerManager.buildSpec(spawnPlan{
		verificationID: uuid.New(),
		image:          "postgres@sha256:x",
		password:       "pw",
		cpuPerJob:      2,
		ramMbPerJob:    1024,
		networkID:      "net-id",
		labels:         map[string]string{LabelAgentID: "agent-1"},
	})

	assert.Equal(t, restoreTunedPostgresCmd, spec.Cmd)
	assert.Equal(t, "postgres", spec.Cmd[0])
}

func Test_GetInContainerConn_UsesInternalPort(t *testing.T) {
	c := &PostgresContainer{password: "pw"}

	conn := c.GetInContainerConn()

	assert.Equal(t, "127.0.0.1", conn.Host)
	assert.Equal(t, pgInternalPort, conn.Port)
	assert.Equal(t, restoreUser, conn.User)
	assert.Equal(t, restoreDB, conn.Database)
	assert.Equal(t, "pw", conn.Password)
}

func Test_GetVerifierConn_WhenSpawned_UsesResolvedHostPort(t *testing.T) {
	c := &PostgresContainer{password: "pw", hostPort: 54321}

	conn := c.GetVerifierConn()

	assert.Equal(t, "127.0.0.1", conn.Host)
	assert.Equal(t, 54321, conn.Port)
	assert.Equal(t, restoreUser, conn.User)
	assert.Equal(t, restoreDB, conn.Database)
	assert.Equal(t, "pw", conn.Password)
}

func Test_ImageForMajor_AppendsMajorTagToConfiguredRepo(t *testing.T) {
	cases := []struct {
		major    string
		expected string
	}{
		{"12", "databasus/verification-postgres:12"},
		{"16", "databasus/verification-postgres:16"},
		{"18", "databasus/verification-postgres:18"},
	}

	for _, tc := range cases {
		t.Run(tc.major, func(t *testing.T) {
			assert.Equal(t, tc.expected, imageForMajor(tc.major, "databasus/verification-postgres"))
		})
	}
}

func Test_ImageForJob_WhenTimescaleVersionSet_ReturnsVersionMatchedTimescaleImage(t *testing.T) {
	image := imageForJob(
		SpawnRequest{PgMajor: "17", TimescaledbVersion: "2.17.0"}, "databasus/verification-postgres")

	assert.Equal(t, "timescale/timescaledb:2.17.0-pg17", image,
		"timescale jobs ignore the engine repo and use the version-matched timescale image")
}

func Test_ImageForJob_WhenNoTimescaleVersion_UsesConfiguredEngineRepo(t *testing.T) {
	image := imageForJob(SpawnRequest{PgMajor: "17"}, "databasus/verification-postgres")

	assert.Equal(t, "databasus/verification-postgres:17", image)
}

func Test_StockFallbackImage_WhenBundledEngineRepo_FallsBackToStockPostgres(t *testing.T) {
	fallbackImage, canFallBack := stockFallbackImage(
		SpawnRequest{PgMajor: "16"}, "databasus/verification-postgres:16")

	assert.True(t, canFallBack)
	assert.Equal(t, "postgres:16", fallbackImage)
}

func Test_StockFallbackImage_WhenAlreadyStockRepo_DoesNotFallBack(t *testing.T) {
	_, canFallBack := stockFallbackImage(SpawnRequest{PgMajor: "16"}, "postgres:16")

	assert.False(t, canFallBack, "a job already on the stock image has nothing to fall back to")
}

func Test_StockFallbackImage_WhenTimescaleJob_DoesNotFallBack(t *testing.T) {
	_, canFallBack := stockFallbackImage(
		SpawnRequest{PgMajor: "16", TimescaledbVersion: "2.17.0"}, "timescale/timescaledb:2.17.0-pg16")

	assert.False(t, canFallBack,
		"timescale needs its exact version-matched image; stock postgres cannot restore it")
}
