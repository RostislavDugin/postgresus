package restores_core

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	io_utils "databasus-backend/internal/util/io"
)

func Test_GetBackupStreamFailure_StorageDropsMidStream_ReportsTheTransportErrorNotTheDecodeError(t *testing.T) {
	errConnectionReset := errors.New("connection reset by peer")
	compressedBackup := compressBackup(t, bytes.Repeat([]byte("INSERT INTO users VALUES (1);\n"), 1000))

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(&severingReader{
		content:    compressedBackup,
		failAfter:  len(compressedBackup) / 2,
		failWith:   errConnectionReset,
		readAtOnce: 64,
	})

	zstdReader, err := zstd.NewReader(storageReadFailureTracker)
	require.NoError(t, err)
	defer zstdReader.Close()

	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(zstdReader)

	_, copyErr := io.Copy(io.Discard, decodedStreamFailureTracker)
	require.Error(t, copyErr, "the truncated download must break decompression")

	streamFailure := GetBackupStreamFailure(BackupStreamTrackers{
		StorageRead:   storageReadFailureTracker,
		DecodedStream: decodedStreamFailureTracker,
	})

	require.ErrorIs(t, streamFailure, errConnectionReset)
}

func Test_GetBackupStreamFailure_StorageIntactButDecodeFails_ReportsTheDecodeError(t *testing.T) {
	errCorruptArchive := errors.New("corrupt zstd frame")

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))
	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(&severingReader{
		content:    []byte("payload"),
		failAfter:  2,
		failWith:   errCorruptArchive,
		readAtOnce: 1,
	})

	_, _ = io.Copy(io.Discard, storageReadFailureTracker)
	_, _ = io.Copy(io.Discard, decodedStreamFailureTracker)

	streamFailure := GetBackupStreamFailure(BackupStreamTrackers{
		StorageRead:   storageReadFailureTracker,
		DecodedStream: decodedStreamFailureTracker,
	})

	require.ErrorIs(t, streamFailure, errCorruptArchive)
}

func Test_GetBackupStreamFailure_BothStreamsCompleted_ReturnsNil(t *testing.T) {
	storageReadFailureTracker := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))
	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))

	_, _ = io.Copy(io.Discard, storageReadFailureTracker)
	_, _ = io.Copy(io.Discard, decodedStreamFailureTracker)

	assert.NoError(t, GetBackupStreamFailure(BackupStreamTrackers{
		StorageRead:   storageReadFailureTracker,
		DecodedStream: decodedStreamFailureTracker,
	}))
}

func compressBackup(t *testing.T, backupContent []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer

	zstdWriter, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)

	_, err = zstdWriter.Write(backupContent)
	require.NoError(t, err)
	require.NoError(t, zstdWriter.Close())

	return compressed.Bytes()
}

type severingReader struct {
	content        []byte
	failAfter      int
	failWith       error
	readAtOnce     int
	deliveredBytes int
}

func (r *severingReader) Read(p []byte) (int, error) {
	if r.deliveredBytes >= r.failAfter {
		return 0, r.failWith
	}

	readable := min(len(p), r.readAtOnce, r.failAfter-r.deliveredBytes)

	copy(p, r.content[r.deliveredBytes:r.deliveredBytes+readable])
	r.deliveredBytes += readable

	return readable, nil
}
