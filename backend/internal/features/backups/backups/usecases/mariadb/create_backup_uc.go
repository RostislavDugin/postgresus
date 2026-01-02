package usecases_mariadb

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/config"
	common "databasus-backend/internal/features/backups/backups/common"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	backupTimeout               = 6 * time.Hour
	shutdownCheckInterval       = 1 * time.Second
	copyBufferSize              = 8 * 1024 * 1024
	progressReportIntervalMB    = 1.0
	zstdStorageCompressionLevel = 5
	exitCodeGenericError        = 1
	exitCodeConnectionError     = 2
)

type CreateMariadbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func (uc *CreateMariadbBackupUsecase) Execute(
	ctx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	db *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*common.BackupMetadata, error) {
	uc.logger.Info(
		"Creating MariaDB backup via mariadb-dump",
		"databaseId", db.ID,
		"storageId", storage.ID,
	)

	if !backupConfig.IsBackupsEnabled {
		return nil, fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
	}

	mdb := db.Mariadb
	if mdb == nil {
		return nil, fmt.Errorf("mariadb database configuration is required")
	}

	if mdb.Database == nil || *mdb.Database == "" {
		return nil, fmt.Errorf("database name is required for mariadb-dump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, mdb.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	args := uc.buildMariadbDumpArgs(mdb)

	return uc.streamToStorage(
		ctx,
		backupID,
		backupConfig,
		tools.GetMariadbExecutable(
			tools.MariadbExecutableMariadbDump,
			mdb.Version,
			config.GetEnv().EnvMode,
			config.GetEnv().MariadbInstallDir,
		),
		args,
		decryptedPassword,
		storage,
		backupProgressListener,
		mdb,
	)
}

func (uc *CreateMariadbBackupUsecase) buildMariadbDumpArgs(
	mdb *mariadbtypes.MariadbDatabase,
) []string {
	args := []string{
		"--host=" + mdb.Host,
		"--port=" + strconv.Itoa(mdb.Port),
		"--user=" + mdb.Username,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		"--quick",
		"--verbose",
	}

	args = append(args, "--compress")

	if mdb.IsHttps {
		args = append(args, "--ssl")
	}

	if mdb.Database != nil && *mdb.Database != "" {
		args = append(args, *mdb.Database)
	}

	return args
}

func (uc *CreateMariadbBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	mariadbBin string,
	args []string,
	password string,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
	mdbConfig *mariadbtypes.MariadbDatabase,
) (*common.BackupMetadata, error) {
	uc.logger.Info("Streaming MariaDB backup to storage", "mariadbBin", mariadbBin)

	ctx, cancel := uc.createBackupContext(parentCtx)
	defer cancel()

	myCnfFile, err := uc.createTempMyCnfFile(mdbConfig, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

	cmd := exec.CommandContext(ctx, mariadbBin, fullArgs...)
	uc.logger.Info("Executing MariaDB backup command", "command", cmd.String())

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	stderrCh := make(chan []byte, 1)
	go func() {
		stderrOutput, _ := io.ReadAll(pgStderr)
		stderrCh <- stderrOutput
	}()

	storageReader, storageWriter := io.Pipe()

	finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
		backupID,
		backupConfig,
		storageWriter,
	)
	if err != nil {
		return nil, err
	}

	zstdWriter, err := zstd.NewWriter(finalWriter,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(zstdStorageCompressionLevel)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd writer: %w", err)
	}
	countingWriter := common.NewCountingWriter(zstdWriter)

	saveErrCh := make(chan error, 1)
	go func() {
		saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
		saveErrCh <- saveErr
	}()

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mariadbBin), err)
	}

	copyResultCh := make(chan error, 1)
	bytesWrittenCh := make(chan int64, 1)
	go func() {
		bytesWritten, err := uc.copyWithShutdownCheck(
			ctx,
			countingWriter,
			pgStdout,
			backupProgressListener,
		)
		bytesWrittenCh <- bytesWritten
		copyResultCh <- err
	}()

	copyErr := <-copyResultCh
	bytesWritten := <-bytesWrittenCh
	waitErr := cmd.Wait()

	select {
	case <-ctx.Done():
		uc.cleanupOnCancellation(zstdWriter, encryptionWriter, storageWriter, saveErrCh)
		return nil, uc.checkCancellationReason()
	default:
	}

	if err := zstdWriter.Close(); err != nil {
		uc.logger.Error("Failed to close zstd writer", "error", err)
	}
	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-saveErrCh
		return nil, err
	}

	saveErr := <-saveErrCh
	stderrOutput := <-stderrCh

	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		return nil, uc.buildMariadbDumpErrorMessage(waitErr, stderrOutput, mariadbBin)
	case copyErr != nil:
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backupMetadata, nil
}

