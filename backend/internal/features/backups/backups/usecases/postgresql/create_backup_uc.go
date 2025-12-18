package usecases_postgresql

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

	"postgresus-backend/internal/config"
	backup_encryption "postgresus-backend/internal/features/backups/backups/encryption"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	pgtypes "postgresus-backend/internal/features/databases/databases/postgresql"
	encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/features/storages"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/tools"

	"github.com/google/uuid"
)

const (
	backupTimeout            = 23 * time.Hour
	shutdownCheckInterval    = 1 * time.Second
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
	pgConnectTimeout         = 30
	compressionLevel         = 5
	exitCodeAccessViolation  = -1073741819
	exitCodeGenericError     = 1
	exitCodeConnectionError  = 2
)

type CreatePostgresqlBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

// Execute creates a backup of the database
func (uc *CreatePostgresqlBackupUsecase) Execute(
	ctx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	db *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(
		completedMBs float64,
	),
) (*BackupMetadata, error) {
	uc.logger.Info(
		"Creating PostgreSQL backup via pg_dump custom format",
		"databaseId",
		db.ID,
		"storageId",
		storage.ID,
	)

	if !backupConfig.IsBackupsEnabled {
		return nil, fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
	}

	pg := db.Postgresql

	if pg == nil {
		return nil, fmt.Errorf("postgresql database configuration is required for pg_dump backups")
	}

	if pg.Database == nil || *pg.Database == "" {
		return nil, fmt.Errorf("database name is required for pg_dump backups")
	}

	args := uc.buildPgDumpArgs(pg)

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, pg.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	return uc.streamToStorage(
		ctx,
		backupID,
		backupConfig,
		tools.GetPostgresqlExecutable(
			pg.Version,
			"pg_dump",
			config.GetEnv().EnvMode,
			config.GetEnv().PostgresesInstallDir,
		),
		args,
		decryptedPassword,
		storage,
		db,
		backupProgressListener,
	)
}

// streamToStorage streams pg_dump output directly to storage
func (uc *CreatePostgresqlBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	pgBin string,
	args []string,
	password string,
	storage *storages.Storage,
	db *databases.Database,
	backupProgressListener func(completedMBs float64),
) (*BackupMetadata, error) {
	uc.logger.Info("Streaming PostgreSQL backup to storage", "pgBin", pgBin, "args", args)

	ctx, cancel := uc.createBackupContext(parentCtx)
	defer cancel()

	pgpassFile, err := uc.setupPgpassFile(db.Postgresql, password)
	if err != nil {
		return nil, err
	}
	defer func() {
		if pgpassFile != "" {
			// Remove the entire temp directory (which contains the .pgpass file)
			_ = os.RemoveAll(filepath.Dir(pgpassFile))
		}
	}()

	cmd := exec.CommandContext(ctx, pgBin, args...)
	uc.logger.Info("Executing PostgreSQL backup command", "command", cmd.String())

	if err := uc.setupPgEnvironment(cmd, pgpassFile, db.Postgresql.IsHttps, password, backupConfig.CpuCount, pgBin); err != nil {
		return nil, err
	}

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// Capture stderr in a separate goroutine to ensure we don't miss any error output
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

	countingWriter := &CountingWriter{writer: finalWriter}

	// The backup ID becomes the object key / filename in storage

	// Start streaming into storage in its own goroutine
	saveErrCh := make(chan error, 1)
	go func() {
		saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
		saveErrCh <- saveErr
	}()

	// Start pg_dump
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
	}

	// Copy pg output directly to storage with shutdown checks
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
		uc.cleanupOnCancellation(encryptionWriter, storageWriter, saveErrCh)
		return nil, uc.checkCancellationReason()
	default:
	}

	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-saveErrCh
		return nil, err
	}

	saveErr := <-saveErrCh
	stderrOutput := <-stderrCh

	// Send final sizing after backup is completed
	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		if err := uc.checkCancellation(ctx); err != nil {
			return nil, err
		}
		return nil, uc.buildPgDumpErrorMessage(waitErr, stderrOutput, pgBin, args, password)
	case copyErr != nil:
		if err := uc.checkCancellation(ctx); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		if err := uc.checkCancellation(ctx); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backupMetadata, nil
}

