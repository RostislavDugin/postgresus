package s3_storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
)

// testInnerPartSize is the S3 minimum non-final part size (5 MiB). Combined with a tiny per-object
// part count it forces object rollover on a few tens of MiB instead of the production ~15.6 GiB.
const testInnerPartSize = 5 * 1024 * 1024

func Test_GetFile_LegacySingleObjectWithoutManifest_ReturnsContent(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(4096)

	fileName := uuid.NewString()
	putRawObject(t, rawClient, storage.S3Bucket, fileName, persistedBytes)

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)

	assert.Equal(t, persistedBytes, reassembledBytes)
}

func Test_DeleteFile_LegacySingleObjectWithoutManifest_RemovesObject(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)

	fileName := uuid.NewString()
	putRawObject(t, rawClient, storage.S3Bucket, fileName, generateBytes(4096))

	require.NoError(t, storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileName))

	assert.Empty(t, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	_, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	assert.Error(t, err)
}

func Test_SaveFile_StreamWithinOneInnerPart_WritesSingleObjectWithoutManifest(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	assert.ElementsMatch(t, []string{fileName}, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)
	assert.Equal(t, persistedBytes, reassembledBytes)
}

func Test_SaveFile_StreamExactlyOneInnerPart_WritesSingleObjectWithoutManifest(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(testInnerPartSize)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	assert.ElementsMatch(t, []string{fileName}, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)
	assert.Equal(t, persistedBytes, reassembledBytes)
}

func Test_SaveAndGetFile_StreamExceedsObjectSize_RoundTripsChunked(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)

	// 23 MiB with 5 MiB parts and 2 parts/object: object1 = parts 1-2, object2 = parts 3-4,
	// object3 = the trailing 3 MiB part, exercising multi-object rollover and a sub-part final chunk.
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	assert.ElementsMatch(t, []string{
		fileName + ".part000001",
		fileName + ".part000002",
		fileName + ".part000003",
		fileName + manifestSuffix,
	}, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)
	assert.Equal(t, persistedBytes, reassembledBytes)
}

func Test_GetFile_ChunkedBackupMissingPart_ReturnsError(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	require.NoError(
		t,
		rawClient.RemoveObject(t.Context(), storage.S3Bucket, fileName+".part000002", minio.RemoveObjectOptions{}),
	)

	reader, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	assert.Error(t, err, "a missing chunk must surface as a read error, never a silently truncated stream")
}

func Test_DeleteFile_ChunkedBackup_RemovesAllPartsAndManifest(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)
	require.NotEmpty(t, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	require.NoError(t, storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileName))

	assert.Empty(t, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))
}

func Test_SaveFile_CancelledMidChunkedUpload_LeavesNoOrphanParts(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)

	ctx, cancel := context.WithCancel(t.Context())

	// 5 MiB parts, 2 parts/object => 10 MiB objects. Deliver 30 MiB but trip cancellation after
	// ~12 MiB: object 1 (parts 1-2) is committed, then cancellation lands while object 2 is in
	// flight. Object 1 is the orphan that cleanup must remove even though ctx is already cancelled.
	source := &cancelAfterReader{
		data:     generateBytes(30 * 1024 * 1024),
		cancelAt: 12 * 1024 * 1024,
		cancel:   cancel,
	}

	fileName := uuid.NewString()
	err := storage.SaveFile(ctx, encryptor, discardLogger(), fileName, source)
	require.Error(t, err, "a cancelled upload must fail")

	assert.Empty(
		t,
		listObjectKeys(t, rawClient, storage.S3Bucket, fileName),
		"cancellation mid-upload must leave no orphan part objects behind",
	)
}

func Test_SaveGetDeleteFile_ChunkedWithPrefix_RoundTripsAndCleansUp(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	storage.S3Prefix = "team-a/nested"

	persistedBytes := generateBytes(23 * 1024 * 1024)
	fileName := uuid.NewString()

	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	// Parts and manifest land under the prefix, and the manifest must reference the prefixed keys so
	// read and delete reconstruct them correctly.
	assert.ElementsMatch(t, []string{
		"team-a/nested/" + fileName + ".part000001",
		"team-a/nested/" + fileName + ".part000002",
		"team-a/nested/" + fileName + ".part000003",
		"team-a/nested/" + fileName + manifestSuffix,
	}, listObjectKeys(t, rawClient, storage.S3Bucket, "team-a/"))

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)
	assert.Equal(t, persistedBytes, reassembledBytes)

	require.NoError(t, storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileName))
	assert.Empty(t, listObjectKeys(t, rawClient, storage.S3Bucket, "team-a/"))
}

