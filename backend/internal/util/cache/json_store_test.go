package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingStore struct {
	err error
}

func (s failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Set(context.Context, Entry) error {
	return s.err
}

func (s failingStore) CreateIfAbsent(context.Context, Entry) (bool, error) {
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

func Test_JSONStore_DefaultAndCustomLifetime(t *testing.T) {
	currentTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	memoryStore := newMemoryStore(1024, func() time.Time { return currentTime })
	jsonStore := NewJSONStore[string](memoryStore, "values")

	require.NoError(t, jsonStore.Set(t.Context(), "default", "default value"))
	require.NoError(t, jsonStore.SetWithLifetime(t.Context(), ExpiringValue[string]{
		Key:      "custom",
		Value:    "custom value",
		Lifetime: time.Minute,
	}))

	currentTime = currentTime.Add(time.Minute)
	customValue, err := jsonStore.Get(t.Context(), "custom")
	require.NoError(t, err)
	assert.Nil(t, customValue)

	defaultValue, err := jsonStore.Get(t.Context(), "default")
	require.NoError(t, err)
	require.NotNil(t, defaultValue)
	assert.Equal(t, "default value", *defaultValue)
}

func Test_JSONStore_NamespacesKeepIdenticalKeysIndependent(t *testing.T) {
	memoryStore := NewMemoryStore(1024)
	firstStore := NewJSONStore[string](memoryStore, "first")
	secondStore := NewJSONStore[string](memoryStore, "second")

	require.NoError(t, firstStore.Set(t.Context(), "shared", "first value"))
	require.NoError(t, secondStore.Set(t.Context(), "shared", "second value"))

	firstValue, err := firstStore.Get(t.Context(), "shared")
	require.NoError(t, err)
	require.NotNil(t, firstValue)
	assert.Equal(t, "first value", *firstValue)

	secondValue, err := secondStore.Get(t.Context(), "shared")
	require.NoError(t, err)
	require.NotNil(t, secondValue)
	assert.Equal(t, "second value", *secondValue)
}

func Test_JSONStore_WhenPayloadMalformed_ReturnsDecodeError(t *testing.T) {
	memoryStore := NewMemoryStore(1024)
	jsonStore := NewJSONStore[string](memoryStore, "values")
	require.NoError(t, memoryStore.Set(t.Context(), Entry{
		Key:      jsonStore.key("malformed"),
		Payload:  []byte("{"),
		Lifetime: time.Minute,
	}))

	value, err := jsonStore.Get(t.Context(), "malformed")
	assert.Nil(t, value)
	assert.ErrorContains(t, err, "decode cached value")
}

func Test_JSONStore_WhenValueCannotBeEncoded_ReturnsEncodeError(t *testing.T) {
	jsonStore := NewJSONStore[func()](NewMemoryStore(1024), "functions")

	err := jsonStore.Set(t.Context(), "function", func() {})
	assert.ErrorContains(t, err, "encode cached value")
}

func Test_JSONStore_WhenProviderFails_ReturnsProviderError(t *testing.T) {
	providerError := errors.New("provider failed")
	jsonStore := NewJSONStore[string](failingStore{err: providerError}, "values")

	_, err := jsonStore.Get(t.Context(), "key")
	assert.ErrorIs(t, err, providerError)

	err = jsonStore.Set(t.Context(), "key", "value")
	assert.ErrorIs(t, err, providerError)
}
