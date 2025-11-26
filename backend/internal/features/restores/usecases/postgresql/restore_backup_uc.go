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
	"postgresus-backend/internal/features/backups/backups"
	"postgresus-backend/internal/features/backups/backups/encryption"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	pgtypes "postgresus-backend/internal/features/databases/databases/postgresql"
	encryption_secrets "postgresus-backend/internal/features/encryption/secrets"
	"postgresus-backend/internal/features/restores/models"
	"postgresus-backend/internal/features/storages"
	util_encryption "postgresus-backend/internal/util/encryption"
	files_utils "postgresus-backend/internal/util/files"
	"postgresus-backend/internal/util/tools"

	"github.com/google/uuid"
)

type RestorePostgresqlBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestorePostgresqlBackupUsecase) Execute(
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypePostgres {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"Restoring PostgreSQL backup via pg_restore",
		"restoreId",
		restore.ID,
		"backupId",
		backup.ID,
	)

	pg := restoringToDB.Postgresql
	if pg == nil {
		return fmt.Errorf("postgresql configuration is required for restore")
	}

	if pg.Database == nil || *pg.Database == "" {
		return fmt.Errorf("target database name is required for pg_restore")
	}

	// Use parallel jobs based on CPU count (same as backup)
	// Cap between 1 and 8 to avoid overwhelming the server
	parallelJobs := max(1, min(backupConfig.CpuCount, 8))

	args := []string{
		"-Fc",                            // expect custom format (same as backup)
		"-j", strconv.Itoa(parallelJobs), // parallel jobs based on CPU count
		"--no-password", // Use environment variable for password, prevent prompts
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", *pg.Database,
		"--verbose",   // Add verbose output to help with debugging
		"--clean",     // Clean (drop) database objects before recreating them
		"--if-exists", // Use IF EXISTS when dropping objects
		"--no-owner",  // Skip restoring ownership
		"--no-acl",    // Skip restoring access privileges (GRANT/REVOKE commands)
	}

	return uc.restoreFromStorage(
		originalDB,
		tools.GetPostgresqlExecutable(
			pg.Version,
			"pg_restore",
			config.GetEnv().EnvMode,
			config.GetEnv().PostgresesInstallDir,
		),
		args,
		pg.Password,
		backup,
		storage,
		pg,
	)
}

// restoreFromStorage restores backup data from storage using pg_restore
func (uc *RestorePostgresqlBackupUsecase) restoreFromStorage(
	database *databases.Database,
	pgBin string,
	args []string,
	password string,
	backup *backups.Backup,
	storage *storages.Storage,
	pgConfig *pgtypes.PostgresqlDatabase,
) error {
	uc.logger.Info(
		"Restoring PostgreSQL backup from storage via temporary file",
		"pgBin",
		pgBin,
		"args",
		args,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
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

	// Create temporary .pgpass file for authentication
	pgpassFile, err := uc.createTempPgpassFile(pgConfig, password)
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

	if info, err := os.Stat(pgpassFile); err == nil {
		uc.logger.Info("Temporary .pgpass file created successfully",
			"pgpassFile", pgpassFile,
			"size", info.Size(),
			"mode", info.Mode(),
		)
	} else {
		return fmt.Errorf("failed to verify .pgpass file: %w", err)
	}

	// Download backup to temporary file
	tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
	if err != nil {
		return fmt.Errorf("failed to download backup to temporary file: %w", err)
	}
	defer cleanupFunc()

	// Add the temporary backup file as the last argument to pg_restore
	args = append(args, tempBackupFile)

	return uc.executePgRestore(ctx, database, pgBin, args, pgpassFile, pgConfig)
}

// downloadBackupToTempFile downloads backup data from storage to a temporary file
func (uc *RestorePostgresqlBackupUsecase) downloadBackupToTempFile(
	ctx context.Context,
	backup *backups.Backup,
	storage *storages.Storage,
) (string, func(), error) {
	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Create temporary directory for backup data
	tempDir, err := os.MkdirTemp(config.GetEnv().TempFolder, "restore_"+uuid.New().String())
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	cleanupFunc := func() {
		_ = os.RemoveAll(tempDir)
	}

	tempBackupFile := filepath.Join(tempDir, "backup.dump")

	// Get backup data from storage
	uc.logger.Info(
		"Downloading backup file from storage to temporary file",
		"backupId",
		backup.ID,
		"tempFile",
		tempBackupFile,
		"encrypted",
		backup.Encryption == backups_config.BackupEncryptionEncrypted,
	)
	fieldEncryptor := util_encryption.GetFieldEncryptor()
	rawReader, err := storage.GetFile(fieldEncryptor, backup.ID)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		if err := rawReader.Close(); err != nil {
			uc.logger.Error("Failed to close backup reader", "error", err)
		}
	}()

	// Create a reader that handles decryption if needed
	var backupReader io.Reader = rawReader
	if backup.Encryption == backups_config.BackupEncryptionEncrypted {
		// Validate encryption metadata
		if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("backup is encrypted but missing encryption metadata")
		}

		// Get master key
		masterKey, err := uc.secretKeyService.GetSecretKey()
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to get master key for decryption: %w", err)
		}

		// Decode salt and IV from base64
		salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to decode encryption salt: %w", err)
		}

		iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to decode encryption IV: %w", err)
		}

		// Create decryption reader
		decryptReader, err := encryption.NewDecryptionReader(
			rawReader,
			masterKey,
			backup.ID,
			salt,
			iv,
		)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to create decryption reader: %w", err)
		}

		backupReader = decryptReader
		uc.logger.Info("Using decryption for encrypted backup", "backupId", backup.ID)
	}

	// Create temporary backup file
	tempFile, err := os.Create(tempBackupFile)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to create temporary backup file: %w", err)
	}
	defer func() {
		if err := tempFile.Close(); err != nil {
			uc.logger.Error("Failed to close temporary file", "error", err)
		}
	}()

	// Copy backup data to temporary file with shutdown checks
	_, err = uc.copyWithShutdownCheck(ctx, tempFile, backupReader)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to write backup to temporary file: %w", err)
	}

	// Close the temp file to ensure all data is written - this is handled by defer
	// Removing explicit close to avoid double-close error

	uc.logger.Info("Backup file written to temporary location", "tempFile", tempBackupFile)
	return tempBackupFile, cleanupFunc, nil
}

