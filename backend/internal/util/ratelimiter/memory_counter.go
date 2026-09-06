package ratelimiter

import (
	"context"
	"strconv"
	"sync"
	"time"

	"databasus-backend/internal/util/cache"
)

const cacheNamespace = "rate_limit_attempts"

type MemoryCounter struct {
	mutex          sync.Mutex
	attempts       *cache.JSONStore[[]time.Time]
	getCurrentTime func() time.Time
}

func NewMemoryCounter(store cache.Store) *MemoryCounter {
	return newMemoryCounter(store, func() time.Time {
		return time.Now().UTC()
	})
}

func newMemoryCounter(store cache.Store, getCurrentTime func() time.Time) *MemoryCounter {
	return &MemoryCounter{
		attempts:       cache.NewJSONStore[[]time.Time](store, cacheNamespace),
		getCurrentTime: getCurrentTime,
	}
}

func (c *MemoryCounter) RecordAttemptAndCheckIsAllowed(
	ctx context.Context,
	attempt Attempt,
) (bool, error) {
	if attempt.Limit <= 0 || attempt.Window <= 0 {
		return false, ErrInvalidSettings
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	currentTime := c.getCurrentTime()
	storedAttempts, err := c.attempts.Get(ctx, counterKey(attempt))
	if err != nil {
		return false, err
	}

	windowStart := currentTime.Add(-attempt.Window)
	recentAttempts := make([]time.Time, 0, attempt.Limit+1)
	if storedAttempts != nil {
		for _, recordedAt := range *storedAttempts {
			if recordedAt.After(windowStart) {
				recentAttempts = append(recentAttempts, recordedAt)
			}
		}
	}

	recentAttempts = append(recentAttempts, currentTime)
	if len(recentAttempts) > attempt.Limit+1 {
		recentAttempts = recentAttempts[len(recentAttempts)-(attempt.Limit+1):]
	}

	if err := c.attempts.SetWithLifetime(
		ctx,
		cache.ExpiringValue[[]time.Time]{
			Key:      counterKey(attempt),
			Value:    recentAttempts,
			Lifetime: attempt.Window,
		},
	); err != nil {
		return false, err
	}

	return len(recentAttempts) <= attempt.Limit, nil
}

func counterKey(attempt Attempt) string {
	return strconv.Itoa(len(attempt.Scope)) + ":" + attempt.Scope + ":" + attempt.Identifier
}
