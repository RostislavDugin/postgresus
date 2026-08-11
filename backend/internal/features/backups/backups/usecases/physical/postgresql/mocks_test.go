package usecases_physical_postgresql

import (
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

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

const testWalSegmentSize = int64(16 * 1024 * 1024)

// Used when a chain-span query must match every segment regardless of position.
const lsnSpanUpperBoundForTests = walmath.LSN(1) << 62

type mockWalStorage struct {
	mu        sync.Mutex
	saved     map[string][]byte
	deleted   []string
	saveCount atomic.Int64

	failSaveTimes int

	isFailingAllSaves atomic.Bool

	// Interleaves the DeleteFull cascade race: SaveFile for this exact object name
	// signals started and waits on release before returning.
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

	if m.isFailingAllSaves.Load() {
		return fmt.Errorf("mock storage is failing every save for %s", fileName)
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

func (m *mockWalStorage) startFailingSaves() {
	m.isFailingAllSaves.Store(true)
}

func (m *mockWalStorage) stopFailingSaves() {
	m.isFailingAllSaves.Store(false)
}

func (m *mockWalStorage) DeleteFile(_ encryption.FieldEncryptor, fileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.saved, fileName)
	m.deleted = append(m.deleted, fileName)

	return nil
}

func (m *mockWalStorage) GetFile(
	_ context.Context,
	_ encryption.FieldEncryptor,
	_ *slog.Logger,
	_ string,
) (io.ReadCloser, error) {
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

// 256 segments per logid, i.e. the default 16 MB segment size.
func walName(timeline uint32, segNo uint64) string {
	const segmentsPerLogID = 256

	return fmt.Sprintf("%08X%08X%08X", timeline, segNo/segmentsPerLogID, segNo%segmentsPerLogID)
}

// The segments are sparse: only the reported size matters to the uploader and to
// the resume-point math, so allocating real bytes would only slow tests down.
func writeSegmentOfSize(t *testing.T, dir, name string, sizeBytes int64) string {
	t.Helper()

	path := filepath.Join(dir, name)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	require.NoError(t, err)

	require.NoError(t, file.Truncate(sizeBytes))
	require.NoError(t, file.Close())

	return path
}

func writeWalFile(t *testing.T, dir, name string) string {
	t.Helper()

	return writeSegmentOfSize(t, dir, name, testWalSegmentSize)
}
