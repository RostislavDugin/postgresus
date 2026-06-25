package usecases_physical_postgresql_test

import (
	"os"
	"testing"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	cache_utils "databasus-backend/internal/util/cache"
)

// SetupDependencies registers the PhysicalSlotCleanupListener on
// DatabaseService.AddDbRemoveListener so RemoveTestDatabase drops
// per-backup and WAL streamer slots when test databases are torn down.
func TestMain(m *testing.M) {
	cache_utils.ClearAllCache()
	backuping_physical.SetupDependencies()

	os.Exit(m.Run())
}
