package usecases_clickhouse

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"databasus-backend/internal/config"
	chmanifest "databasus-backend/internal/features/backups/backups/clickhouse_manifest"
	common "databasus-backend/internal/features/backups/backups/common"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	chtypes "databasus-backend/internal/features/databases/databases/clickhouse"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	backupTimeout            = 23 * time.Hour
	shutdownCheckInterval    = 1 * time.Second
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
)

type CreateClickhouseBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

func (uc *CreateClickhouseBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core.Backup,
	backupConfig *backups_config.BackupConfig,
	db *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*common.BackupMetadata, error) {
	uc.logger.Info(
		"creating ClickHouse backup",
		"databaseId", db.ID,
		"storageId", storage.ID,
	)

	ch := db.Clickhouse
	if ch == nil {
		return nil, errors.New("clickhouse database configuration is required")
	}
	if ch.Database == "" {
		return nil, errors.New("database name is required")
	}

	password, err := uc.fieldEncryptor.Decrypt(db.ID, ch.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	enumCtx, enumCancel := context.WithTimeout(ctx, 60*time.Second)
	defer enumCancel()

	conn, err := chtypes.OpenConn(enumCtx, ch, password)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			uc.logger.Error("failed to close clickhouse connection", "error", closeErr)
		}
	}()

	chVersion, err := chtypes.DetectClickhouseVersion(enumCtx, conn)
	if err != nil {
		return nil, err
	}

	if err := chtypes.RejectUnsupportedTypes(enumCtx, conn, ch.Database); err != nil {
		return nil, err
	}

	tables, mvs, err := chtypes.EnumerateAndValidate(enumCtx, conn, ch.Database)
	if err != nil {
		return nil, err
	}

	dbEngine, dbDDL, err := chtypes.GetDatabaseDDL(enumCtx, conn, ch.Database)
	if err != nil {
		return nil, err
	}

	header := chmanifest.Header{
		SchemaVersion:     chmanifest.SchemaVersion,
		CreatedAt:         time.Now().UTC(),
		ClickhouseVersion: string(chVersion),
		SourceDatabase:    ch.Database,
		SourceDBEngine:    dbEngine,
		Tables:            tables,
		MaterializedViews: mvs,
		Notes: "v1: *MergeTree family only; Variant/Dynamic/JSON rejected; " +
			"only TO-form materialized views supported; cross-table consistency not guaranteed",
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "ch-backup-"+backup.ID.String()+"-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	if err := os.Chmod(tempDir, 0o700); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("chmod temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	bgCtx, cancel := uc.createBackupContext(ctx)
	defer cancel()

	storageReader, storageWriter := io.Pipe()
	finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
		backup.ID, backupConfig, storageWriter,
	)
	if err != nil {
		return nil, err
	}
	countingWriter := common.NewCountingWriter(finalWriter)

	saveErrCh := make(chan error, 1)
	go func() {
		saveErr := storage.SaveFile(
			bgCtx,
			uc.fieldEncryptor,
			uc.logger,
			backup.FileName,
			storageReader,
		)
		// If the storage backend errored mid-stream (S3 5xx, network timeout,
		// disk full), the writer side of the pipe would otherwise block
		// forever on its next Write because the reader is gone. CloseWithError
		// makes pending writes return saveErr, so the producer goroutine can
		// observe the failure and call failPipeline instead of deadlocking.
		// On success the writer's own Close() already drained the pipe and
		// this is a no-op.
		if saveErr != nil {
			_ = storageReader.CloseWithError(saveErr)
		}
		saveErrCh <- saveErr
	}()

	cliBin := tools.GetClickhouseExecutable(tools.ClickhouseExecutableClient)
	if _, err := exec.LookPath(cliBin); err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
			fmt.Errorf("clickhouse-client binary not accessible at %s: %w", cliBin, err))
	}

	tw := tar.NewWriter(countingWriter)
	footer := chmanifest.Footer{}
	var totalBytes int64

	headerBytes, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
			fmt.Errorf("marshal header: %w", err))
	}
	if err := writeTarBytes(tw, "manifest.header.json", header.CreatedAt, headerBytes); err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
	}

	if err := writeTarBytes(tw, "db.sql", header.CreatedAt, []byte(dbDDL)); err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
	}

	for _, t := range tables {
		if config.IsShouldShutdown() || bgCtx.Err() != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
				errors.New("backup cancelled"))
		}

		size, sha, rowCount, err := uc.dumpTableEntries(
			bgCtx, conn, tw, ch, password, cliBin, tempDir,
			t, header.CreatedAt, backupProgressListener, &totalBytes,
		)
		if err != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
		}

		footer.Tables = append(footer.Tables, chmanifest.TableFooterEntry{
			ID:          t.ID,
			Database:    ch.Database,
			Name:        t.Name,
			NativeBytes: size,
			SHA256:      sha,
			RowCount:    rowCount,
		})
		totalBytes += size
	}

	for _, mv := range mvs {
		ddl, err := chtypes.GetTableDDL(bgCtx, conn, ch.Database, mv.Name)
		if err != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
		}
		cleanedDDL, _, err := chtypes.CleanDDLForBackup(ddl)
		if err != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
				fmt.Errorf("clean DDL for mv %s: %w", mv.Name, err))
		}
		prefix := path.Join("mvs", mv.ID)
		if err := writeTarBytes(tw, path.Join(prefix, "ddl.sql"), header.CreatedAt, []byte(ddl)); err != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
		}
		if err := writeTarBytes(tw, path.Join(prefix, "restore.sql"), header.CreatedAt, []byte(cleanedDDL)); err != nil {
			return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
		}
	}

	footer.TotalBytes = totalBytes
	footer.CompletedAt = time.Now().UTC()
	footer.Status = "OK"
	footerBytes, err := json.MarshalIndent(footer, "", "  ")
	if err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
			fmt.Errorf("marshal footer: %w", err))
	}
	if err := writeTarBytes(tw, "manifest.footer.json", footer.CompletedAt, footerBytes); err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh, err)
	}

	if err := tw.Close(); err != nil {
		return nil, uc.failPipeline(storageWriter, encryptionWriter, saveErrCh,
			fmt.Errorf("close tar writer: %w", err))
	}

	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-saveErrCh
		return nil, err
	}

	saveErr := <-saveErrCh
	if saveErr != nil {
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	if backupProgressListener != nil {
		backupProgressListener(float64(totalBytes) / (1024 * 1024))
	}

	uc.logger.Info(
		"clickhouse backup completed",
		"databaseId", db.ID,
		"tableCount", len(tables),
		"mvCount", len(mvs),
		"totalBytes", totalBytes,
	)

	return &backupMetadata, nil
}

