package usecases_physical_postgresql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func createSupervisorWithStagedSegment(
	t *testing.T,
	fixture *PhysicalDBFixture,
	store *mockWalStorage,
	segmentNo uint64,
) (supervisor *WalStreamSupervisor, stagedSegmentPath string) {
	t.Helper()

	supervisor = NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:     fixture.DB.ID,
		SourceDB:       fixture.DB.PostgresqlPhysical,
		StorageID:      fixture.Storage.ID,
		Storage:        store,
		Encryption:     backups_core_enums.BackupEncryptionNone,
		FieldEncryptor: encryption.GetFieldEncryptor(),
		WalSegmentRepo: physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:    physical_repositories.GetWalHistoryRepository(),
		WatchDirRoot:   t.TempDir(),
		Logger:         logger.GetLogger(),
	})

	pendingUploadDir := filepath.Join(supervisor.watchDir, pendingUploadDirName)
	require.NoError(t, os.MkdirAll(pendingUploadDir, 0o700))

	return supervisor, writeSegmentOfSize(t, pendingUploadDir, walName(1, segmentNo), testWalSegmentSize)
}

func Test_SweepPendingUploads_WhenUploadSucceeds_RemovesSegmentFromStagingDir(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	supervisor, stagedSegmentPath := createSupervisorWithStagedSegment(t, fixture, store, 70)

	supervisor.sweepPendingUploads(t.Context(), logger.GetLogger(), supervisor.uploader.ProcessSegment)

	require.True(t, store.hasObject(walSegmentObjectName(fixture.DB.ID, 1, walName(1, 70))),
		"a segment staged out of the resume path is valid WAL of the older chain and must still reach storage")
	require.NoFileExists(t, stagedSegmentPath, "an uploaded segment must not keep occupying the local queue")
}

func Test_SweepPendingUploads_WhenUploadFails_RetainsSegmentForNextTick(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.startFailingSaves()

	supervisor, stagedSegmentPath := createSupervisorWithStagedSegment(t, fixture, store, 71)

	supervisor.sweepPendingUploads(t.Context(), logger.GetLogger(), supervisor.uploader.ProcessSegment)

	require.FileExists(t, stagedSegmentPath, "staged WAL must survive a storage outage")

	store.stopFailingSaves()
	supervisor.sweepPendingUploads(t.Context(), logger.GetLogger(), supervisor.uploader.ProcessSegment)

	require.True(t, store.hasObject(walSegmentObjectName(fixture.DB.ID, 1, walName(1, 71))))
	require.NoFileExists(t, stagedSegmentPath)
}
