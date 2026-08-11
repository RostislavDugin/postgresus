package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SplitAddr_ValidAddress_ReturnsHostAndPort(t *testing.T) {
	host, port, err := SplitAddr("test-postgres-16:5432")

	require.NoError(t, err)
	assert.Equal(t, "test-postgres-16", host)
	assert.Equal(t, 5432, port)
}

func Test_SplitAddr_NoPort_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr("test-postgres-16")

	assert.Error(t, err)
}

func Test_SplitAddr_EmptyHost_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr(":5432")

	assert.Error(t, err)
}

func Test_SplitAddr_NonNumericPort_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr("test-minio:not-a-port")

	assert.Error(t, err)
}
