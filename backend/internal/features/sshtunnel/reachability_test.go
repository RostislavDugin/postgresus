package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/testing/containers"
)

// The caller is a supervision loop between two receiver spawns, so an answer that took the
// transport's own 30s dial budget would be useless — it has to come back inside the deadline it
// was given.
func Test_IsBastionReachable_WhenTheBastionIsGone_ReturnsFalseWithinTheCallersDeadline(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))
	assertTargetReachableThroughTunnel(t, forwarder)

	forwarder.clientMutex.Lock()
	cachedClient := forwarder.client
	forwarder.clientMutex.Unlock()

	forwarder.discardClient(cachedClient)

	forwarder.dialBastionConn = func(ctx context.Context, _ string) (net.Conn, error) {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	probeBudget := 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(t.Context(), probeBudget)
	defer cancel()

	probeStartedAt := time.Now()
	isReachable := forwarder.IsBastionReachable(ctx)

	assert.False(t, isReachable)
	assert.Less(t, time.Since(probeStartedAt), bastionDialTimeout/2,
		"the probe outlived the caller's deadline")
}

// Asking a question must not authenticate: getOrDialBastion would have installed a live session as
// a side effect, so a probe built on it would silently reconnect on every call.
func Test_IsBastionReachable_WhenNoClientIsCached_DoesNotCacheTheDialedClient(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))

	forwarder.clientMutex.Lock()
	cachedClient := forwarder.client
	forwarder.clientMutex.Unlock()

	forwarder.discardClient(cachedClient)

	assert.True(t, forwarder.IsBastionReachable(t.Context()))

	forwarder.clientMutex.Lock()
	defer forwarder.clientMutex.Unlock()

	assert.Nil(t, forwarder.client, "the probe installed a session nobody asked for")
}

// The cached-session branch cannot dial, so this is where the caller's deadline reaches it. A probe
// that fell back to the keepalive loop's own 15s budget would block a supervision loop that asked
// for three seconds.
func Test_BudgetUntil_WithACallerDeadline_ReportsTheRemainingTimeRatherThanTheDialTimeout(t *testing.T) {
	callerBudget := 3 * time.Second

	ctx, cancel := context.WithTimeout(t.Context(), callerBudget)
	defer cancel()

	assert.InDelta(t, callerBudget, budgetUntil(ctx), float64(200*time.Millisecond))
}

func Test_BudgetUntil_WithoutACallerDeadline_FallsBackToTheDialTimeout(t *testing.T) {
	assert.Equal(t, bastionDialTimeout, budgetUntil(context.Background()))
}

func Test_BudgetUntil_WithAnExpiredDeadline_ReportsZeroRatherThanANegativeBudget(t *testing.T) {
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Minute))
	defer cancel()

	assert.Equal(t, time.Duration(0), budgetUntil(ctx))
}

// startStallableBastionProxy fronts the real bastion with a pipe the test can stop feeding. Halting
// it mid-session is the failure keepalive exists for: the TCP connection stays open with no FIN, so
// a request written into it is never answered and never fails either.
func startStallableBastionProxy(t *testing.T, bastion containers.Endpoint) (containers.Endpoint, *atomic.Bool) {
	t.Helper()

	var listenConfig net.ListenConfig

	proxy, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = proxy.Close() })

	isStalled := &atomic.Bool{}

	go func() {
		for {
			clientConn, acceptErr := proxy.Accept()
			if acceptErr != nil {
				return
			}

			go pipeUntilStalled(t.Context(), clientConn, bastion, isStalled)
		}
	}()

	proxyAddr, isTCPAddr := proxy.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddr)

	return containers.Endpoint{Host: "127.0.0.1", Port: proxyAddr.Port}, isStalled
}

func pipeUntilStalled(
	ctx context.Context,
	clientConn net.Conn,
	bastion containers.Endpoint,
	isStalled *atomic.Bool,
) {
	defer func() { _ = clientConn.Close() }()

	bastionConn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", bastion.Host, bastion.Port), 10*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = bastionConn.Close() }()

	copyUntilStalled := func(dst, src net.Conn) {
		buffer := make([]byte, 32*1024)
		for {
			readCount, readErr := src.Read(buffer)
			if isStalled.Load() {
				// Neither answering nor hanging up is the whole point; unwinding at test end keeps
				// that from outliving the test.
				<-ctx.Done()

				return
			}

			if readCount > 0 {
				if _, writeErr := dst.Write(buffer[:readCount]); writeErr != nil {
					return
				}
			}

			if readErr != nil {
				return
			}
		}
	}

	go copyUntilStalled(bastionConn, clientConn)
	copyUntilStalled(clientConn, bastionConn)
}

// The cached-session branch cannot dial, so this is the only place the caller's deadline reaches it.
// With the keepalive loop's own 15s budget hard-coded here instead, a supervision loop asking for one
// second would block for fifteen between two receiver spawns.
func Test_IsBastionReachable_WhenTheCachedSessionHangs_ReturnsFalseWithinTheCallersDeadline(t *testing.T) {
	bastion := startTestBastion(t)
	proxy, isStalled := startStallableBastionProxy(t, bastion)

	forwarder := openTestForwarder(t, bastionConfig(proxy))

	forwarder.clientMutex.Lock()
	hasCachedClient := forwarder.client != nil
	forwarder.clientMutex.Unlock()
	require.True(t, hasCachedClient, "the probe must take the cached-session branch")

	isStalled.Store(true)

	probeBudget := time.Second

	ctx, cancel := context.WithTimeout(t.Context(), probeBudget)
	defer cancel()

	probeStartedAt := time.Now()

	assert.False(t, forwarder.IsBastionReachable(ctx))
	assert.Less(t, time.Since(probeStartedAt), keepaliveTimeout/2,
		"the probe waited out the keepalive loop's budget instead of the caller's")
}