// dumpTableEntries handles all four tar entries for one table:
//
//	tables/<id>/ddl.sql           — original DDL
//	tables/<id>/restore.sql       — DDL with ON CLUSTER + UUID stripped
//	tables/<id>/native.meta.json  — written BEFORE native bytes for restore-side validation
//	tables/<id>/native            — raw Native bytes (via temp file)
func (uc *CreateClickhouseBackupUsecase) dumpTableEntries(
	ctx context.Context,
	conn driver.Conn,
	tw *tar.Writer,
	ch *chtypes.ClickhouseDatabase,
	password, cliBin, tempDir string,
	t chmanifest.TableHeaderEntry,
	modTime time.Time,
	progressListener func(completedMBs float64),
	totalBytesPtr *int64,
) (int64, string, int64, error) {
	originalDDL, err := chtypes.GetTableDDL(ctx, conn, ch.Database, t.Name)
	if err != nil {
		return 0, "", 0, err
	}
	cleanedDDL, _, err := chtypes.CleanDDLForBackup(originalDDL)
	if err != nil {
		return 0, "", 0, fmt.Errorf("clean DDL for %s: %w", t.Name, err)
	}

	prefix := path.Join("tables", t.ID)

	if err := writeTarBytes(tw, path.Join(prefix, "ddl.sql"), modTime, []byte(originalDDL)); err != nil {
		return 0, "", 0, err
	}
	if err := writeTarBytes(tw, path.Join(prefix, "restore.sql"), modTime, []byte(cleanedDDL)); err != nil {
		return 0, "", 0, err
	}

	rowCount := readRowCount(ctx, conn, uc.logger, ch.Database, t.Name)

	nativePath := filepath.Join(tempDir, t.ID+".native")
	sha, size, err := dumpTableNative(ctx, cliBin, ch, password, t, nativePath, uc.logger)
	if err != nil {
		return 0, "", 0, err
	}

	ddlHash := sha256.Sum256([]byte(originalDDL))
	meta := chmanifest.NativeMeta{
		Columns:     t.Columns,
		NativeBytes: size,
		SHA256:      sha,
		RowCount:    rowCount,
		DDLHash:     hex.EncodeToString(ddlHash[:]),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return 0, "", 0, fmt.Errorf("marshal native meta for %s: %w", t.Name, err)
	}
	if err := writeTarBytes(tw, path.Join(prefix, "native.meta.json"), modTime, metaBytes); err != nil {
		return 0, "", 0, err
	}

	if err := writeTarFile(
		tw,
		path.Join(prefix, "native"),
		modTime,
		nativePath,
		size,
		progressListener,
		totalBytesPtr,
	); err != nil {
		return 0, "", 0, err
	}
	_ = os.Remove(nativePath)

	return size, sha, rowCount, nil
}