// executePgRestore executes the pg_restore command with proper environment setup
func (uc *RestorePostgresqlBackupUsecase) executePgRestore(
	ctx context.Context,
	database *databases.Database,
	pgBin string,
	args []string,
	pgpassFile string,
	pgConfig *pgtypes.PostgresqlDatabase,
) error {
	cmd := exec.CommandContext(ctx, pgBin, args...)
	uc.logger.Info("Executing PostgreSQL restore command", "command", cmd.String())

	// Setup environment variables
	uc.setupPgRestoreEnvironment(cmd, pgpassFile, pgConfig)

	// Verify executable exists and is accessible
	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf(
			"PostgreSQL executable not found or not accessible: %s - %w",
			pgBin,
			err,
		)
	}

	// Get stderr to capture any error output
	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Capture stderr in a separate goroutine
	stderrCh := make(chan []byte, 1)
	go func() {
		stderrOutput, _ := io.ReadAll(pgStderr)
		stderrCh <- stderrOutput
	}()

	// Start pg_restore
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
	}

	// Wait for the restore to finish
	waitErr := cmd.Wait()
	stderrOutput := <-stderrCh

	// Check for shutdown before finalizing
	if config.IsShouldShutdown() {
		return fmt.Errorf("restore cancelled due to shutdown")
	}

	if waitErr != nil {
		if config.IsShouldShutdown() {
			return fmt.Errorf("restore cancelled due to shutdown")
		}

		return uc.handlePgRestoreError(database, waitErr, stderrOutput, pgBin, args, pgConfig)
	}

	return nil
}

// setupPgRestoreEnvironment configures environment variables for pg_restore
func (uc *RestorePostgresqlBackupUsecase) setupPgRestoreEnvironment(
	cmd *exec.Cmd,
	pgpassFile string,
	pgConfig *pgtypes.PostgresqlDatabase,
) {
	// Start with system environment variables
	cmd.Env = os.Environ()

	// Use the .pgpass file for authentication
	cmd.Env = append(cmd.Env, "PGPASSFILE="+pgpassFile)
	uc.logger.Info("Using temporary .pgpass file for authentication", "pgpassFile", pgpassFile)

	// Add PostgreSQL-specific environment variables
	cmd.Env = append(cmd.Env, "PGCLIENTENCODING=UTF8")
	cmd.Env = append(cmd.Env, "PGCONNECT_TIMEOUT=30")

	// Add encoding-related environment variables
	cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8")
	cmd.Env = append(cmd.Env, "LANG=C.UTF-8")
	cmd.Env = append(cmd.Env, "PGOPTIONS=--client-encoding=UTF8")

	shouldRequireSSL := pgConfig.IsHttps

	// Configure SSL settings
	if shouldRequireSSL {
		cmd.Env = append(cmd.Env, "PGSSLMODE=require")
		uc.logger.Info("Using required SSL mode", "configuredHttps", pgConfig.IsHttps)
	} else {
		cmd.Env = append(cmd.Env, "PGSSLMODE=prefer")
		uc.logger.Info("Using preferred SSL mode", "configuredHttps", pgConfig.IsHttps)
	}

	// Set other SSL parameters to avoid certificate issues
	cmd.Env = append(cmd.Env, "PGSSLCERT=")
	cmd.Env = append(cmd.Env, "PGSSLKEY=")
	cmd.Env = append(cmd.Env, "PGSSLROOTCERT=")
	cmd.Env = append(cmd.Env, "PGSSLCRL=")
}

