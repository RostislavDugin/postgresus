package stream_guard

import (
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"

	cache_utils "databasus-backend/internal/util/cache"
)

const (
	downloadLockPrefix = "backup_download_lock:"
	// downloadLockTTL must exceed downloadHeartbeatDelay so an in-flight stream's
	// heartbeat keeps renewing the lock, while a stream that dies without
	// releasing it self-heals within the TTL instead of locking the user out
	// until the cache's default expiry (10 min).
	downloadLockTTL        = 5 * time.Second
	downloadLockValue      = "1"
	downloadHeartbeatDelay = 3 * time.Second
)

type Tracker struct {
	cache *cache_utils.CacheUtil[string]
}

func NewTracker(client valkey.Client) *Tracker {
	return &Tracker{
		cache: cache_utils.NewCacheUtil[string](client, downloadLockPrefix),
	}
}

func (t *Tracker) AcquireDownloadLock(userID uuid.UUID) error {
	key := userID.String()

	existingLock := t.cache.Get(key)
	if existingLock != nil {
		return ErrDownloadAlreadyInProgress
	}

	value := downloadLockValue
	t.cache.SetWithExpiration(key, &value, downloadLockTTL)

	return nil
}

func (t *Tracker) RefreshDownloadLock(userID uuid.UUID) {
	key := userID.String()
	value := downloadLockValue
	t.cache.SetWithExpiration(key, &value, downloadLockTTL)
}

func (t *Tracker) ReleaseDownloadLock(userID uuid.UUID) {
	key := userID.String()
	t.cache.Invalidate(key)
}

func (t *Tracker) IsDownloadInProgress(userID uuid.UUID) bool {
	key := userID.String()
	existingLock := t.cache.Get(key)
	return existingLock != nil
}

func GetDownloadHeartbeatInterval() time.Duration {
	return downloadHeartbeatDelay
}