func (uc *CreateMariadbBackupUsecase) createTempMyCnfFile(
	mdbConfig *mariadbtypes.MariadbDatabase,
	password string,
) (string, error) {
	tempDir, err := os.MkdirTemp(config.GetEnv().TempFolder, "mycnf_"+uuid.New().String())
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	myCnfFile := filepath.Join(tempDir, ".my.cnf")

	content := fmt.Sprintf(`[client]
user=%s
password="%s"
host=%s
port=%d
`, mdbConfig.Username, tools.EscapeMariadbPassword(password), mdbConfig.Host, mdbConfig.Port)

	if mdbConfig.IsHttps {
		content += "ssl=true\n"
	} else {
		content += "ssl=false\n"
	}

	err = os.WriteFile(myCnfFile, []byte(content), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write .my.cnf: %w", err)
	}

	return myCnfFile, nil
}

func (uc *CreateMariadbBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	backupProgressListener func(completedMBs float64),
) (int64, error) {
	buf := make([]byte, copyBufferSize)
	var totalBytesWritten int64
	var lastReportedMB float64

	for {
		select {
		case <-ctx.Done():
			return totalBytesWritten, fmt.Errorf("copy cancelled: %w", ctx.Err())
		default:
		}

		if config.IsShouldShutdown() {
			return totalBytesWritten, fmt.Errorf("copy cancelled due to shutdown")
		}

		bytesRead, readErr := src.Read(buf)
		if bytesRead > 0 {
			writeResultCh := make(chan writeResult, 1)
			go func() {
				bytesWritten, writeErr := dst.Write(buf[0:bytesRead])
				writeResultCh <- writeResult{bytesWritten, writeErr}
			}()

			var bytesWritten int
			var writeErr error

			select {
			case <-ctx.Done():
				return totalBytesWritten, fmt.Errorf("copy cancelled during write: %w", ctx.Err())
			case result := <-writeResultCh:
				bytesWritten = result.bytesWritten
				writeErr = result.writeErr
			}

			if bytesWritten < 0 || bytesRead < bytesWritten {
				bytesWritten = 0
				if writeErr == nil {
					writeErr = fmt.Errorf("invalid write result")
				}
			}

			if writeErr != nil {
				return totalBytesWritten, writeErr
			}

			if bytesRead != bytesWritten {
				return totalBytesWritten, io.ErrShortWrite
			}

			totalBytesWritten += int64(bytesWritten)

			if backupProgressListener != nil {
				currentSizeMB := float64(totalBytesWritten) / (1024 * 1024)
				if currentSizeMB >= lastReportedMB+progressReportIntervalMB {
					backupProgressListener(currentSizeMB)
					lastReportedMB = currentSizeMB
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				return totalBytesWritten, readErr
			}
			break
		}
	}

	return totalBytesWritten, nil
}

func (uc *CreateMariadbBackupUsecase) createBackupContext(
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
			case <-parentCtx.Done():
				cancel()
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

func (uc *CreateMariadbBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, common.BackupMetadata, error) {
	metadata := common.BackupMetadata{}

	if backupConfig.Encryption != backups_config.BackupEncryptionEncrypted {
		metadata.Encryption = backups_config.BackupEncryptionNone
		uc.logger.Info("Encryption disabled for backup", "backupId", backupID)
		return storageWriter, nil, metadata, nil
	}

	salt, err := backup_encryption.GenerateSalt()
	if err != nil {
		return nil, nil, metadata, fmt.Errorf("failed to generate salt: %w", err)
	}

	nonce, err := backup_encryption.GenerateNonce()
	if err != nil {
		return nil, nil, metadata, fmt.Errorf("failed to generate nonce: %w", err)
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, nil, metadata, fmt.Errorf("failed to get master key: %w", err)
	}

	encWriter, err := backup_encryption.NewEncryptionWriter(
		storageWriter,
		masterKey,
		backupID,
		salt,
		nonce,
	)
	if err != nil {
		return nil, nil, metadata, fmt.Errorf("failed to create encrypting writer: %w", err)
	}

	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	nonceBase64 := base64.StdEncoding.EncodeToString(nonce)
	metadata.EncryptionSalt = &saltBase64
	metadata.EncryptionIV = &nonceBase64
	metadata.Encryption = backups_config.BackupEncryptionEncrypted

	uc.logger.Info("Encryption enabled for backup", "backupId", backupID)
	return encWriter, encWriter, metadata, nil
}

func (uc *CreateMariadbBackupUsecase) cleanupOnCancellation(
	zstdWriter *zstd.Encoder,
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
	saveErrCh chan error,
) {
	if zstdWriter != nil {
		go func() {
			if closeErr := zstdWriter.Close(); closeErr != nil {
				uc.logger.Error(
					"Failed to close zstd writer during cancellation",
					"error",
					closeErr,
				)
			}
		}()
	}

	if encryptionWriter != nil {
		go func() {
			if closeErr := encryptionWriter.Close(); closeErr != nil {
				uc.logger.Error(
					"Failed to close encrypting writer during cancellation",
					"error",
					closeErr,
				)
			}
		}()
	}

	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("Failed to close pipe writer during cancellation", "error", err)
	}

	<-saveErrCh
}

func (uc *CreateMariadbBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
) error {
	encryptionCloseErrCh := make(chan error, 1)
	if encryptionWriter != nil {
		go func() {
			closeErr := encryptionWriter.Close()
			if closeErr != nil {
				uc.logger.Error("Failed to close encrypting writer", "error", closeErr)
			}
			encryptionCloseErrCh <- closeErr
		}()
	} else {
		encryptionCloseErrCh <- nil
	}

	encryptionCloseErr := <-encryptionCloseErrCh
	if encryptionCloseErr != nil {
		if err := storageWriter.Close(); err != nil {
			uc.logger.Error("Failed to close pipe writer after encryption error", "error", err)
		}
		return fmt.Errorf("failed to close encryption writer: %w", encryptionCloseErr)
	}

	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("Failed to close pipe writer", "error", err)
		return err
	}

	return nil
}

