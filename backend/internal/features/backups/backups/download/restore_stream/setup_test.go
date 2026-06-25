package restore_stream_test

import (
	"os"
	"testing"

	cache_utils "databasus-backend/internal/util/cache"
)

func TestMain(m *testing.M) {
	cache_utils.ClearAllCache()

	os.Exit(m.Run())
}
