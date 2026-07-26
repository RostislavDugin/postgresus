package usecases_mysql

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
)

// The masking this test covers only happens when the client process is fed a real OS pipe
// by os/exec, so there is no HTTP-level way to reach it — hence a stub client binary and a
// deliberately truncated stream instead of a controller test.
func writeFailingMysqlClientStub(t *testing.T) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("stub client relies on a POSIX shell")
	}

	stubPath := filepath.Join(t.TempDir(), "mysql-stub")
	stubScript := "#!/bin/sh\ncat > /dev/null\necho \"ERROR 1064 (42000) at line 53465: You have an error in your SQL syntax\" >&2\nexit 1\n"

	require.NoError(t, os.WriteFile(stubPath, []byte(stubScript), 0o700))

	return stubPath
}

func compressToZstd(t *testing.T, payload []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer

	zstdWriter, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)

	_, err = zstdWriter.Write(payload)
	require.NoError(t, err)
	require.NoError(t, zstdWriter.Close())

	return compressed.Bytes()
}

func Test_ExecuteMysqlRestore_WhenBackupStreamTruncates_ReportsStreamFailureNotSyntaxError(
	t *testing.T,
) {
	dumpStatements := bytes.Repeat(
		[]byte("INSERT INTO orders VALUES ('2020-07-13 18:49:41');\n"),
		200_000,
	)
	compressedDump := compressToZstd(t, dumpStatements)
	truncatedDump := compressedDump[:len(compressedDump)*3/4]

	restoreUsecase := &RestoreMysqlBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMysqlRestore(
		t.Context(),
		&databases.Database{},
		writeFailingMysqlClientStub(t),
		nil,
		filepath.Join(t.TempDir(), ".my.cnf"),
		io.NopCloser(bytes.NewReader(truncatedDump)),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.Contains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.NotContains(t, restoreErr.Error(), "SQL syntax")
}

func Test_ExecuteMysqlRestore_WhenBackupStreamIsComplete_ReportsClientExitError(t *testing.T) {
	compressedDump := compressToZstd(t, []byte("INSERT INTO orders VALUES (1);\n"))

	restoreUsecase := &RestoreMysqlBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMysqlRestore(
		t.Context(),
		&databases.Database{},
		writeFailingMysqlClientStub(t),
		nil,
		filepath.Join(t.TempDir(), ".my.cnf"),
		io.NopCloser(bytes.NewReader(compressedDump)),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.Contains(t, restoreErr.Error(), "SQL syntax")
}
