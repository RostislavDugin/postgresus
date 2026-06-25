package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

const testWalSegmentSize = int64(16 * 1024 * 1024)

// lsnSpanUpperBoundForTests is an effectively-unbounded upper LSN used when a
// chain-span query should return every segment regardless of position.
const lsnSpanUpperBoundForTests = walmath.LSN(1) << 62

// mockWalStorage is a controllable StorageFileSaver for WAL uploader tests: it
// records saved/deleted object names, can fail the first N SaveFile calls, and
// can block one SaveFile until released (to interleave the DeleteFull cascade
// race).
type mockWalStorage struct {
	mu        sync.Mutex
	saved     map[string][]byte
	deleted   []string
	saveCount atomic.Int64

	failSaveTimes int

	// blockOn, when set, makes SaveFile for that exact object name signal started
	// and wait on release before returning.
	blockOn string
	started chan struct{}
	release chan struct{}
}

func newMockWalStorage() *mockWalStorage {
	return &mockWalStorage{saved: make(map[string][]byte)}
}

func (m *mockWalStorage) SaveFile(
	_ context.Context, _ encryption.FieldEncryptor, _ *slog.Logger, fileName string, file io.Reader,
) error {
	m.saveCount.Add(1)

	body, _ := io.ReadAll(file)

	if m.blockOn != "" && fileName == m.blockOn {
		close(m.started)
		<-m.release
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failSaveTimes > 0 {
		m.failSaveTimes--

		return fmt.Errorf("mock storage induced failure for %s", fileName)
	}

	m.saved[fileName] = body

	return nil
}

func (m *mockWalStorage) DeleteFile(_ encryption.FieldEncryptor, fileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.saved, fileName)
	m.deleted = append(m.deleted, fileName)

	return nil
}

func (m *mockWalStorage) GetFile(_ encryption.FieldEncryptor, _ string) (io.ReadCloser, error) {
	return nil, errors.New("GetFile not implemented in mockWalStorage")
}
func (m *mockWalStorage) Validate(_ encryption.FieldEncryptor) error             { return nil }
func (m *mockWalStorage) TestConnection(_ encryption.FieldEncryptor) error       { return nil }
func (m *mockWalStorage) HideSensitiveData()                                     {}
func (m *mockWalStorage) EncryptSensitiveData(_ encryption.FieldEncryptor) error { return nil }

func (m *mockWalStorage) hasObject(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.saved[name]

	return ok
}

// walName builds a canonical 24-hex WAL filename for a timeline + segment number
// at the default 16 MB segment size (256 segments per logid).
func walName(timeline uint32, segNo uint64) string {
	const segmentsPerLogID = 256

	return fmt.Sprintf("%08X%08X%08X", timeline, segNo/segmentsPerLogID, segNo%segmentsPerLogID)
}

// writeWalFile writes a small placeholder segment file under dir named for the
// WAL filename. The body size is irrelevant — LSN bounds come from the filename,
// not the file length — so a tiny file keeps the test fast.
func writeWalFile(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("wal-segment-body-"+name), 0o600))

	return path
}

// newTestUploader wires a WalUploader against the real catalog repo and a mock
// storage, using the fixture's database/storage IDs for FK integrity.
func newTestUploader(
	fixture *PhysicalDBFixture,
	store *mockWalStorage,
	onGap func(walmath.LSN, walmath.LSN),
) *WalUploader {
	return NewWalUploader(WalUploadDeps{
		DatabaseID:          fixture.DB.ID,
		StorageID:           fixture.Storage.ID,
		Storage:             store,
		Encryption:          backups_core_enums.BackupEncryptionNone,
		FieldEncryptor:      encryption.GetFieldEncryptor(),
		WalSegmentRepo:      physical_repositories.GetWalSegmentRepository(),
		WalSegmentSizeBytes: testWalSegmentSize,
		Logger:              logger.GetLogger(),
		OnGapDetected:       onGap,
	})
}

func findWalSegment(
	t *testing.T,
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) *physical_models.PhysicalWalSegment {
	t.Helper()

	row, err := physical_repositories.GetWalSegmentRepository().FindByChainKey(databaseID, timelineID, startLSN)
	require.NoError(t, err)

	return row
}