// handlePgRestoreError processes and formats pg_restore errors
func (uc *RestorePostgresqlBackupUsecase) handlePgRestoreError(
	database *databases.Database,
	waitErr error,
	stderrOutput []byte,
	pgBin string,
	args []string,
	pgConfig *pgtypes.PostgresqlDatabase,
) error {
	// Enhanced error handling for PostgreSQL connection and restore issues
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

		if exitCode == 1 && strings.TrimSpace(stderrStr) == "" {
			errorMsg = fmt.Sprintf(
				"%s failed with exit status 1 but provided no error details. "+
					"This often indicates: "+
					"1) Connection timeout or refused connection, "+
					"2) Authentication failure with incorrect credentials, "+
					"3) Database does not exist, "+
					"4) Network connectivity issues, "+
					"5) PostgreSQL server not running, "+
					"6) Backup file is corrupted or incompatible. "+
					"Command executed: %s %s",
				filepath.Base(pgBin),
				pgBin,
				strings.Join(args, " "),
			)
		} else if exitCode == -1073741819 { // 0xC0000005 in decimal
			errorMsg = fmt.Sprintf(
				"%s crashed with access violation (0xC0000005). This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. stderr: %s",
				filepath.Base(pgBin),
				stderrStr,
			)
		} else if exitCode == 1 || exitCode == 2 {
			// Check for common connection and authentication issues
			if containsIgnoreCase(stderrStr, "pg_hba.conf") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection rejected by server configuration (pg_hba.conf). stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "no password supplied") || containsIgnoreCase(stderrStr, "fe_sendauth") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL authentication failed - no password supplied. stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "ssl") && containsIgnoreCase(stderrStr, "connection") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL SSL connection failed. stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "connection") && containsIgnoreCase(stderrStr, "refused") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection refused. Check if the server is running and accessible. stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "authentication") || containsIgnoreCase(stderrStr, "password") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL authentication failed. Check username and password. stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "timeout") {
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection timeout. stderr: %s",
					stderrStr,
				)
			} else if containsIgnoreCase(stderrStr, "database") && containsIgnoreCase(stderrStr, "does not exist") {
				backupDbName := "unknown"
				if database.Postgresql != nil && database.Postgresql.Database != nil {
					backupDbName = *database.Postgresql.Database
				}

				targetDbName := "unknown"
				if pgConfig.Database != nil {
					targetDbName = *pgConfig.Database
				}

				errorMsg = fmt.Sprintf(
					"Target database does not exist (backup db %s, not found %s). Create the database before restoring. stderr: %s",
					backupDbName,
					targetDbName,
					stderrStr,
				)
			}
		}
	}

	return errors.New(errorMsg)
}

// copyWithShutdownCheck copies data from src to dst while checking for shutdown
func (uc *RestorePostgresqlBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var totalBytesWritten int64

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
func (uc *RestorePostgresqlBackupUsecase) createTempPgpassFile(
	pgConfig *pgtypes.PostgresqlDatabase,
	password string,
) (string, error) {
	if pgConfig == nil || password == "" {
		return "", nil
	}

	// Escape special characters in password as per PostgreSQL .pgpass format
	// Per official PostgreSQL documentation: only backslash and colon need escaping
	escapedPassword := strings.NewReplacer(
		"\\", "\\\\",
		":", "\\:",
	).Replace(password)

	pgpassContent := fmt.Sprintf("%s:%d:*:%s:%s",
		pgConfig.Host,
		pgConfig.Port,
		pgConfig.Username,
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
