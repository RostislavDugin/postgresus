package ratelimiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/cache"
)

type failingStore struct {
	err error
}

func (s failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Set(context.Context, cache.Entry) error {
	return s.err
}

func (s failingStore) CreateIfAbsent(context.Context, cache.Entry) (bool, error) {
	return false, s.err
}

func (s failingStore) ReadAndDelete(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Delete(context.Context, string) error {
	return s.err
}

func (s failingStore) Clear(context.Context) error {
	return s.err
}

func Test_MemoryCounter_SlidingWindowCountsRejectedAttempts(t *testing.T) {
	currentTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	counter := newMemoryCounter(cache.NewMemoryStore(4096), func() time.Time { return currentTime })
	attempt := Attempt{Scope: "signin", Identifier: "user", Limit: 2, Window: time.Minute}

	isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.True(t, isAllowed)

	currentTime = currentTime.Add(10 * time.Second)
	isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.True(t, isAllowed)

	currentTime = currentTime.Add(10 * time.Second)
	isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.False(t, isAllowed)

	currentTime = currentTime.Add(10 * time.Second)
	isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.False(t, isAllowed)

	currentTime = currentTime.Add(40 * time.Second)
	isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.False(t, isAllowed)

	currentTime = currentTime.Add(20 * time.Second)
	isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.True(t, isAllowed)
}

func Test_MemoryCounter_WhenSettingsInvalid_DoesNotRecordAttempt(t *testing.T) {
	counter := NewMemoryCounter(cache.NewMemoryStore(4096))
	attempt := Attempt{Scope: "signin", Identifier: "user", Limit: 0, Window: time.Minute}

	_, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	assert.ErrorIs(t, err, ErrInvalidSettings)

	attempt.Limit = 1
	isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
	require.NoError(t, err)
	assert.True(t, isAllowed)
}

func Test_MemoryCounter_WhenProviderFails_ReturnsErrorAndRejectsAttempt(t *testing.T) {
	providerError := errors.New("provider failed")
	counter := NewMemoryCounter(failingStore{err: providerError})

	isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), Attempt{
		Scope: "signin", Identifier: "user", Limit: 1, Window: time.Minute,
	})
	assert.False(t, isAllowed)
	assert.ErrorIs(t, err, providerError)
}