func Test_WalUpload_ClaimSucceeds_SaveFileLands_FileNameUpdatedFromNullToNonNull(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, nil)

	dir := t.TempDir()
	name := walName(1, 5)
	localPath := writeWalFile(t, dir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), localPath, name))

	startLSN := walmath.LSN(5 * uint64(testWalSegmentSize))
	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.NotNil(t, row.FileName, "file_name must be flipped from NULL to the object key")

	objectName := walSegmentObjectName(fixture.DB.ID, 1, name)
	require.Equal(t, objectName, *row.FileName)
	require.True(t, store.hasObject(objectName), "artifact must be in storage under the deterministic name")
	require.True(t, store.hasObject(objectName+metadataSuffix), "sidecar must be uploaded")

	require.NoFileExists(t, localPath, "local segment must be removed after a committed upload")
}

func Test_WalUpload_SaveFileFails_ClaimRowDeletedLocalRetained_NextTickSucceeds(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.failSaveTimes = 1
	uploader := newTestUploader(fixture, store, nil)

	dir := t.TempDir()
	name := walName(1, 9)
	localPath := writeWalFile(t, dir, name)

	startLSN := walmath.LSN(9 * uint64(testWalSegmentSize))

	// First attempt: SaveFile fails -> claim row deleted, local retained.
	require.Error(t, uploader.ProcessSegment(context.Background(), localPath, name))
	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, startLSN), "failed claim row must be deleted")
	require.FileExists(t, localPath, "local segment must be retained for retry")

	// Second attempt: succeeds end-to-end.
	require.NoError(t, uploader.ProcessSegment(context.Background(), localPath, name))

	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.NotNil(t, row.FileName)
	require.NoFileExists(t, localPath)
}

func Test_WalUpload_DuplicateClaim_CompletedRow_DropsLocalNoStorageWrite(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, nil)

	dir := t.TempDir()
	name := walName(1, 11)

	// First uploader commits the segment.
	first := writeWalFile(t, dir, name)
	require.NoError(t, uploader.ProcessSegment(context.Background(), first, name))

	savesAfterFirst := store.saveCount.Load()

	// A second local copy of the same segment (another process) must not re-upload.
	secondDir := t.TempDir()
	second := writeWalFile(t, secondDir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), second, name))

	require.Equal(t, savesAfterFirst, store.saveCount.Load(), "completed segment must not be re-uploaded")
	require.NoFileExists(t, second, "redundant local copy must be removed")
}

func Test_WalUpload_DuplicateClaim_InFlightClaim_RetainsLocalNoStorageWrite(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, nil)

	name := walName(1, 13)
	startLSN := walmath.LSN(13 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	// Pre-seed an in-flight claim (file_name NULL) as if a live owner holds it.
	repo := physical_repositories.GetWalSegmentRepository()
	inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: name,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  backups_core_enums.BackupEncryptionNone,
	})
	require.NoError(t, err)
	require.True(t, inserted)

	dir := t.TempDir()
	local := writeWalFile(t, dir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), local, name))

	require.Zero(t, store.saveCount.Load(), "must not upload over another owner's in-flight claim")
	require.FileExists(t, local, "local file must be retained while another claim is in flight")

	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.Nil(t, row.FileName, "the in-flight claim must be left untouched")
}

func Test_WalUpload_DeleteCascadesIntoNullClaim_UploadCompletes_DeleteFileCleansOrphan(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	name := walName(1, 17)
	startLSN := walmath.LSN(17 * uint64(testWalSegmentSize))
	objectName := walSegmentObjectName(fixture.DB.ID, 1, name)

	store := newMockWalStorage()
	store.blockOn = objectName
	store.started = make(chan struct{})
	store.release = make(chan struct{})
	uploader := newTestUploader(fixture, store, nil)

	dir := t.TempDir()
	local := writeWalFile(t, dir, name)

	uploadErr := make(chan error, 1)
	go func() { uploadErr <- uploader.ProcessSegment(context.Background(), local, name) }()

	// Wait until SaveFile is mid-flight, then simulate a DeleteFull cascade
	// removing the NULL claim row out from under the uploader.
	<-store.started

	require.NoError(t, physical_repositories.GetWalSegmentRepository().DeleteClaim(
		findWalSegment(t, fixture.DB.ID, 1, startLSN).ID,
	))

	close(store.release)

	require.NoError(t, <-uploadErr)

	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, startLSN), "cascade-deleted claim must stay gone")
	require.False(t, store.hasObject(objectName), "orphaned ciphertext must be deleted after a lost commit")
	require.NoFileExists(t, local)
}

