package cache_test

import (
	"testing"

	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/cache/cachetest"
)

func Test_MemoryStore_ProviderContract(t *testing.T) {
	cachetest.RunStoreContract(t, func(*testing.T) cache.Store {
		return cache.NewMemoryStore(64 * 1024)
	})
}
