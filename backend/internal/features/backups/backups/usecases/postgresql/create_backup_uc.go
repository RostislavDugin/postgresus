package usecases_postgresql

import (
	"context"
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
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	pgtypes "postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/storages"
	"postgresus-backend/internal/util/tools"

	"github.com/google/uuid"
)

type CreatePostgresqlBackupUsecase struct {
	logger *slog.Logger
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
) error {
	uc.logger.Info(
		"Creating PostgreSQL backup via pg_dump custom format",
		"databaseId",
		db.ID,
		"storageId",
		storage.ID,
	)

	if !backupConfig.IsBackupsEnabled {
		return fmt.Errorf("backups are not enabled for this database: \"%s\"", db.Name)
	}

	pg := db.Postgresql

	if pg == nil {
		return fmt.Errorf("postgresql database configuration is required for pg_dump backups")
	}

	if pg.Database == nil || *pg.Database == "" {
		return fmt.Errorf("database name is required for pg_dump backups")
	}

	args := []string{
		"-Fc",           // custom format with built-in compression
		"--no-password", // Use environment variable for password, prevent prompts
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", *pg.Database,
		"--verbose", // Add verbose output to help with debugging
	}

	// Use zstd compression level 5 for PostgreSQL 16+ (better compression and speed)
	// Fall back to gzip compression level 5 for older versions (12-15)
	if pg.Version == tools.PostgresqlVersion12 || pg.Version == tools.PostgresqlVersion13 ||
		pg.Version == tools.PostgresqlVersion14 || pg.Version == tools.PostgresqlVersion15 {
		args = append(args, "-Z", "5")
		uc.logger.Info("Using gzip compression level 5 (zstd not available)", "version", pg.Version)
	} else {
		args = append(args, "--compress=zstd:5")
		uc.logger.Info("Using zstd compression level 5", "version", pg.Version)
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
		pg.Password,
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
) error {
	uc.logger.Info("Streaming PostgreSQL backup to storage", "pgBin", pgBin, "args", args)

	// if backup not fit into 23 hours, Postgresus
	// seems not to work for such database size
	ctx, cancel := context.WithTimeout(parentCtx, 23*time.Hour)
	defer cancel()

	// Monitor for shutdown and cancel context if needed
	go func() {
		ticker := time.NewTicker(1 * time.Second)
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

	// Create temporary .pgpass file as a more reliable alternative to PGPASSWORD
	pgpassFile, err := uc.createTempPgpassFile(db.Postgresql, password)
	if err != nil {
		return fmt.Errorf("failed to create temporary .pgpass file: %w", err)
	}
	defer func() {
		if pgpassFile != "" {
			_ = os.Remove(pgpassFile)
		}
	}()

	// Verify .pgpass file was created successfully
	if pgpassFile == "" {
		return fmt.Errorf("temporary .pgpass file was not created")
	}

	// Verify .pgpass file was created correctly
	if info, err := os.Stat(pgpassFile); err == nil {
		uc.logger.Info("Temporary .pgpass file created successfully",
			"pgpassFile", pgpassFile,
			"size", info.Size(),
			"mode", info.Mode(),
		)
	} else {
		return fmt.Errorf("failed to verify .pgpass file: %w", err)
	}

	cmd := exec.CommandContext(ctx, pgBin, args...)
	uc.logger.Info("Executing PostgreSQL backup command", "command", cmd.String())

	// Start with system environment variables to preserve Windows PATH, SystemRoot, etc.
	cmd.Env = os.Environ()

	// Use the .pgpass file for authentication
	cmd.Env = append(cmd.Env, "PGPASSFILE="+pgpassFile)
	uc.logger.Info("Using temporary .pgpass file for authentication", "pgpassFile", pgpassFile)

	// Debug password setup (without exposing the actual password)
	uc.logger.Info("Setting up PostgreSQL environment",
		"passwordLength", len(password),
		"passwordEmpty", password == "",
		"pgBin", pgBin,
		"usingPgpassFile", true,
		"parallelJobs", backupConfig.CpuCount,
	)

	// Add PostgreSQL-specific environment variables
	cmd.Env = append(cmd.Env, "PGCLIENTENCODING=UTF8")
	cmd.Env = append(cmd.Env, "PGCONNECT_TIMEOUT=30")

	// Add encoding-related environment variables to handle character encoding issues
	cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8")
	cmd.Env = append(cmd.Env, "LANG=C.UTF-8")

	// Add PostgreSQL-specific encoding settings
	cmd.Env = append(cmd.Env, "PGOPTIONS=--client-encoding=UTF8")

	shouldRequireSSL := db.Postgresql.IsHttps

	// Require SSL when explicitly configured
	if shouldRequireSSL {
		cmd.Env = append(cmd.Env, "PGSSLMODE=require")
		uc.logger.Info("Using required SSL mode", "configuredHttps", db.Postgresql.IsHttps)
	} else {
		// SSL not explicitly required, but prefer it if available
		cmd.Env = append(cmd.Env, "PGSSLMODE=prefer")
		uc.logger.Info("Using preferred SSL mode", "configuredHttps", db.Postgresql.IsHttps)
	}

	// Set other SSL parameters to avoid certificate issues
	cmd.Env = append(cmd.Env, "PGSSLCERT=")     // No client certificate
	cmd.Env = append(cmd.Env, "PGSSLKEY=")      // No client key
	cmd.Env = append(cmd.Env, "PGSSLROOTCERT=") // No root certificate verification
	cmd.Env = append(cmd.Env, "PGSSLCRL=")      // No certificate revocation list

	// Verify executable exists and is accessible
	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf(
			"PostgreSQL executable not found or not accessible: %s - %w",
			pgBin,
			err,
		)
	}

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Capture stderr in a separate goroutine to ensure we don't miss any error output
	stderrCh := make(chan []byte, 1)
	go func() {
		stderrOutput, _ := io.ReadAll(pgStderr)
		stderrCh <- stderrOutput
	}()

	// A pipe connecting pg_dump output → storage
	storageReader, storageWriter := io.Pipe()

	// Create a counting writer to track bytes
	countingWriter := &CountingWriter{writer: storageWriter}

	// The backup ID becomes the object key / filename in storage

	// Start streaming into storage in its own goroutine
	saveErrCh := make(chan error, 1)
	go func() {
		saveErrCh <- storage.SaveFile(uc.logger, backupID, storageReader)
	}()

	// Start pg_dump
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
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

	// Wait for the copy to finish first, then the dump process
	copyErr := <-copyResultCh
	bytesWritten := <-bytesWrittenCh
	waitErr := cmd.Wait()

	// Check for shutdown or cancellation before finalizing
	select {
	case <-ctx.Done():
		if pipeWriter, ok := countingWriter.writer.(*io.PipeWriter); ok {
			if err := pipeWriter.Close(); err != nil {
				uc.logger.Error("Failed to close counting writer", "error", err)
			}
		}

		<-saveErrCh // Wait for storage to finish

		if config.IsShouldShutdown() {
			return fmt.Errorf("backup cancelled due to shutdown")
		}
		return fmt.Errorf("backup cancelled")
	default:
	}

	// Close the pipe writer to signal end of data
	if pipeWriter, ok := countingWriter.writer.(*io.PipeWriter); ok {
		if err := pipeWriter.Close(); err != nil {
			uc.logger.Error("Failed to close counting writer", "error", err)
		}
	}

	// Wait until storage ends reading
	saveErr := <-saveErrCh
	stderrOutput := <-stderrCh

	// Send final sizing after backup is completed
	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		select {
		case <-ctx.Done():
			if config.IsShouldShutdown() {
				return fmt.Errorf("backup cancelled due to shutdown")
			}
			return fmt.Errorf("backup cancelled")
		default:
		}

		// Enhanced error handling for PostgreSQL connection and SSL issues
		stderrStr := string(stderrOutput)
		errorMsg := fmt.Sprintf(
			"%s failed: %v – stderr: %s",
			filepath.Base(pgBin),
			waitErr,
			stderrStr,
		)

		// Check for specific PostgreSQL error patterns
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			// Enhanced debugging for exit status 1 with empty stderr
			if exitCode == 1 && strings.TrimSpace(stderrStr) == "" {
				uc.logger.Error("pg_dump failed with exit status 1 but no stderr output",
					"pgBin", pgBin,
					"args", args,
					"env_vars", []string{
						"PGCLIENTENCODING=UTF8",
						"PGCONNECT_TIMEOUT=30",
						"LC_ALL=C.UTF-8",
						"LANG=C.UTF-8",
						"PGOPTIONS=--client-encoding=UTF8",
					},
				)

				errorMsg = fmt.Sprintf(
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
			} else if exitCode == -1073741819 { // 0xC0000005 in decimal
				uc.logger.Error("PostgreSQL tool crashed with access violation",
					"pgBin", pgBin,
					"args", args,
					"exitCode", fmt.Sprintf("0x%X", uint32(exitCode)),
				)

				errorMsg = fmt.Sprintf(
					"%s crashed with access violation (0xC0000005). This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. stderr: %s",
					filepath.Base(pgBin),
					stderrStr,
				)
			} else if exitCode == 1 || exitCode == 2 {
				// Check for common connection and authentication issues
				if containsIgnoreCase(stderrStr, "pg_hba.conf") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL connection rejected by server configuration (pg_hba.conf). The server may not allow connections from your IP address or may require different authentication settings. stderr: %s",
						stderrStr,
					)
				} else if containsIgnoreCase(stderrStr, "no password supplied") || containsIgnoreCase(stderrStr, "fe_sendauth") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL authentication failed - no password supplied. "+
							"PGPASSWORD environment variable may not be working correctly on this system. "+
							"Password length: %d, Password empty: %v. "+
							"Consider using a .pgpass file as an alternative. stderr: %s",
						len(password),
						password == "",
						stderrStr,
					)
				} else if containsIgnoreCase(stderrStr, "ssl") && containsIgnoreCase(stderrStr, "connection") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL SSL connection failed. The server may require SSL encryption or have SSL configuration issues. stderr: %s",
						stderrStr,
					)
				} else if containsIgnoreCase(stderrStr, "connection") && containsIgnoreCase(stderrStr, "refused") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL connection refused. Check if the server is running and accessible from your network. stderr: %s",
						stderrStr,
					)
				} else if containsIgnoreCase(stderrStr, "authentication") || containsIgnoreCase(stderrStr, "password") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL authentication failed. Check username and password. stderr: %s",
						stderrStr,
					)
				} else if containsIgnoreCase(stderrStr, "timeout") {
					errorMsg = fmt.Sprintf(
						"PostgreSQL connection timeout. The server may be unreachable or overloaded. stderr: %s",
						stderrStr,
					)
				}
			}
		}

		return errors.New(errorMsg)
	case copyErr != nil:
		select {
		case <-ctx.Done():
			if config.IsShouldShutdown() {
				return fmt.Errorf("backup cancelled due to shutdown")
			}
			return fmt.Errorf("backup cancelled")
		default:
		}

		return fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		select {
		case <-ctx.Done():
			if config.IsShouldShutdown() {
				return fmt.Errorf("backup cancelled due to shutdown")
			}
			return fmt.Errorf("backup cancelled")
		default:
		}

		return fmt.Errorf("save to storage: %w", saveErr)
	}

	return nil
}

