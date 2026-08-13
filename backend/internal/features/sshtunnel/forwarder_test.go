package sshtunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
)

// The bastion's sshd also listens on 2222, which is never published to the host, so forwarding
// there proves bytes reach the far side without a second container. Port 22 would be a bad target:
// on a machine that runs its own sshd, a forwarder that dialled the host loopback instead of going
// through the bastion would still read a banner and pass.
const (
	bastionInternalPort = 2222
	sshBannerPrefix     = "SSH-2.0-"
)

func startTestBastion(t *testing.T) containers.Endpoint {
	t.Helper()

	return containers.StartSshBastion(t)
}

func readTestKey(t *testing.T, name string) string {
	t.Helper()

	key, err := os.ReadFile(filepath.Join(containers.GetSshBastionTestdataDir(t), name))
	require.NoError(t, err)

	return string(key)
}

func bastionConfig(bastion containers.Endpoint) Config {
	return Config{
		IsEnabled: true,
		Host:      bastion.Host,
		Port:      bastion.Port,
		Username:  containers.SshBastionUsername,
		AuthType:  AuthTypePassword,
		Password:  containers.SshBastionPassword,
	}
}

func openTestForwarder(t *testing.T, config Config) *Forwarder {
	t.Helper()

	forwarder, err := Open(t.Context(), OpenSpec{
		Config: config,
		Target: Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
		Logger: logger.GetLogger(),
	})
	require.NoError(t, err)

	t.Cleanup(forwarder.Close)

	return forwarder
}

func readBannerThroughTunnel(forwarder *Forwarder) error {
	conn, err := net.DialTimeout("tcp", forwarder.GetLocalEndpoint().String(), 15*time.Second)
	if err != nil {
		return err
	}

	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}

	banner := make([]byte, len(sshBannerPrefix))
	if _, err := io.ReadFull(conn, banner); err != nil {
		return err
	}

	if string(banner) != sshBannerPrefix {
		return fmt.Errorf("expected %q through the tunnel, got %q", sshBannerPrefix, string(banner))
	}

	return nil
}

func assertTargetReachableThroughTunnel(t *testing.T, forwarder *Forwarder) {
	t.Helper()

	require.NoError(t, readBannerThroughTunnel(forwarder))
}

// Negative control for every other test in this file: it pins that the target port really is
// unreachable without the tunnel. Without it, a forwarder that stopped tunnelling and dialled
// the target directly would keep all the happy-path tests green.
func Test_TargetPort_WithoutTheTunnel_IsUnreachableFromTheHost(t *testing.T) {
	bastion := startTestBastion(t)

	conn, err := net.DialTimeout(
		"tcp",
		Endpoint{Host: bastion.Host, Port: bastionInternalPort}.String(),
		5*time.Second,
	)
	if err == nil {
		_ = conn.Close()
	}

	require.Error(t, err, "the bastion's internal port must not be published, or the tunnel tests prove nothing")
}

func Test_Forwarder_WithPasswordAuth_ForwardsBytesToTarget(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))

	assertTargetReachableThroughTunnel(t, forwarder)
}

func Test_Forwarder_WithPrivateKeyAuth_ForwardsBytesToTarget(t *testing.T) {
	bastion := startTestBastion(t)

	config := bastionConfig(bastion)
	config.AuthType = AuthTypePrivateKey
	config.Password = ""
	config.PrivateKey = readTestKey(t, "test_key")

	forwarder := openTestForwarder(t, config)

	assertTargetReachableThroughTunnel(t, forwarder)
}

func Test_Forwarder_WithPassphraseProtectedPrivateKey_ForwardsBytesToTarget(t *testing.T) {
	bastion := startTestBastion(t)

	config := bastionConfig(bastion)
	config.AuthType = AuthTypePrivateKey
	config.Password = ""
	config.PrivateKey = readTestKey(t, "test_key_passphrase")
	config.PrivateKeyPassphrase = "testpassphrase"

	forwarder := openTestForwarder(t, config)

	assertTargetReachableThroughTunnel(t, forwarder)
}

