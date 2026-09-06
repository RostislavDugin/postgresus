package ratelimiter_test

import (
	"testing"

	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/ratelimiter"
	"databasus-backend/internal/util/ratelimiter/ratelimitertest"
)

func Test_MemoryCounter_ProviderContract(t *testing.T) {
	ratelimitertest.RunCounterContract(t, func(*testing.T) ratelimiter.Counter {
		return ratelimiter.NewMemoryCounter(cache.NewMemoryStore(64 * 1024))
	})
}
