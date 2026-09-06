package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MemoryStore_Contract(t *testing.T) {
	currentTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(128, func() time.Time { return currentTime })
	ctx := t.Context()

	require.NoError(t, store.Set(ctx, Entry{Key: "alpha", Payload: []byte("value"), Lifetime: time.Minute}))

	payload, isFound, err := store.Get(ctx, "alpha")
	require.NoError(t, err)
	require.True(t, isFound)
	assert.Equal(t, []byte("value"), payload)

	payload[0] = 'X'
	storedPayload, isFound, err := store.Get(ctx, "alpha")
	require.NoError(t, err)
	require.True(t, isFound)
	assert.Equal(t, []byte("value"), storedPayload)

	sourcePayload := []byte("source")
	require.NoError(t, store.Set(ctx, Entry{Key: "copied", Payload: sourcePayload, Lifetime: time.Minute}))
	sourcePayload[0] = 'X'
	storedPayload, isFound, err = store.Get(ctx, "copied")
	require.NoError(t, err)
	require.True(t, isFound)
	assert.Equal(t, []byte("source"), storedPayload)

	currentTime = currentTime.Add(time.Minute)
	_, isFound, err = store.Get(ctx, "alpha")
	require.NoError(t, err)
	assert.False(t, isFound)

	require.NoError(t, store.Clear(ctx))
	_, isFound, err = store.Get(ctx, "copied")
	require.NoError(t, err)
	assert.False(t, isFound)
}

func Test_MemoryStore_WhenContextCanceled_DoesNotChangeEntries(t *testing.T) {
	store := NewMemoryStore(128)
	require.NoError(t, store.Set(t.Context(), Entry{Key: "existing", Payload: []byte("value"), Lifetime: time.Minute}))

	canceledContext, cancel := context.WithCancel(t.Context())
	cancel()

	assert.ErrorIs(
		t,
		store.Set(canceledContext, Entry{Key: "new", Payload: []byte("value"), Lifetime: time.Minute}),
		context.Canceled,
	)
	_, isFound, err := store.Get(t.Context(), "new")
	require.NoError(t, err)
	assert.False(t, isFound)

	assert.ErrorIs(t, store.Delete(canceledContext, "existing"), context.Canceled)
	_, isFound, err = store.Get(t.Context(), "existing")
	require.NoError(t, err)
	assert.True(t, isFound)
}

func Test_MemoryStore_WhenPayloadBudgetReached_RejectsWriteWithoutEvictingLiveEntries(t *testing.T) {
	currentTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(10, func() time.Time { return currentTime })

	require.NoError(t, store.Set(t.Context(), Entry{Key: "a", Payload: []byte("1234"), Lifetime: time.Minute}))
	require.NoError(t, store.Set(t.Context(), Entry{Key: "b", Payload: []byte("1234"), Lifetime: time.Minute}))

	err := store.Set(t.Context(), Entry{Key: "c", Payload: []byte("1"), Lifetime: time.Minute})
	assert.ErrorIs(t, err, ErrPayloadBudgetReached)

	_, isFound, err := store.Get(t.Context(), "a")
	require.NoError(t, err)
	assert.True(t, isFound)
	_, isFound, err = store.Get(t.Context(), "b")
	require.NoError(t, err)
	assert.True(t, isFound)

	currentTime = currentTime.Add(time.Minute)
	require.NoError(t, store.Set(t.Context(), Entry{Key: "r", Payload: []byte("x"), Lifetime: time.Minute}))
}

func Test_MemoryStore_WhenWriteFitsBudget_DoesNotScanUnrelatedEntries(t *testing.T) {
	currentTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(128, func() time.Time { return currentTime })

	require.NoError(t, store.Set(t.Context(), Entry{
		Key: "expired", Payload: []byte("value"), Lifetime: time.Minute,
	}))
	currentTime = currentTime.Add(time.Minute)

	require.NoError(t, store.Set(t.Context(), Entry{
		Key: "current", Payload: []byte("value"), Lifetime: time.Minute,
	}))
	assert.Contains(t, store.entries, "expired")
}

func Test_MemoryStore_WhenLifetimeInvalid_ReturnsError(t *testing.T) {
	store := NewMemoryStore(128)

	err := store.Set(t.Context(), Entry{Key: "invalid", Payload: []byte("value")})
	assert.True(t, errors.Is(err, ErrInvalidLifetime))
}