func (uc *CreateMariadbBackupUsecase) checkCancellationReason() error {
	if config.IsShouldShutdown() {
		return fmt.Errorf("backup cancelled due to shutdown")
	}
	return fmt.Errorf("backup cancelled")
}

func (uc *CreateMariadbBackupUsecase) buildMariadbDumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	mariadbBin string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf(
		"%s failed: %v – stderr: %s",
		filepath.Base(mariadbBin),
		waitErr,
		stderrStr,
	)

	exitErr, ok := waitErr.(*exec.ExitError)
	if !ok {
		return errors.New(errorMsg)
	}

	exitCode := exitErr.ExitCode()

	if exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError {
		return uc.handleConnectionErrors(stderrStr)
	}

	return errors.New(errorMsg)
}

func (uc *CreateMariadbBackupUsecase) handleConnectionErrors(stderrStr string) error {
	if containsIgnoreCase(stderrStr, "access denied") {
		return fmt.Errorf(
			"MariaDB access denied. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "can't connect") ||
		containsIgnoreCase(stderrStr, "connection refused") {
		return fmt.Errorf(
			"MariaDB connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "unknown database") {
		return fmt.Errorf(
			"MariaDB database does not exist. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") {
		return fmt.Errorf(
			"MariaDB SSL connection failed. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MariaDB connection timeout. stderr: %s",
			stderrStr,
		)
	}

	return fmt.Errorf("MariaDB connection or authentication error. stderr: %s", stderrStr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
