package restore_token

import (
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"

	cache_utils "databasus-backend/internal/util/cache"
)

const restoreTokenPrefix = "physical_restore_token:"

// restoreTokenTTL is longer than a download token's 5 min: a physical restore
// stream can be large (full + incrementals + WAL) and is often piped straight
// into an extract, so the window to start it must comfortably outlast a human
// pasting the curl command.
const restoreTokenTTL = 15 * time.Minute

// Token authorizes one agent-less physical restore stream. Unlike a download
// token (one backup file) it is keyed by a restore SPEC — a database and an
// optional point-in-time — because the stream is resolved from many artifacts
// at request time, not a single stored object.
//
// It lives only in the cache, never in PostgreSQL: the secret token string is
// the key, the cache TTL expires it, and GETDEL consumes it exactly once. So it
// carries no persisted ID, expiry, or used flag — the store provides all three.
type Token struct {
	DatabaseID uuid.UUID  `json:"databaseId"`
	UserID     uuid.UUID  `json:"userId"`
	TargetTime *time.Time `json:"targetTime"`

	// BackupID, when set, switches the stream to a per-backup restore (the FULL
	// plus its incremental ancestors, no WAL) instead of the point-in-time path
	// driven by TargetTime. Exactly one of BackupID / TargetTime semantics
	// applies: a non-nil BackupID takes precedence and TargetTime is ignored.
	BackupID *uuid.UUID `json:"backupId"`
}

// store holds issued restore tokens in Valkey. Consuming via GETDEL makes
// single-use atomic across instances: two concurrent stream requests can never
// both consume the same token, which a DB find-then-mark-used path cannot
// guarantee without extra locking.
type store struct {
	cache *cache_utils.CacheUtil[Token]
}

func newStore(client valkey.Client) *store {
	return &store{
		cache: cache_utils.NewCacheUtil[Token](client, restoreTokenPrefix),
	}
}

func (s *store) issue(token string, restoreToken *Token) {
	s.cache.SetWithExpiration(token, restoreToken, restoreTokenTTL)
}

// consume atomically reads and deletes the token, returning nil when it is
// missing, expired, or already consumed.
func (s *store) consume(token string) *Token {
	return s.cache.GetAndDelete(token)
}
