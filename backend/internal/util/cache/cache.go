package cache_utils

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/valkey-io/valkey-go"

	"databasus-backend/internal/config"
)

var valkeyClient valkey.Client

var initCache = sync.OnceFunc(func() {
	env := config.GetEnv()

	options := valkey.ClientOption{
		InitAddress: []string{env.ValkeyHost + ":" + env.ValkeyPort},
		Password:    env.ValkeyPassword,
		Username:    env.ValkeyUsername,
		// 0 in production; per-worker logical DB under `go test` so parallel test
		// binaries never share keys (see config.applyTestWorkerSlot).
		SelectDB: env.ValkeySelectDB,
	}

	if env.ValkeyIsSsl {
		options.TLSConfig = &tls.Config{
			ServerName: env.ValkeyHost,
		}
	}

	client, err := valkey.NewClient(options)
	if err != nil {
		panic(err)
	}

	valkeyClient = client
})

func getCache() valkey.Client {
	initCache()
	return valkeyClient
}

func GetValkeyClient() valkey.Client {
	return getCache()
}

func TestCacheConnection() {
	// Get Valkey client from cache package
	client := getCache()

	// Create a simple test cache util for strings
	cacheUtil := NewCacheUtil[string](client, "test:")

	// Test data
	testKey := "connection_test"
	testValue := "valkey_is_working"

	// Test Set operation
	cacheUtil.Set(testKey, &testValue)

	// Test Get operation
	retrievedValue := cacheUtil.Get(testKey)

	// Verify the value was retrieved correctly
	if retrievedValue == nil {
		panic("Cache test failed: could not retrieve cached value")
	}

	if *retrievedValue != testValue {
		panic("Cache test failed: retrieved value does not match expected")
	}

	// Clean up - remove test key
	cacheUtil.Invalidate(testKey)

	// Verify cleanup worked
	cleanupCheck := cacheUtil.Get(testKey)
	if cleanupCheck != nil {
		panic("Cache test failed: test key was not properly invalidated")
	}
}

// FlushAll wipes every key across all Valkey logical DBs (FLUSHALL), regardless
// of which DB the client selected. Used by cleanup_test_db to reset the cache for
// every test worker slot at once; ClearAllCache only touches the selected DB.
func FlushAll() error {
	client := getCache()

	ctx, cancel := context.WithTimeout(context.Background(), DefaultQueueTimeout)
	defer cancel()

	return client.Do(ctx, client.B().Flushall().Build()).Error()
}

func ClearAllCache() error {
	pattern := "*"
	cursor := uint64(0)
	batchSize := int64(100)

	cacheUtil := NewCacheUtil[string](getCache(), "")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultQueueTimeout)

		result := cacheUtil.client.Do(
			ctx,
			cacheUtil.client.B().Scan().Cursor(cursor).Match(pattern).Count(batchSize).Build(),
		)
		cancel()

		if result.Error() != nil {
			return result.Error()
		}

		scanResult, err := result.AsScanEntry()
		if err != nil {
			return err
		}

		if len(scanResult.Elements) > 0 {
			delCtx, delCancel := context.WithTimeout(context.Background(), cacheUtil.timeout)
			cacheUtil.client.Do(
				delCtx,
				cacheUtil.client.B().Del().Key(scanResult.Elements...).Build(),
			)
			delCancel()
		}

		cursor = scanResult.Cursor
		if cursor == 0 {
			break
		}
	}

	return nil
}
