package storage_connections

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const conditionWaitLimit = 5 * time.Second

type fakeSession struct {
	dialNumber int
	closed     chan struct{}
	closeOnce  sync.Once
	isBroken   atomic.Bool
	probeCount atomic.Int32
}

func (s *fakeSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

type fakeServer struct {
	dialCount atomic.Int32
}

func (f *fakeServer) dial(_ context.Context) (*fakeSession, error) {
	return &fakeSession{dialNumber: int(f.dialCount.Add(1)), closed: make(chan struct{})}, nil
}

func newTestPool(settings PoolSettings) *SessionPool[*fakeSession] {
	return NewSessionPool(SessionOperations[*fakeSession]{
		Close: func(session *fakeSession) {
			session.closeOnce.Do(func() { close(session.closed) })
		},
		Probe: func(_ context.Context, session *fakeSession) error {
			session.probeCount.Add(1)

			if session.isBroken.Load() {
				return errors.New("session is broken")
			}

			return nil
		},
	}, settings)
}

func newTestKey(storageID uuid.UUID, fingerprintParts ...any) Key {
	return Key{StorageID: storageID, Fingerprint: Fingerprint(fingerprintParts...)}
}

func checkoutSession(
	t *testing.T,
	pool *SessionPool[*fakeSession],
	server *fakeServer,
	key Key,
) *Lease[*fakeSession] {
	t.Helper()

	lease, err := pool.Checkout(t.Context(), key, server.dial)
	require.NoError(t, err)

	return lease
}

func Test_Fingerprint_WhenArgumentsMatchOrDiffer_ReturnsSameOrDifferentValue(t *testing.T) {
	baseline := Fingerprint("host", 22, "user", "encrypted-password")

	assert.Equal(t, baseline, Fingerprint("host", 22, "user", "encrypted-password"))
	assert.NotEqual(t, baseline, Fingerprint("other-host", 22, "user", "encrypted-password"))
	assert.NotEqual(t, baseline, Fingerprint("host", 2222, "user", "encrypted-password"))
	assert.NotEqual(t, baseline, Fingerprint("host", 22, "user", "changed-password"))
}

func Test_Checkout_WhenCalledSequentially_ReusesOneSession(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	firstLease := checkoutSession(t, pool, server, key)
	firstSession := firstLease.GetSession()
	firstLease.Release(nil)

	secondLease := checkoutSession(t, pool, server, key)
	defer secondLease.Release(nil)

	assert.Same(t, firstSession, secondLease.GetSession())
	assert.EqualValues(t, 1, server.dialCount.Load())
}

func Test_Checkout_WhenTwoLeasesAreHeldAtOnce_HandsOutDifferentSessions(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	firstLease := checkoutSession(t, pool, server, key)
	defer firstLease.Release(nil)

	secondLease := checkoutSession(t, pool, server, key)
	defer secondLease.Release(nil)

	assert.NotSame(t, firstLease.GetSession(), secondLease.GetSession())
	assert.EqualValues(t, 2, server.dialCount.Load())
}

func Test_Release_WhenCallFailed_ClosesSessionAndNextCheckoutDials(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	failedLease := checkoutSession(t, pool, server, key)
	failedSession := failedLease.GetSession()
	failedLease.Release(errors.New("write failed"))

	nextLease := checkoutSession(t, pool, server, key)
	defer nextLease.Release(nil)

	assert.True(t, failedSession.isClosed())
	assert.NotSame(t, failedSession, nextLease.GetSession())
	assert.EqualValues(t, 2, server.dialCount.Load())
}

func Test_Release_WhenIdleCapIsReached_ClosesFifthSession(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	leases := make([]*Lease[*fakeSession], 0, 5)
	for range 5 {
		leases = append(leases, checkoutSession(t, pool, server, key))
	}

	for _, lease := range leases {
		lease.Release(nil)
	}

	for _, lease := range leases[:4] {
		assert.False(t, lease.GetSession().isClosed())
	}

	assert.True(t, leases[4].GetSession().isClosed())
}

func Test_Checkout_WhenIdleSessionFailsProbe_DialsNewSession(t *testing.T) {
	settings := GetDefaultPoolSettings()
	settings.ProbeAfter = time.Millisecond
	pool := newTestPool(settings)
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	staleLease := checkoutSession(t, pool, server, key)
	staleSession := staleLease.GetSession()
	staleLease.Release(nil)
	releasedAt := time.Now()

	staleSession.isBroken.Store(true)
	require.Eventually(t, func() bool {
		return time.Since(releasedAt) > settings.ProbeAfter
	}, conditionWaitLimit, time.Millisecond)

	freshLease := checkoutSession(t, pool, server, key)
	defer freshLease.Release(nil)

	assert.EqualValues(t, 1, staleSession.probeCount.Load())
	assert.True(t, staleSession.isClosed())
	assert.NotSame(t, staleSession, freshLease.GetSession())
}

func Test_Checkout_WhenContextEndsWhileHolderIsBlocked_ClosesSessionAndCallReturns(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	lease, err := pool.Checkout(ctx, key, server.dial)
	require.NoError(t, err)

	blockedCallReturned := make(chan struct{})
	go func() {
		<-lease.GetSession().closed
		close(blockedCallReturned)
	}()

	cancel()

	select {
	case <-blockedCallReturned:
	case <-time.After(conditionWaitLimit):
		t.Fatal("blocked call did not return after the lease context was cancelled")
	}

	lease.Release(errors.New("call failed after its session was closed"))

	nextLease := checkoutSession(t, pool, server, key)
	defer nextLease.Release(nil)

	assert.NotSame(t, lease.GetSession(), nextLease.GetSession())
}

func Test_Checkout_WhenFingerprintChanges_ClosesOldSessionsAndStaleLeaseOnRelease(t *testing.T) {
	pool := newTestPool(GetDefaultPoolSettings())
	server := &fakeServer{}
	storageID := uuid.New()
	oldKey := newTestKey(storageID, "old-password")
	newKey := newTestKey(storageID, "new-password")

	idleOldLease := checkoutSession(t, pool, server, oldKey)
	heldOldLease := checkoutSession(t, pool, server, oldKey)
	idleOldLease.Release(nil)

	newLease := checkoutSession(t, pool, server, newKey)
	defer newLease.Release(nil)

	assert.True(t, idleOldLease.GetSession().isClosed())
	assert.NotSame(t, idleOldLease.GetSession(), newLease.GetSession())

	heldOldLease.Release(nil)

	assert.True(t, heldOldLease.GetSession().isClosed())
}

func Test_Release_WhenSessionStaysIdlePastIdleTimeout_ClosesSession(t *testing.T) {
	settings := GetDefaultPoolSettings()
	settings.IdleTimeout = 50 * time.Millisecond
	pool := newTestPool(settings)
	server := &fakeServer{}
	key := newTestKey(uuid.New(), "settings")

	lease := checkoutSession(t, pool, server, key)
	lease.Release(nil)

	require.Eventually(t, lease.GetSession().isClosed, conditionWaitLimit, 10*time.Millisecond)

	nextLease := checkoutSession(t, pool, server, key)
	defer nextLease.Release(nil)

	assert.NotSame(t, lease.GetSession(), nextLease.GetSession())
}
