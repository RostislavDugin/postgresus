package cache_utils

import (
	"context"
	"encoding/json"
	"time"

	"github.com/valkey-io/valkey-go"

	"databasus-backend/internal/util/logger"
)

const (
	DefaultCacheTimeout = 10 * time.Second
	DefaultCacheExpiry  = 10 * time.Minute
	DefaultQueueTimeout = 30 * time.Second
)

type CacheUtil[T any] struct {
	client  valkey.Client
	prefix  string
	timeout time.Duration
	expiry  time.Duration
}

func NewCacheUtil[T any](client valkey.Client, prefix string) *CacheUtil[T] {
	return &CacheUtil[T]{
		client:  client,
		prefix:  prefix,
		timeout: DefaultCacheTimeout,
		expiry:  DefaultCacheExpiry,
	}
}

func (c *CacheUtil[T]) Get(key string) *T {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	fullKey := c.prefix + key
	result := c.client.Do(ctx, c.client.B().Get().Key(fullKey).Build())

	if err := result.Error(); err != nil {
		// A missing key is the normal case and says nothing; anything else means the cache is down,
		// which is otherwise indistinguishable from a miss at every call site.
		if !valkey.IsValkeyNil(err) {
			logger.GetLogger().Warn("cache read failed", "cache_key", fullKey, "error", err)
		}

		return nil
	}

	data, err := result.AsBytes()
	if err != nil {
		logger.GetLogger().Warn("cache read returned an unreadable value", "cache_key", fullKey, "error", err)

		return nil
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		logger.GetLogger().Warn("cache read returned malformed json", "cache_key", fullKey, "error", err)

		return nil
	}

	return &item
}

func (c *CacheUtil[T]) Set(key string, item *T) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	data, err := json.Marshal(item)
	if err != nil {
		return
	}

	fullKey := c.prefix + key
	c.client.Do(ctx, c.client.B().Set().Key(fullKey).Value(string(data)).Ex(c.expiry).Build())
}

func (c *CacheUtil[T]) SetWithExpiration(key string, item *T, expiry time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	data, err := json.Marshal(item)
	if err != nil {
		return
	}

	fullKey := c.prefix + key
	c.client.Do(ctx, c.client.B().Set().Key(fullKey).Value(string(data)).Ex(expiry).Build())
}

func (c *CacheUtil[T]) GetAndDelete(key string) *T {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	fullKey := c.prefix + key
	result := c.client.Do(ctx, c.client.B().Getdel().Key(fullKey).Build())

	if result.Error() != nil {
		return nil
	}

	data, err := result.AsBytes()
	if err != nil {
		return nil
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return nil
	}

	return &item
}

func (c *CacheUtil[T]) Invalidate(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	fullKey := c.prefix + key
	c.client.Do(ctx, c.client.B().Del().Key(fullKey).Build())
}
