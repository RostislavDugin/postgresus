package backuping_logical

import (
	"os"
	"testing"

	cache_utils "databasus-backend/internal/util/cache"
)

// TestMain wipes Valkey at the start of this package's test binary so the
// scheduler/backuper subscriptions don't inherit leftover state from a
// previous run that died before its deferred Unsubscribe fired (e.g. a
// -failfast'd suite that exited mid-test).
func TestMain(m *testing.M) {
	if err := cache_utils.ClearAllCache(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
