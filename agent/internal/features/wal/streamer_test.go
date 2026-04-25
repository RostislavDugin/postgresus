package wal

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-agent/internal/config"
	"databasus-agent/internal/features/api"
	"databasus-agent/internal/logger"
)

func Test_UploadSegment_SingleSegment_ServerReceivesCorrectHeadersAndBody(t *testing.T) {
	walDir := createTestWalDir(t)
	segmentContent := []byte("test-wal-segment-data-for-upload")
	writeTestSegment(t, walDir, "000000010000000100000001", segmentContent)

	var receivedHeaders http.Header
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = body

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	require.NotNil(t, receivedHeaders)
	assert.Equal(t, "test-token", receivedHeaders.Get("Authorization"))
	assert.Equal(t, "application/octet-stream", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "000000010000000100000001", receivedHeaders.Get("X-Wal-Segment-Name"))

	decompressed := decompressZstd(t, receivedBody)
	assert.Equal(t, segmentContent, decompressed)
}

func Test_UploadSegments_MultipleSegmentsOutOfOrder_UploadedInAscendingOrder(t *testing.T) {
	walDir := createTestWalDir(t)
	writeTestSegment(t, walDir, "000000010000000100000003", []byte("third"))
	writeTestSegment(t, walDir, "000000010000000100000001", []byte("first"))
	writeTestSegment(t, walDir, "000000010000000100000002", []byte("second"))

	var mu sync.Mutex
	var uploadOrder []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		uploadOrder = append(uploadOrder, r.Header.Get("X-Wal-Segment-Name"))
		mu.Unlock()

		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, uploadOrder, 3)
	assert.Equal(t, "000000010000000100000001", uploadOrder[0])
	assert.Equal(t, "000000010000000100000002", uploadOrder[1])
	assert.Equal(t, "000000010000000100000003", uploadOrder[2])
}

func Test_UploadSegments_DirectoryHasTmpFiles_TmpFilesIgnored(t *testing.T) {
	walDir := createTestWalDir(t)
	writeTestSegment(t, walDir, "000000010000000100000001", []byte("real segment"))
	writeTestSegment(t, walDir, "000000010000000100000002.tmp", []byte("partial copy"))

	var mu sync.Mutex
	var uploadedSegments []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		uploadedSegments = append(uploadedSegments, r.Header.Get("X-Wal-Segment-Name"))
		mu.Unlock()

		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, uploadedSegments, 1)
	assert.Equal(t, "000000010000000100000001", uploadedSegments[0])
}