// copyWithShutdownCheck copies data from src to dst while checking for shutdown
func (uc *CreatePostgresqlBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	backupProgressListener func(completedMBs float64),
) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var totalBytesWritten int64

	// Progress reporting interval - report every 1MB of data
	var lastReportedMB float64
	const reportIntervalMB = 1.0

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
			bytesWritten, writeErr := dst.Write(buf[0:bytesRead])
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

			// Report progress based on total size
			if backupProgressListener != nil {
				currentSizeMB := float64(totalBytesWritten) / (1024 * 1024)

				// Only report if we've written at least 1MB more data than last report
				if currentSizeMB >= lastReportedMB+reportIntervalMB {
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

// containsIgnoreCase checks if a string contains a substring, ignoring case
func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// createTempPgpassFile creates a temporary .pgpass file with the given password
func (uc *CreatePostgresqlBackupUsecase) createTempPgpassFile(
	pgConfig *pgtypes.PostgresqlDatabase,
	password string,
) (string, error) {
	if pgConfig == nil || password == "" {
		return "", nil
	}

	pgpassContent := fmt.Sprintf("%s:%d:*:%s:%s",
		pgConfig.Host,
		pgConfig.Port,
		pgConfig.Username,
		password,
	)

	// it always create unique directory like /tmp/pgpass-1234567890
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
