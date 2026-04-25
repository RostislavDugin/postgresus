package restore

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-agent/internal/features/api"
	"databasus-agent/internal/logger"
)

const (
	testRestorePlanPath     = "/api/v1/backups/postgres/wal/restore/plan"
	testRestoreDownloadPath = "/api/v1/backups/postgres/wal/restore/download"

	testFullBackupID = "full-backup-id-1234"
	testWalSegment1  = "000000010000000100000001"
	testWalSegment2  = "000000010000000100000002"
)

func Test_RunRestore_WhenBasebackupAndWalSegmentsAvailable_FilesExtractedAndRecoveryConfigured(t *testing.T) {
	tarFiles := map[string][]byte{
		"PG_VERSION":      []byte("16"),
		"base/1/somefile": []byte("table-data"),
	}
	zstdTarData := createZstdTar(t, tarFiles)
	walData1 := createZstdData(t, []byte("wal-segment-1-data"))
	walData2 := createZstdData(t, []byte("wal-segment-2-data"))

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:                  testFullBackupID,
					FullBackupWalStartSegment: testWalSegment1,
					FullBackupWalStopSegment:  testWalSegment1,
					PgVersion:                 "16",
					CreatedAt:                 time.Now().UTC(),
					SizeBytes:                 1024,
				},
				WalSegments: []api.RestorePlanWalSegment{
					{BackupID: "wal-1", SegmentName: testWalSegment1, SizeBytes: 512},
					{BackupID: "wal-2", SegmentName: testWalSegment2, SizeBytes: 512},
				},
				TotalSizeBytes:         2048,
				LatestAvailableSegment: testWalSegment2,
			})

		case testRestoreDownloadPath:
			backupID := r.URL.Query().Get("backupId")
			switch backupID {
			case testFullBackupID:
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(zstdTarData)
			case "wal-1":
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(walData1)
			case "wal-2":
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(walData2)
			default:
				w.WriteHeader(http.StatusBadRequest)
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	pgVersionContent, err := os.ReadFile(filepath.Join(targetDir, "PG_VERSION"))
	require.NoError(t, err)
	assert.Equal(t, "16", string(pgVersionContent))

	someFileContent, err := os.ReadFile(filepath.Join(targetDir, "base", "1", "somefile"))
	require.NoError(t, err)
	assert.Equal(t, "table-data", string(someFileContent))

	walSegment1Content, err := os.ReadFile(filepath.Join(targetDir, walRestoreDir, testWalSegment1))
	require.NoError(t, err)
	assert.Equal(t, "wal-segment-1-data", string(walSegment1Content))

	walSegment2Content, err := os.ReadFile(filepath.Join(targetDir, walRestoreDir, testWalSegment2))
	require.NoError(t, err)
	assert.Equal(t, "wal-segment-2-data", string(walSegment2Content))

	recoverySignalPath := filepath.Join(targetDir, "recovery.signal")
	recoverySignalInfo, err := os.Stat(recoverySignalPath)
	require.NoError(t, err)
	assert.Equal(t, int64(0), recoverySignalInfo.Size())

	autoConfContent, err := os.ReadFile(filepath.Join(targetDir, "postgresql.auto.conf"))
	require.NoError(t, err)
	autoConfStr := string(autoConfContent)

	assert.Contains(t, autoConfStr, "restore_command")
	assert.Contains(t, autoConfStr, walRestoreDir)
	assert.Contains(t, autoConfStr, "recovery_target_action = 'promote'")
	assert.Contains(t, autoConfStr, "recovery_end_command")
	assert.NotContains(t, autoConfStr, "recovery_target_time")
}

func Test_RunRestore_WhenTargetTimeProvided_RecoveryTargetTimeWrittenToConfig(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zstdTarData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "2026-02-28T14:30:00Z", "")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	autoConfContent, err := os.ReadFile(filepath.Join(targetDir, "postgresql.auto.conf"))
	require.NoError(t, err)

	assert.Contains(t, string(autoConfContent), "recovery_target_time = '2026-02-28T14:30:00Z'")
}

func Test_RunRestore_WhenPgDataDirNotEmpty_ReturnsError(t *testing.T) {
	targetDir := createTestTargetDir(t)

	err := os.WriteFile(filepath.Join(targetDir, "existing-file"), []byte("data"), 0o644)
	require.NoError(t, err)

	restorer := newTestRestorer("http://localhost:0", targetDir, "", "", "")

	err = restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func Test_RunRestore_WhenPgDataDirDoesNotExist_ReturnsError(t *testing.T) {
	nonExistentDir := filepath.Join(os.TempDir(), "databasus-test-nonexistent-dir-12345")

	restorer := newTestRestorer("http://localhost:0", nonExistentDir, "", "", "")

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func Test_RunRestore_WhenNoBackupsAvailable_ReturnsError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.GetRestorePlanErrorResponse{
			Error:   "no_backups",
			Message: "No full backups available",
		})
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No full backups available")
}

