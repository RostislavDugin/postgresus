package container

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/features/dbconn"
)

func Test_WaitForReady_WhenPortRefusesConnections_ReturnsDeadlineWithLastPingError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	closedPort := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	const password = "secret-restore-pw-123"

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	err = waitForReady(ctx, dbconn.Conn{
		Host: "127.0.0.1", Port: closedPort, User: restoreUser, Password: password, Database: restoreDB,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "last ping error")
	assert.Contains(t, err.Error(), "connection refused",
		"the error must say why Postgres stayed unreachable, not only that time ran out")
	assert.NotContains(t, err.Error(), password, "the report must not leak the database password")
}