func readRowCount(ctx context.Context, conn driver.Conn, logger *slog.Logger, database, table string) int64 {
	q := fmt.Sprintf(
		"SELECT count() FROM `%s`.`%s`",
		strings.ReplaceAll(database, "`", "``"),
		strings.ReplaceAll(table, "`", "``"),
	)
	var rowCount int64
	if err := conn.QueryRow(ctx, q).Scan(&rowCount); err != nil {
		logger.Warn("failed to read row count", "table", table, "error", err)
		return -1
	}
	return rowCount
}

// dumpTableNative execs clickhouse-client to write Native bytes to nativePath,
// computes their sha256 (hex) on the fly, and returns (sha, size, err).
func dumpTableNative(
	ctx context.Context,
	cliBin string,
	ch *chtypes.ClickhouseDatabase,
	password string,
	t chmanifest.TableHeaderEntry,
	nativePath string,
	logger *slog.Logger,
) (string, int64, error) {
	cols := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		cols = append(cols, "`"+strings.ReplaceAll(c, "`", "``")+"`")
	}

	query := fmt.Sprintf(
		"SELECT %s FROM `%s`.`%s` FORMAT Native",
		strings.Join(cols, ", "),
		strings.ReplaceAll(ch.Database, "`", "``"),
		strings.ReplaceAll(t.Name, "`", "``"),
	)

	args := []string{
		"--host=" + ch.Host,
		"--port=" + fmt.Sprintf("%d", ch.Port),
		"--user=" + ch.Username,
		"--database=" + ch.Database,
		"--query=" + query,
	}
	if ch.IsHttps {
		args = append(args, "--secure")
		if !ch.IsStrictTls {
			args = append(args, "--accept-invalid-certificate")
		}
	}

	cmd := exec.CommandContext(ctx, cliBin, args...)
	cmd.Env = filterAndAppendEnv(os.Environ(), map[string]string{
		"CLICKHOUSE_PASSWORD": password,
		"LC_ALL":              "C.UTF-8",
		"LANG":                "C.UTF-8",
	})

	f, err := os.OpenFile(nativePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("open native temp file: %w", err)
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	cmd.Stdout = io.MultiWriter(f, hasher)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Debug("clickhouse-client SELECT FORMAT Native",
		"table", t.Name,
		"binary", filepath.Base(cliBin),
	)

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf(
			"clickhouse-client failed for %s.%s: %w (stderr: %s)",
			ch.Database, t.Name, err, strings.TrimSpace(stderr.String()),
		)
	}

	if err := f.Close(); err != nil {
		return "", 0, fmt.Errorf("close native temp file: %w", err)
	}

	info, err := os.Stat(nativePath)
	if err != nil {
		return "", 0, fmt.Errorf("stat native temp file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), info.Size(), nil
}

// filterAndAppendEnv drops any inherited CLICKHOUSE_* env var, then appends ours.
// Otherwise an operator-set CLICKHOUSE_HOST or CLICKHOUSE_PASSWORD would
// silently override our explicit values.
func filterAndAppendEnv(parent []string, ours map[string]string) []string {
	out := make([]string, 0, len(parent)+len(ours))
	for _, e := range parent {
		if !strings.HasPrefix(e, "CLICKHOUSE_") {
			out = append(out, e)
		}
	}
	for k, v := range ours {
		out = append(out, k+"="+v)
	}
	return out
}

func writeTarBytes(tw *tar.Writer, name string, modTime time.Time, data []byte) error {
	h := &tar.Header{
		Name:     name,
		Mode:     0o600,
		Size:     int64(len(data)),
		ModTime:  modTime,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("tar header for %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("tar write %s: %w", name, err)
	}
	return nil
}

// writeTarFile streams a temp-file to a tar entry, with shutdown checks every
// iteration and progress reporting every 1 MB. The total written is added to
// totalBytesPtr so the caller's progress accounting stays consistent.
func writeTarFile(
	tw *tar.Writer,
	name string,
	modTime time.Time,
	srcPath string,
	size int64,
	progressListener func(completedMBs float64),
	totalBytesPtr *int64,
) error {
	h := &tar.Header{
		Name:     name,
		Mode:     0o600,
		Size:     size,
		ModTime:  modTime,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("tar header for %s: %w", name, err)
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open native file %s: %w", srcPath, err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, copyBufferSize)
	var copied int64
	var lastReportedMB float64

	for {
		if config.IsShouldShutdown() {
			return errors.New("backup cancelled due to shutdown")
		}

		n, readErr := f.Read(buf)
		if n > 0 {
			written, writeErr := tw.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("tar write for %s: %w", name, writeErr)
			}
			if written != n {
				return io.ErrShortWrite
			}
			copied += int64(n)

			if progressListener != nil {
				cumulativeMB := float64(*totalBytesPtr+copied) / (1024 * 1024)
				if cumulativeMB-lastReportedMB >= progressReportIntervalMB {
					progressListener(cumulativeMB)
					lastReportedMB = cumulativeMB
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read native file %s: %w", srcPath, readErr)
		}
	}

	if copied != size {
		return fmt.Errorf("tar entry size mismatch: declared %d, copied %d", size, copied)
	}
	return nil
}

func (uc *CreateClickhouseBackupUsecase) failPipeline(
	storageWriter *io.PipeWriter,
	encryptionWriter *backup_encryption.EncryptionWriter,
	saveErrCh chan error,
	cause error,
) error {
	// Unblock the storage-save goroutine FIRST. EncryptionWriter.Close() may
	// flush buffered bytes by writing to the pipe; if the reader has already
	// exited (cancelled ctx, storage backend error), that write blocks forever.
	// CloseWithError makes any pending Write return immediately.
	_ = storageWriter.CloseWithError(cause)
	<-saveErrCh
	// Best-effort: release any encryption-writer state. We intentionally do
	// NOT propagate its error — the backup is already failing for `cause`.
	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			uc.logger.Debug("encryption writer close after pipeline failure", "error", err)
		}
	}
	return cause
}

func (uc *CreateClickhouseBackupUsecase) createBackupContext(
	parentCtx context.Context,
) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parentCtx, backupTimeout)

	go func() {
		ticker := time.NewTicker(shutdownCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if config.IsShouldShutdown() {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel
}

func (uc *CreateClickhouseBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, common.BackupMetadata, error) {
	backupMetadata := common.BackupMetadata{
		BackupID:   backupID,
		Encryption: backups_config.BackupEncryptionNone,
	}

	if backupConfig.Encryption != backups_config.BackupEncryptionEncrypted {
		return storageWriter, nil, backupMetadata, nil
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to get master key: %w", err)
	}

	encSetup, err := backup_encryption.SetupEncryptionWriter(storageWriter, masterKey, backupID)
	if err != nil {
		return nil, nil, backupMetadata, err
	}

	backupMetadata.Encryption = backups_config.BackupEncryptionEncrypted
	backupMetadata.EncryptionSalt = &encSetup.SaltBase64
	backupMetadata.EncryptionIV = &encSetup.NonceBase64

	return encSetup.Writer, encSetup.Writer, backupMetadata, nil
}

func (uc *CreateClickhouseBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
) error {
	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			uc.logger.Error("failed to close encryption writer", "error", err)
			return fmt.Errorf("failed to close encryption writer: %w", err)
		}
	}
	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("failed to close storage writer", "error", err)
		return fmt.Errorf("failed to close storage writer: %w", err)
	}
	return nil
}