func (uc *CreatePostgresqlBackupUsecase) copyWithShutdownCheck(
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

func (uc *CreatePostgresqlBackupUsecase) buildPgDumpArgs(pg *pgtypes.PostgresqlDatabase) []string {
	args := []string{
		"-Fc",
		"--no-password",
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", *pg.Database,
		"--verbose",
	}

	for _, schema := range pg.IncludeSchemas {
		args = append(args, "-n", schema)
	}

	compressionArgs := uc.getCompressionArgs(pg.Version)
	return append(args, compressionArgs...)
}

func (uc *CreatePostgresqlBackupUsecase) getCompressionArgs(
	version tools.PostgresqlVersion,
) []string {
	if uc.isOlderPostgresVersion(version) {
		uc.logger.Info("Using gzip compression level 5 (zstd not available)", "version", version)
		return []string{"-Z", strconv.Itoa(compressionLevel)}
	}

	uc.logger.Info("Using zstd compression level 5", "version", version)
	return []string{fmt.Sprintf("--compress=zstd:%d", compressionLevel)}
}

func (uc *CreatePostgresqlBackupUsecase) isOlderPostgresVersion(
	version tools.PostgresqlVersion,
) bool {
	return version == tools.PostgresqlVersion12 ||
		version == tools.PostgresqlVersion13 ||
		version == tools.PostgresqlVersion14 ||
		version == tools.PostgresqlVersion15
}

func (uc *CreatePostgresqlBackupUsecase) createBackupContext(
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

func (uc *CreatePostgresqlBackupUsecase) setupPgpassFile(
	pgConfig *pgtypes.PostgresqlDatabase,
	password string,
) (string, error) {
	pgpassFile, err := uc.createTempPgpassFile(pgConfig, password)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary .pgpass file: %w", err)
	}

	if pgpassFile == "" {
		return "", fmt.Errorf("temporary .pgpass file was not created")
	}

	if info, err := os.Stat(pgpassFile); err == nil {
		uc.logger.Info("Temporary .pgpass file created successfully",
			"pgpassFile", pgpassFile,
			"size", info.Size(),
			"mode", info.Mode(),
		)
	} else {
		return "", fmt.Errorf("failed to verify .pgpass file: %w", err)
	}

	return pgpassFile, nil
}

func (uc *CreatePostgresqlBackupUsecase) setupPgEnvironment(
	cmd *exec.Cmd,
	pgpassFile string,
	shouldRequireSSL bool,
	password string,
	cpuCount int,
	pgBin string,
) error {
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PGPASSFILE="+pgpassFile)

	uc.logger.Info("Using temporary .pgpass file for authentication", "pgpassFile", pgpassFile)
	uc.logger.Info("Setting up PostgreSQL environment",
		"passwordLength", len(password),
		"passwordEmpty", password == "",
		"pgBin", pgBin,
		"usingPgpassFile", true,
		"parallelJobs", cpuCount,
	)

	cmd.Env = append(cmd.Env,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT="+strconv.Itoa(pgConnectTimeout),
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	if shouldRequireSSL {
		cmd.Env = append(cmd.Env, "PGSSLMODE=require")
		uc.logger.Info("Using required SSL mode", "configuredHttps", shouldRequireSSL)
	} else {
		cmd.Env = append(cmd.Env, "PGSSLMODE=prefer")
		uc.logger.Info("Using preferred SSL mode", "configuredHttps", shouldRequireSSL)
	}

	cmd.Env = append(cmd.Env,
		"PGSSLCERT=",
		"PGSSLKEY=",
		"PGSSLROOTCERT=",
		"PGSSLCRL=",
	)

	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf("PostgreSQL executable not found or not accessible: %s - %w", pgBin, err)
	}

	return nil
}

func (uc *CreatePostgresqlBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, BackupMetadata, error) {
	metadata := BackupMetadata{}

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

func (uc *CreatePostgresqlBackupUsecase) cleanupOnCancellation(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
	saveErrCh chan error,
) {
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

func (uc *CreatePostgresqlBackupUsecase) closeWriters(
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

func (uc *CreatePostgresqlBackupUsecase) checkCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if config.IsShouldShutdown() {
			return fmt.Errorf("backup cancelled due to shutdown")
		}
		return fmt.Errorf("backup cancelled")
	default:
		return nil
	}
}

func (uc *CreatePostgresqlBackupUsecase) checkCancellationReason() error {
	if config.IsShouldShutdown() {
		return fmt.Errorf("backup cancelled due to shutdown")
	}
	return fmt.Errorf("backup cancelled")
}

func (uc *CreatePostgresqlBackupUsecase) buildPgDumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	pgBin string,
	args []string,
	password string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf("%s failed: %v – stderr: %s", filepath.Base(pgBin), waitErr, stderrStr)

	exitErr, ok := waitErr.(*exec.ExitError)
	if !ok {
		return errors.New(errorMsg)
	}

	exitCode := exitErr.ExitCode()

	if exitCode == exitCodeGenericError && strings.TrimSpace(stderrStr) == "" {
		return uc.handleExitCode1NoStderr(pgBin, args)
	}

	if exitCode == exitCodeAccessViolation {
		return uc.handleAccessViolation(pgBin, stderrStr)
	}

	if exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError {
		return uc.handleConnectionErrors(stderrStr, password)
	}

	return errors.New(errorMsg)
}

func (uc *CreatePostgresqlBackupUsecase) handleExitCode1NoStderr(
	pgBin string,
	args []string,
) error {
	uc.logger.Error("pg_dump failed with exit status 1 but no stderr output",
		"pgBin", pgBin,
		"args", args,
		"env_vars", []string{
			"PGCLIENTENCODING=UTF8",
			"PGCONNECT_TIMEOUT=" + strconv.Itoa(pgConnectTimeout),
			"LC_ALL=C.UTF-8",
			"LANG=C.UTF-8",
		},
	)

	return fmt.Errorf(
		"%s failed with exit status 1 but provided no error details. "+
			"This often indicates: "+
			"1) Connection timeout or refused connection, "+
			"2) Authentication failure with incorrect credentials, "+
			"3) Database does not exist, "+
			"4) Network connectivity issues, "+
			"5) PostgreSQL server not running. "+
			"Command executed: %s %s",
		filepath.Base(pgBin),
		pgBin,
		strings.Join(args, " "),
	)
}

func (uc *CreatePostgresqlBackupUsecase) handleAccessViolation(
	pgBin string,
	stderrStr string,
) error {
	uc.logger.Error("PostgreSQL tool crashed with access violation",
		"pgBin", pgBin,
		"exitCode", "0xC0000005",
	)

	return fmt.Errorf(
		"%s crashed with access violation (0xC0000005). "+
			"This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. "+
			"stderr: %s",
		filepath.Base(pgBin),
		stderrStr,
	)
}

func (uc *CreatePostgresqlBackupUsecase) handleConnectionErrors(
	stderrStr string,
	password string,
) error {
	if containsIgnoreCase(stderrStr, "pg_hba.conf") {
		return fmt.Errorf(
			"PostgreSQL connection rejected by server configuration (pg_hba.conf). "+
				"The server may not allow connections from your IP address or may require different authentication settings. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "no password supplied") ||
		containsIgnoreCase(stderrStr, "fe_sendauth") {
		return fmt.Errorf(
			"PostgreSQL authentication failed - no password supplied. "+
				"PGPASSWORD environment variable may not be working correctly on this system. "+
				"Password length: %d, Password empty: %v. "+
				"Consider using a .pgpass file as an alternative. "+
				"stderr: %s",
			len(password),
			password == "",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") && containsIgnoreCase(stderrStr, "connection") {
		return fmt.Errorf(
			"PostgreSQL SSL connection failed. "+
				"The server may require SSL encryption or have SSL configuration issues. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "connection") && containsIgnoreCase(stderrStr, "refused") {
		return fmt.Errorf(
			"PostgreSQL connection refused. "+
				"Check if the server is running and accessible from your network. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "authentication") ||
		containsIgnoreCase(stderrStr, "password") {
		return fmt.Errorf(
			"PostgreSQL authentication failed. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"PostgreSQL connection timeout. The server may be unreachable or overloaded. stderr: %s",
			stderrStr,
		)
	}

	return fmt.Errorf("PostgreSQL connection or authentication error. stderr: %s", stderrStr)
}

func (uc *CreatePostgresqlBackupUsecase) createTempPgpassFile(
	pgConfig *pgtypes.PostgresqlDatabase,
	password string,
) (string, error) {
	if pgConfig == nil || password == "" {
		return "", nil
	}

	escapedHost := tools.EscapePgpassField(pgConfig.Host)
	escapedUsername := tools.EscapePgpassField(pgConfig.Username)
	escapedPassword := tools.EscapePgpassField(password)

	pgpassContent := fmt.Sprintf("%s:%d:*:%s:%s",
		escapedHost,
		pgConfig.Port,
		escapedUsername,
		escapedPassword,
	)

	tempDir, err := os.MkdirTemp("", "pgpass")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	pgpassFile := filepath.Join(tempDir, ".pgpass")
	err = os.WriteFile(pgpassFile, []byte(pgpassContent), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write temporary .pgpass file: %w", err)
	}

	return pgpassFile, nil
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