func Test_SaveAndGetFile_EmptyStream_WritesSingleEmptyObjectWithoutManifest(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)

	fileName := uuid.NewString()
	require.NoError(t, storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(nil)))

	assert.ElementsMatch(t, []string{fileName}, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	assert.Empty(t, readWholeFile(t, storage, encryptor, fileName))
}

func Test_SaveAndGetFile_StreamExactlyFillsObjects_NoTrailingEmptyObject(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)

	// 20 MiB is exactly two 10 MiB objects (5 MiB part x 2). The boundary must not commit an empty
	// trailing object: the bucket holds two parts plus the manifest, not three.
	persistedBytes := generateBytes(20 * 1024 * 1024)
	fileName := uuid.NewString()

	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	assert.ElementsMatch(t, []string{
		fileName + ".part000001",
		fileName + ".part000002",
		fileName + manifestSuffix,
	}, listObjectKeys(t, rawClient, storage.S3Bucket, fileName))

	reassembledBytes := readWholeFile(t, storage, encryptor, fileName)
	assert.Equal(t, persistedBytes, reassembledBytes)
}

func Test_GetFile_ChunkedBackupCorruptedPart_ReturnsChecksumError(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	// Overwrite a committed 10 MiB part with same-length but different bytes: the size still matches
	// the manifest, so only the per-part sha256 can catch the corruption.
	corruptedSameSize := bytes.Repeat([]byte{0xAB}, 10*1024*1024)
	putRawObject(t, rawClient, storage.S3Bucket, fileName+".part000002", corruptedSameSize)

	reader, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func Test_GetFile_ChunkedBackupTruncatedPart_ReturnsSizeError(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	// Replace a committed 10 MiB part with a shorter object: the manifest expects 10 MiB, so the
	// length check must reject it before its bytes reach the restore.
	putRawObject(t, rawClient, storage.S3Bucket, fileName+".part000002", generateBytes(4*1024*1024))

	reader, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size mismatch")
}

func Test_GetFile_CorruptedManifest_ReturnsError(t *testing.T) {
	rawClient, storage, encryptor := setupChunkedStorage(t)
	persistedBytes := generateBytes(23 * 1024 * 1024)

	fileName := uuid.NewString()
	require.NoError(
		t,
		storage.SaveFile(t.Context(), encryptor, discardLogger(), fileName, bytes.NewReader(persistedBytes)),
	)

	// A manifest that exists but cannot be parsed must fail loudly, not fall through to the
	// single-object path and silently read nothing.
	putRawObject(t, rawClient, storage.S3Bucket, fileName+manifestSuffix, []byte("not a valid manifest{"))

	_, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest")
}

func Test_TestConnection_WhenDeleteIsDenied_Succeeds(t *testing.T) {
	storage, encryptor := setupProbeStubStorage(t, s3ProbeStub{deleteErrorCode: s3ErrCodeAccessDenied})

	assert.NoError(t, storage.TestConnection(encryptor))
}

func Test_TestConnection_WhenBucketIsMissing_ReturnsBucketDoesNotExist(t *testing.T) {
	storage, encryptor := setupProbeStubStorage(t, s3ProbeStub{putErrorCode: s3ErrCodeNoSuchBucket})

	err := storage.TestConnection(encryptor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func Test_TestConnection_WhenWriteIsDenied_ReturnsWriteAccessError(t *testing.T) {
	storage, encryptor := setupProbeStubStorage(t, s3ProbeStub{putErrorCode: s3ErrCodeAccessDenied})

	err := storage.TestConnection(encryptor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write to")
}

func Test_TestConnection_WhenReadIsDenied_ReturnsReadAccessError(t *testing.T) {
	storage, encryptor := setupProbeStubStorage(t, s3ProbeStub{getErrorCode: s3ErrCodeAccessDenied})

	err := storage.TestConnection(encryptor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read back from")
}

func Test_TestConnection_WhenReadReturnsDifferentContent_ReturnsMismatchError(t *testing.T) {
	storage, encryptor := setupProbeStubStorage(t, s3ProbeStub{servedContent: "corrupted"})

	err := storage.TestConnection(encryptor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "different content")
}

func setupChunkedStorage(t *testing.T) (*minio.Client, *S3Storage, encryption.FieldEncryptor) {
	t.Helper()

	endpoint := containers.StartMinio(t)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	bucketName := "test-bucket"

	rawClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Secure: false,
		Region: containers.MinioRegion,
	})
	require.NoError(t, err, "failed to create raw minio client")

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	require.NoError(t, rawClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: containers.MinioRegion}))

	encryptor := encryption.GetFieldEncryptor()

	accessKey, err := encryptor.Encrypt(containers.MinioRootUser)
	require.NoError(t, err)
	secretKey, err := encryptor.Encrypt(containers.MinioRootPassword)
	require.NoError(t, err)

	storage := &S3Storage{
		StorageID:             uuid.New(),
		S3Bucket:              bucketName,
		S3Region:              containers.MinioRegion,
		S3AccessKey:           accessKey,
		S3SecretKey:           secretKey,
		S3Endpoint:            "http://" + address,
		innerPartSizeOverride: testInnerPartSize,
		maxPartsOverride:      2,
	}

	return rawClient, storage, encryptor
}

func setupProbeStubStorage(
	t *testing.T,
	stub s3ProbeStub,
) (*S3Storage, encryption.FieldEncryptor) {
	t.Helper()

	server := startS3ProbeServer(t, stub)

	encryptor := encryption.GetFieldEncryptor()

	accessKey, err := encryptor.Encrypt(containers.MinioRootUser)
	require.NoError(t, err)
	secretKey, err := encryptor.Encrypt(containers.MinioRootPassword)
	require.NoError(t, err)

	storage := &S3Storage{
		StorageID:   uuid.New(),
		S3Bucket:    "test-bucket",
		S3Region:    containers.MinioRegion,
		S3AccessKey: accessKey,
		S3SecretKey: secretKey,
		S3Endpoint:  server.URL,
	}

	return storage, encryptor
}

// A real MinIO container cannot express these branches: its root credentials ignore bucket
// policies, so a denied write, read or delete is not reproducible against it.
type s3ProbeStub struct {
	putErrorCode    string
	getErrorCode    string
	deleteErrorCode string
	servedContent   string
}

func startS3ProbeServer(t *testing.T, stub s3ProbeStub) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if stub.putErrorCode != "" {
				writeS3ErrorResponse(w, stub.putErrorCode)
				return
			}

			w.Header().Set("ETag", `"probe"`)
		case http.MethodGet:
			if stub.getErrorCode != "" {
				writeS3ErrorResponse(w, stub.getErrorCode)
				return
			}

			content := stub.servedContent
			if content == "" {
				content = connectionTestObjectContent
			}

			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
			_, _ = w.Write([]byte(content))
		case http.MethodDelete:
			if stub.deleteErrorCode != "" {
				writeS3ErrorResponse(w, stub.deleteErrorCode)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func writeS3ErrorResponse(w http.ResponseWriter, code string) {
	statusByCode := map[string]int{
		s3ErrCodeNoSuchBucket: http.StatusNotFound,
		s3ErrCodeAccessDenied: http.StatusForbidden,
	}

	status, isKnownCode := statusByCode[code]
	if !isKnownCode {
		status = http.StatusForbidden
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(
		w,
		`<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`,
		code,
		code,
	)
}

func putRawObject(t *testing.T, client *minio.Client, bucket, key string, data []byte) {
	t.Helper()

	_, err := client.PutObject(
		t.Context(),
		bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{},
	)
	require.NoError(t, err, "failed to write raw object %s", key)
}

func readWholeFile(t *testing.T, storage *S3Storage, encryptor encryption.FieldEncryptor, fileName string) []byte {
	t.Helper()

	reader, err := storage.GetFile(t.Context(), encryptor, discardLogger(), fileName)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)

	return content
}

func listObjectKeys(t *testing.T, client *minio.Client, bucket, prefix string) []string {
	t.Helper()

	var keys []string
	for object := range client.ListObjects(t.Context(), bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		require.NoError(t, object.Err)
		keys = append(keys, object.Key)
	}

	return keys
}

// generateBytes returns deterministic, position-dependent bytes so a round trip can be compared
// exactly without holding a second random source.
func generateBytes(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i*7 + 13)
	}

	return data
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// cancelAfterReader serves data and cancels its context once cancelAt bytes have been read, to
// simulate a backup cancelled partway through a chunked upload.
type cancelAfterReader struct {
	data     []byte
	pos      int
	cancelAt int
	cancel   context.CancelFunc
}

func (r *cancelAfterReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n

	if r.cancel != nil && r.pos >= r.cancelAt {
		r.cancel()
		r.cancel = nil
	}

	return n, nil
}