func Test_WalUpload_PostUpdateGapProbe_EmitsSingleNotificationPerOutage(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	var gapCount atomic.Int64

	var gapStart, gapEnd walmath.LSN

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, func(start, end walmath.LSN) {
		gapCount.Add(1)
		gapStart, gapEnd = start, end
	})

	dir := t.TempDir()

	// Commit segment 20, then a discontiguous backlog 22, 23, 24 (segment 21 was
	// lost to a storage outage). Only the first post-gap segment (22) is
	// non-contiguous, so exactly one notification must fire.
	for _, segNo := range []uint64{20, 22, 23, 24} {
		name := walName(1, segNo)
		require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir, name), name))
	}

	require.Equal(t, int64(1), gapCount.Load(), "a single multi-segment outage must notify exactly once")
	require.Equal(t, walmath.LSN(21*uint64(testWalSegmentSize)), gapStart)
	require.Equal(t, walmath.LSN(22*uint64(testWalSegmentSize)), gapEnd)
}

func Test_WalUpload_ContiguousSegments_NoGapNotification(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	var gapCount atomic.Int64

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, func(_, _ walmath.LSN) { gapCount.Add(1) })

	dir := t.TempDir()

	for _, segNo := range []uint64{30, 31, 32} {
		name := walName(1, segNo)
		require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir, name), name))
	}

	require.Zero(t, gapCount.Load(), "contiguous segments must not report a gap")
}

func Test_SegmentBounds_AtLogIdBoundary_ComputesContiguousLSNs(t *testing.T) {
	// The last segment in logid 0 (FF) and the first in logid 1 must yield
	// contiguous LSNs across the boundary — the invariant the post-upload gap
	// probe (prev.end_lsn == this.start_lsn) relies on.
	timelineID, start, end, err := segmentBounds("0000000100000000000000FF", testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, 1, timelineID)
	require.Equal(t, walmath.LSN(0xFF)*walmath.LSN(testWalSegmentSize), start)
	require.Equal(t, start+walmath.LSN(testWalSegmentSize), end)

	_, nextStart, _, err := segmentBounds("000000010000000100000000", testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, end, nextStart, "segment LSNs must be contiguous across the logid boundary")
}

func Test_WalUpload_RecoverSegment_TakesOverOwnNullClaim_CommitsExistingRow(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, nil)

	name := walName(1, 23)
	startLSN := walmath.LSN(23 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	// Seed a pre-crash NULL-file_name claim row, as the restart sweep would find.
	repo := physical_repositories.GetWalSegmentRepository()
	inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: name,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  backups_core_enums.BackupEncryptionNone,
	})
	require.NoError(t, err)
	require.True(t, inserted)

	claim := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, claim)
	require.Nil(t, claim.FileName)

	dir := t.TempDir()
	local := writeWalFile(t, dir, name)

	require.NoError(t, uploader.RecoverSegment(context.Background(), local, name))

	committed := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, committed)
	require.Equal(t, claim.ID, committed.ID, "takeover must reuse the pre-crash row, not insert a duplicate")
	require.NotNil(t, committed.FileName, "own pre-crash NULL claim must be taken over and committed")
	require.True(t, store.hasObject(*committed.FileName))
	require.True(t, store.hasObject(*committed.FileName+metadataSuffix))
	require.NoFileExists(t, local)
}

func Test_WalUpload_RecoverSegment_WhenAlreadyCommitted_DropsLocalNoReupload(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(fixture, store, nil)

	name := walName(1, 29)

	// A normal upload commits the segment (crash happened after MarkUploaded).
	dir1 := t.TempDir()
	require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir1, name), name))

	savesAfterCommit := store.saveCount.Load()

	// The restart sweep finds a leftover local copy whose row is already committed.
	dir2 := t.TempDir()
	local := writeWalFile(t, dir2, name)

	require.NoError(t, uploader.RecoverSegment(context.Background(), local, name))

	require.Equal(t, savesAfterCommit, store.saveCount.Load(), "already-committed segment must not be re-uploaded")
	require.NoFileExists(t, local, "redundant local copy must be removed on recovery")
}

func Test_BuildWalSegmentArtifactReader_WhenLargeSegment_StreamsAndCountsArtifact(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "000000010000000000000001")
	require.NoError(t, os.WriteFile(localPath, bytes.Repeat([]byte("x"), 2*1024*1024), 0o600))

	artifact, salt, iv, err := buildWalSegmentArtifactReader(
		localPath,
		backups_core_enums.BackupEncryptionNone,
		"",
		uuid.New(),
	)
	require.NoError(t, err)
	require.Empty(t, salt)
	require.Empty(t, iv)

	body, err := io.ReadAll(artifact)
	require.NoError(t, err)

	compressedSizeBytes, err := artifact.wait()
	require.NoError(t, err)
	require.Equal(t, int64(len(body)), compressedSizeBytes)
	require.NotEmpty(t, body)
}