func Test_RunRestore_WhenWalChainBroken_ReturnsError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.GetRestorePlanErrorResponse{
			Error:                 "wal_chain_broken",
			Message:               "WAL chain broken",
			LastContiguousSegment: testWalSegment1,
		})
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAL chain broken")
	assert.Contains(t, err.Error(), testWalSegment1)
}

func Test_DownloadWalSegment_WhenFirstAttemptFails_RetriesAndSucceeds(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)
	walData := createZstdData(t, []byte("wal-segment-data"))

	var mu sync.Mutex
	var walDownloadAttempts int

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments: []api.RestorePlanWalSegment{
					{BackupID: "wal-1", SegmentName: testWalSegment1, SizeBytes: 512},
				},
				TotalSizeBytes:         1536,
				LatestAvailableSegment: testWalSegment1,
			})

		case testRestoreDownloadPath:
			backupID := r.URL.Query().Get("backupId")
			if backupID == testFullBackupID {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(zstdTarData)
				return
			}

			mu.Lock()
			walDownloadAttempts++
			attempt := walDownloadAttempts
			mu.Unlock()

			if attempt == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"storage unavailable"}`))
				return
			}

			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(walData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	origDelay := retryDelayOverride
	testDelay := 10 * time.Millisecond
	retryDelayOverride = &testDelay
	defer func() { retryDelayOverride = origDelay }()

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	mu.Lock()
	attempts := walDownloadAttempts
	mu.Unlock()

	assert.Equal(t, 2, attempts)

	walContent, err := os.ReadFile(filepath.Join(targetDir, walRestoreDir, testWalSegment1))
	require.NoError(t, err)
	assert.Equal(t, "wal-segment-data", string(walContent))
}

func Test_DownloadWalSegment_WhenAllAttemptsFail_ReturnsErrorWithSegmentName(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments: []api.RestorePlanWalSegment{
					{BackupID: "wal-1", SegmentName: testWalSegment1, SizeBytes: 512},
				},
				TotalSizeBytes:         1536,
				LatestAvailableSegment: testWalSegment1,
			})

		case testRestoreDownloadPath:
			backupID := r.URL.Query().Get("backupId")
			if backupID == testFullBackupID {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(zstdTarData)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"storage unavailable"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	origDelay := retryDelayOverride
	testDelay := 10 * time.Millisecond
	retryDelayOverride = &testDelay
	defer func() { retryDelayOverride = origDelay }()

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), testWalSegment1)
	assert.Contains(t, err.Error(), "3 attempts")
}

func Test_RunRestore_WhenInvalidTargetTimeFormat_ReturnsError(t *testing.T) {
	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer("http://localhost:0", targetDir, "", "not-a-valid-time", "")

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --target-time format")
}

func Test_RunRestore_WhenBasebackupDownloadFails_ReturnsError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"storage error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "basebackup download failed")
}

func Test_RunRestore_WhenNoWalSegmentsInPlan_BasebackupRestoredSuccessfully(t *testing.T) {
	tarFiles := map[string][]byte{
		"PG_VERSION":        []byte("16"),
		"global/pg_control": []byte("control-data"),
	}
	zstdTarData := createZstdTar(t, tarFiles)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zstdTarData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	pgVersionContent, err := os.ReadFile(filepath.Join(targetDir, "PG_VERSION"))
	require.NoError(t, err)
	assert.Equal(t, "16", string(pgVersionContent))

	walRestoreDirInfo, err := os.Stat(filepath.Join(targetDir, walRestoreDir))
	require.NoError(t, err)
	assert.True(t, walRestoreDirInfo.IsDir())

	_, err = os.Stat(filepath.Join(targetDir, "recovery.signal"))
	require.NoError(t, err)

	autoConfContent, err := os.ReadFile(filepath.Join(targetDir, "postgresql.auto.conf"))
	require.NoError(t, err)
	assert.Contains(t, string(autoConfContent), "restore_command")
}

func Test_RunRestore_WhenMakingApiCalls_AuthTokenIncludedInRequests(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)

	var receivedAuthHeaders atomic.Int32
	var mu sync.Mutex
	var authHeaderValues []string

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			receivedAuthHeaders.Add(1)

			mu.Lock()
			authHeaderValues = append(authHeaderValues, authHeader)
			mu.Unlock()
		}

		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zstdTarData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, int(receivedAuthHeaders.Load()), 2)

	mu.Lock()
	defer mu.Unlock()

	for _, headerValue := range authHeaderValues {
		assert.Equal(t, "test-token", headerValue)
	}
}

func Test_ConfigurePostgresRecovery_WhenPgTypeHost_UsesHostAbsolutePath(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zstdTarData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "host")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	autoConfContent, err := os.ReadFile(filepath.Join(targetDir, "postgresql.auto.conf"))
	require.NoError(t, err)
	autoConfStr := string(autoConfContent)

	absTargetDir, _ := filepath.Abs(targetDir)
	absTargetDir = filepath.ToSlash(absTargetDir)
	expectedWalPath := absTargetDir + "/" + walRestoreDir

	assert.Contains(t, autoConfStr, fmt.Sprintf("restore_command = 'cp %s/%%f %%p'", expectedWalPath))
	assert.Contains(t, autoConfStr, fmt.Sprintf("recovery_end_command = 'rm -rf %s'", expectedWalPath))
	assert.NotContains(t, autoConfStr, "/var/lib/postgresql/data")
}

func Test_ConfigurePostgresRecovery_WhenPgTypeDocker_UsesContainerPath(t *testing.T) {
	tarFiles := map[string][]byte{"PG_VERSION": []byte("16")}
	zstdTarData := createZstdTar(t, tarFiles)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testRestorePlanPath:
			writeJSON(w, api.GetRestorePlanResponse{
				FullBackup: api.RestorePlanFullBackup{
					BackupID:  testFullBackupID,
					PgVersion: "16",
					CreatedAt: time.Now().UTC(),
					SizeBytes: 1024,
				},
				WalSegments:            []api.RestorePlanWalSegment{},
				TotalSizeBytes:         1024,
				LatestAvailableSegment: "",
			})

		case testRestoreDownloadPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zstdTarData)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	targetDir := createTestTargetDir(t)
	restorer := newTestRestorer(server.URL, targetDir, "", "", "docker")

	err := restorer.Run(t.Context())
	require.NoError(t, err)

	autoConfContent, err := os.ReadFile(filepath.Join(targetDir, "postgresql.auto.conf"))
	require.NoError(t, err)
	autoConfStr := string(autoConfContent)

	expectedWalPath := "/var/lib/postgresql/data/" + walRestoreDir

	assert.Contains(t, autoConfStr, fmt.Sprintf("restore_command = 'cp %s/%%f %%p'", expectedWalPath))
	assert.Contains(t, autoConfStr, fmt.Sprintf("recovery_end_command = 'rm -rf %s'", expectedWalPath))

	absTargetDir, _ := filepath.Abs(targetDir)
	absTargetDir = filepath.ToSlash(absTargetDir)
	assert.NotContains(t, autoConfStr, absTargetDir)
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server
}

func createTestTargetDir(t *testing.T) string {
	t.Helper()

	baseDir := filepath.Join(".", ".test-tmp")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("failed to create base test dir: %v", err)
	}

	dir, err := os.MkdirTemp(baseDir, t.Name()+"-*")
	if err != nil {
		t.Fatalf("failed to create test target dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

func createZstdTar(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	createdDirs := make(map[string]bool)

	for name, content := range files {
		dir := filepath.Dir(name)
		if dir != "." && !createdDirs[dir] {
			parts := strings.Split(filepath.ToSlash(dir), "/")
			for partIndex := range parts {
				partialDir := strings.Join(parts[:partIndex+1], "/")
				if !createdDirs[partialDir] {
					err := tarWriter.WriteHeader(&tar.Header{
						Name:     partialDir + "/",
						Typeflag: tar.TypeDir,
						Mode:     0o755,
					})
					require.NoError(t, err)

					createdDirs[partialDir] = true
				}
			}
		}

		err := tarWriter.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0o644,
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)

		_, err = tarWriter.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())

	var zstdBuffer bytes.Buffer

	encoder, err := zstd.NewWriter(&zstdBuffer,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(5)),
		zstd.WithEncoderCRC(true),
	)
	require.NoError(t, err)

	_, err = encoder.Write(tarBuffer.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	return zstdBuffer.Bytes()
}

func createZstdData(t *testing.T, data []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer

	encoder, err := zstd.NewWriter(&buffer,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(5)),
		zstd.WithEncoderCRC(true),
	)
	require.NoError(t, err)

	_, err = encoder.Write(data)
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	return buffer.Bytes()
}

func newTestRestorer(serverURL, targetPgDataDir, backupID, targetTime, pgType string) *Restorer {
	apiClient := api.NewClient(serverURL, "test-token", false, logger.GetLogger())

	return NewRestorer(apiClient, logger.GetLogger(), targetPgDataDir, backupID, targetTime, pgType)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(value); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