// Every other test passes a nil encryptor, but a persisted database always has one. Handing the
// ciphertext straight to ssh.Password would leave the whole suite green and the feature dead.
// Split per credential so a break in one decryption path cannot hide behind the other auth method.
func Test_Open_WithEncryptedPassword_DecryptsItBeforeAuthenticating(t *testing.T) {
	bastion := startTestBastion(t)

	encryptor := prefixingEncryptor{}

	config := bastionConfig(bastion)
	require.NoError(t, config.EncryptSensitiveFields(encryptor))
	require.NotEqual(t, containers.SshBastionPassword, config.Password)

	assertTargetReachableThroughTunnel(t, openEncryptedForwarder(t, config, encryptor))
}

func Test_Open_WithEncryptedPrivateKeyAndPassphrase_DecryptsThemBeforeAuthenticating(t *testing.T) {
	bastion := startTestBastion(t)

	encryptor := prefixingEncryptor{}

	config := bastionConfig(bastion)
	config.AuthType = AuthTypePrivateKey
	config.Password = ""
	config.PrivateKey = readTestKey(t, "test_key_passphrase")
	config.PrivateKeyPassphrase = "testpassphrase"

	require.NoError(t, config.EncryptSensitiveFields(encryptor))
	require.NotEqual(t, "testpassphrase", config.PrivateKeyPassphrase)

	assertTargetReachableThroughTunnel(t, openEncryptedForwarder(t, config, encryptor))
}

func openEncryptedForwarder(t *testing.T, config Config, encryptor prefixingEncryptor) *Forwarder {
	t.Helper()

	forwarder, err := Open(t.Context(), OpenSpec{
		Config:    config,
		Target:    Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
		Encryptor: encryptor,
		Logger:    logger.GetLogger(),
	})
	require.NoError(t, err)

	t.Cleanup(forwarder.Close)

	return forwarder
}

func Test_Open_WhenPasswordIsWrong_ReturnsErrorWithoutLeakingTheSecret(t *testing.T) {
	bastion := startTestBastion(t)

	config := bastionConfig(bastion)
	config.Password = "definitely-not-the-password"

	forwarder, err := Open(t.Context(), OpenSpec{
		Config: config,
		Target: Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
		Logger: logger.GetLogger(),
	})

	require.Error(t, err)
	assert.Nil(t, forwarder)
	assert.NotContains(t, err.Error(), config.Password)
}

func Test_Open_WhenPassphraseIsWrong_ReturnsError(t *testing.T) {
	bastion := startTestBastion(t)

	config := bastionConfig(bastion)
	config.AuthType = AuthTypePrivateKey
	config.Password = ""
	config.PrivateKey = readTestKey(t, "test_key_passphrase")
	config.PrivateKeyPassphrase = "wrong-passphrase"

	forwarder, err := Open(t.Context(), OpenSpec{
		Config: config,
		Target: Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
		Logger: logger.GetLogger(),
	})

	require.Error(t, err)
	assert.Nil(t, forwarder)
	assert.NotContains(t, err.Error(), config.PrivateKeyPassphrase)
}

func Test_Close_ReleasesTheLocalPort(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))
	localEndpoint := forwarder.GetLocalEndpoint()

	assertTargetReachableThroughTunnel(t, forwarder)

	forwarder.Close()

	conn, err := net.DialTimeout("tcp", localEndpoint.String(), 5*time.Second)
	if err == nil {
		_ = conn.Close()
	}

	assert.Error(t, err)
}

// A connection that got past Accept before Close must not be able to open a fresh bastion session:
// nothing would be left to close it, and Close would sit in its Wait until the far end hung up.
// Driving the guard directly rather than racing Close against live traffic, because the race window
// is far too narrow to hit on purpose - a timing-based test here passes with the guard removed.
func Test_GetOrDialBastion_AfterClose_RefusesToOpenANewSession(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))
	assertTargetReachableThroughTunnel(t, forwarder)

	forwarder.Close()

	client, err := forwarder.getOrDialBastion()

	require.ErrorIs(t, err, ErrForwarderClosed)
	assert.Nil(t, client)

	forwarder.clientMutex.Lock()
	defer forwarder.clientMutex.Unlock()

	assert.Nil(t, forwarder.client, "a session opened after Close would never be closed")
}

