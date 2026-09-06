package ratelimiter

import "databasus-backend/internal/util/cache"

var defaultCounter Counter = NewMemoryCounter(cache.GetStore())

func GetCounter() Counter {
	return defaultCounter
}
