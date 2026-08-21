package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParsePostgresqlVersion_WhenVersionIsSupported_ReturnsEnum(t *testing.T) {
	version, err := ParsePostgresqlVersion("19")

	require.NoError(t, err)
	assert.Equal(t, PostgresqlVersion19, version)
}

func Test_ParsePostgresqlVersion_WhenVersionIsUnsupported_ReturnsError(t *testing.T) {
	version, err := ParsePostgresqlVersion("20")

	assert.Empty(t, version)
	require.EqualError(t, err, "invalid postgresql version: 20")
}