// Close cancels the redial context before taking clientMutex. Without that, a redial parked on a
// TCP connect to an unreachable bastion holds the mutex for the full dial timeout, and Close waits
// behind it. Exercised through the dial seam because no real address stalls a connect on every host.
func Test_Close_WhileARedialIsStuckOnConnect_CancelsItInsteadOfWaitingOutTheTimeout(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))

	dialStarted := make(chan struct{})

	var dialStartedOnce sync.Once

	forwarder.dialBastionConn = func(ctx context.Context, _ string) (net.Conn, error) {
		dialStartedOnce.Do(func() { close(dialStarted) })

		<-ctx.Done()

		return nil, ctx.Err()
	}

	forwarder.clientMutex.Lock()
	cachedClient := forwarder.client
	forwarder.clientMutex.Unlock()

	forwarder.discardClient(cachedClient)

	go func() { _, _ = forwarder.getOrDialBastion() }()

	<-dialStarted

	closed := make(chan struct{})

	go func() {
		forwarder.Close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(bastionDialTimeout / 3):
		t.Fatal("Close waited on the stuck redial instead of cancelling it")
	}
}

// A bastion address that completes the TCP connect and then says nothing is what a blackholing
// middlebox or a wrong port looks like. ssh.ClientConfig.Timeout does not cover the handshake, so
// without an explicit deadline this blocks forever while holding clientMutex, taking Close with it.
func Test_Open_WhenBastionAcceptsButNeverHandshakes_FailsInsteadOfHanging(t *testing.T) {
	silentBastion, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = silentBastion.Close() })

	go func() {
		for {
			conn, err := silentBastion.Accept()
			if err != nil {
				return
			}

			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	silentAddress, isTCPAddress := silentBastion.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)

	opened := make(chan error, 1)

	go func() {
		forwarder, err := Open(t.Context(), OpenSpec{
			Config: Config{
				IsEnabled: true,
				Host:      "127.0.0.1",
				Port:      silentAddress.Port,
				Username:  containers.SshBastionUsername,
				AuthType:  AuthTypePassword,
				Password:  containers.SshBastionPassword,
			},
			Target:           Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
			Logger:           logger.GetLogger(),
			HandshakeTimeout: 2 * time.Second,
		})
		if forwarder != nil {
			forwarder.Close()
		}

		opened <- err
	}()

	select {
	case err := <-opened:
		require.Error(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("Open blocked on the SSH handshake instead of timing out")
	}
}

func Test_Close_CalledTwice_DoesNotPanic(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))

	forwarder.Close()
	forwarder.Close()
}

// The cached client dies between operations far more often than it dies mid-dial - a bastion
// restart, an idle timeout, a NAT drop. Every waiting connection then discovers the corpse at
// once, so the redial has to be serialized or the losers leak clients and goroutines.
func Test_Forwarder_WhenCachedClientIsDead_ConcurrentConnectionsAllRecover(t *testing.T) {
	bastion := startTestBastion(t)

	forwarder := openTestForwarder(t, bastionConfig(bastion))

	assertTargetReachableThroughTunnel(t, forwarder)

	var redialCount atomic.Int32

	dialOverTCP := forwarder.dialBastionConn
	forwarder.dialBastionConn = func(ctx context.Context, address string) (net.Conn, error) {
		redialCount.Add(1)

		return dialOverTCP(ctx, address)
	}

	forwarder.clientMutex.Lock()
	require.NotNil(t, forwarder.client)
	require.NoError(t, forwarder.client.Close())
	forwarder.clientMutex.Unlock()

	const concurrentConnections = 8

	// testify's FailNow only works on the goroutine running the test, so each connection reports
	// back and the assertions happen after the group is joined.
	connectionErrors := make([]error, concurrentConnections)

	var connectionGroup sync.WaitGroup

	for connectionIndex := range concurrentConnections {
		connectionGroup.Go(func() {
			connectionErrors[connectionIndex] = readBannerThroughTunnel(forwarder)
		})
	}

	connectionGroup.Wait()

	for connectionIndex, connectionError := range connectionErrors {
		assert.NoErrorf(t, connectionError, "connection %d failed to recover", connectionIndex)
	}

	// Not exactly one: dialTarget releases clientMutex between discardClient and its retry, so a
	// second waiter can slip in. Unserialized it would be one per connection.
	assert.Less(t, redialCount.Load(), int32(3), "the redial must be serialized, or the losers leak sessions")
}
