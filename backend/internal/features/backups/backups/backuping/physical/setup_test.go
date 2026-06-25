package backuping_physical

import (
	"os"
	"testing"

	cache_utils "databasus-backend/internal/util/cache"
)

func TestMain(m *testing.M) {
	if err := cache_utils.ClearAllCache(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