func Test_UploadSegment_DeleteEnabled_FileRemovedAfterUpload(t *testing.T) {
	walDir := createTestWalDir(t)
	segmentName := "000000010000000100000001"
	writeTestSegment(t, walDir, segmentName, []byte("segment data"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	isDeleteEnabled := true
	cfg := createTestConfig(walDir, server.URL)
	cfg.IsDeleteWalAfterUpload = &isDeleteEnabled
	apiClient := api.NewClient(server.URL, cfg.Token, false, logger.GetLogger())
	streamer := NewStreamer(cfg, apiClient, logger.GetLogger())

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	_, err := os.Stat(filepath.Join(walDir, segmentName))
	assert.True(t, os.IsNotExist(err), "segment file should be deleted after successful upload")
}

func Test_UploadSegment_DeleteDisabled_FileKeptAfterUpload(t *testing.T) {
	walDir := createTestWalDir(t)
	segmentName := "000000010000000100000001"
	writeTestSegment(t, walDir, segmentName, []byte("segment data"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	isDeleteDisabled := false
	cfg := createTestConfig(walDir, server.URL)
	cfg.IsDeleteWalAfterUpload = &isDeleteDisabled
	apiClient := api.NewClient(server.URL, cfg.Token, false, logger.GetLogger())
	streamer := NewStreamer(cfg, apiClient, logger.GetLogger())

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	_, err := os.Stat(filepath.Join(walDir, segmentName))
	assert.NoError(t, err, "segment file should be kept when delete is disabled")
}

func Test_UploadSegment_ServerReturns500_FileKeptInQueue(t *testing.T) {
	walDir := createTestWalDir(t)
	segmentName := "000000010000000100000001"
	writeTestSegment(t, walDir, segmentName, []byte("segment data"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	_, err := os.Stat(filepath.Join(walDir, segmentName))
	assert.NoError(t, err, "segment file should remain in queue after server error")
}

func Test_ProcessQueue_EmptyDirectory_NoUploads(t *testing.T) {
	walDir := createTestWalDir(t)

	uploadCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadCount++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	assert.Equal(t, 0, uploadCount, "no uploads should occur for empty directory")
}

func Test_Run_ContextCancelled_StopsImmediately(t *testing.T) {
	walDir := createTestWalDir(t)

	streamer := newTestStreamer(walDir, "http://localhost:0")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})
	go func() {
		streamer.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run should have stopped immediately when context is already cancelled")
	}
}

func Test_UploadSegment_ServerReturns409_FileNotDeleted(t *testing.T) {
	walDir := createTestWalDir(t)
	segmentName := "000000010000000100000005"
	writeTestSegment(t, walDir, segmentName, []byte("gap segment"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)

		resp := map[string]string{
			"error":               "gap_detected",
			"expectedSegmentName": "000000010000000100000003",
			"receivedSegmentName": segmentName,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	go streamer.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	_, err := os.Stat(filepath.Join(walDir, segmentName))
	assert.NoError(t, err, "segment file should not be deleted on gap detection")
}

func Test_UploadSegment_WhenUploadStalls_FailsWithIdleTimeout(t *testing.T) {
	walDir := createTestWalDir(t)

	// Use incompressible random data to ensure TCP buffers fill up
	segmentContent := make([]byte, 1024*1024)
	_, err := rand.Read(segmentContent)
	require.NoError(t, err)

	writeTestSegment(t, walDir, "000000010000000100000001", segmentContent)

	var requestReceived atomic.Bool
	handlerDone := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived.Store(true)

		// Read one byte then stall — simulates a network stall
		buf := make([]byte, 1)
		_, _ = r.Body.Read(buf)
		<-handlerDone
	}))
	defer server.Close()
	defer close(handlerDone)

	origIdleTimeout := uploadIdleTimeout
	uploadIdleTimeout = 200 * time.Millisecond
	defer func() { uploadIdleTimeout = origIdleTimeout }()

	streamer := newTestStreamer(walDir, server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	uploadErr := streamer.uploadSegment(ctx, "000000010000000100000001")

	assert.Error(t, uploadErr, "upload should fail when stalled")
	assert.True(t, requestReceived.Load(), "server should have received the request")
	assert.Contains(t, uploadErr.Error(), "idle timeout", "error should mention idle timeout")

	_, statErr := os.Stat(filepath.Join(walDir, "000000010000000100000001"))
	assert.NoError(t, statErr, "segment file should remain in queue after idle timeout")
}

func newTestStreamer(walDir, serverURL string) *Streamer {
	cfg := createTestConfig(walDir, serverURL)
	apiClient := api.NewClient(serverURL, cfg.Token, false, logger.GetLogger())

	return NewStreamer(cfg, apiClient, logger.GetLogger())
}

func createTestWalDir(t *testing.T) string {
	t.Helper()

	baseDir := filepath.Join(".", ".test-tmp")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("failed to create base test dir: %v", err)
	}

	dir, err := os.MkdirTemp(baseDir, t.Name()+"-*")
	if err != nil {
		t.Fatalf("failed to create test wal dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

func writeTestSegment(t *testing.T, dir, name string, content []byte) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
		t.Fatalf("failed to write test segment %s: %v", name, err)
	}
}

func createTestConfig(walDir, serverURL string) *config.Config {
	isDeleteEnabled := true

	return &config.Config{
		DatabasusHost:          serverURL,
		DbID:                   "test-db-id",
		Token:                  "test-token",
		PgWalDir:               walDir,
		IsDeleteWalAfterUpload: &isDeleteEnabled,
	}
}

func decompressZstd(t *testing.T, data []byte) []byte {
	t.Helper()

	decoder, err := zstd.NewReader(nil)
	require.NoError(t, err)
	defer decoder.Close()

	decoded, err := decoder.DecodeAll(data, nil)
	require.NoError(t, err)

	return decoded
}
