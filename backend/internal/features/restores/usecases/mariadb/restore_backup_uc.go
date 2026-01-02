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
	"databasus-backend/internal/features/backups/backups"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

type RestoreMariadbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMariadbBackupUsecase) Execute(
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMariadb {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"Restoring MariaDB backup via mariadb client",
		"restoreId", restore.ID,
		"backupId", backup.ID,
	)

	mdb := restoringToDB.Mariadb
	if mdb == nil {
		return fmt.Errorf("mariadb configuration is required for restore")
	}

	if mdb.Database == nil || *mdb.Database == "" {
		return fmt.Errorf("target database name is required for mariadb restore")
	}

	args := []string{
		"--host=" + mdb.Host,
		"--port=" + strconv.Itoa(mdb.Port),
		"--user=" + mdb.Username,
		"--verbose",
	}

	if mdb.IsHttps {
		args = append(args, "--ssl")
	}

	if mdb.Database != nil && *mdb.Database != "" {
		args = append(args, *mdb.Database)
	}

	return uc.restoreFromStorage(
		originalDB,
		tools.GetMariadbExecutable(
			tools.MariadbExecutableMariadb,
			mdb.Version,
			config.GetEnv().EnvMode,
			config.GetEnv().MariadbInstallDir,
		),
		args,
		mdb.Password,
		backup,
		storage,
		mdb,
	)
}

func (uc *RestoreMariadbBackupUsecase) restoreFromStorage(
	database *databases.Database,
	mariadbBin string,
	args []string,
	password string,
	backup *backups.Backup,
	storage *storages.Storage,
	mdbConfig *mariadbtypes.MariadbDatabase,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

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

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(database.ID, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	myCnfFile, err := uc.createTempMyCnfFile(mdbConfig, decryptedPassword)
	if err != nil {
		return fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer cleanupFunc()

	return uc.executeMariadbRestore(
		ctx,
		database,
		mariadbBin,
		args,
		myCnfFile,
		tempBackupFile,
		backup,
	)
}

func (uc *RestoreMariadbBackupUsecase) executeMariadbRestore(
	ctx context.Context,
	database *databases.Database,
	mariadbBin string,
	args []string,
	myCnfFile string,
	backupFile string,
	backup *backups.Backup,
) error {
	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

	cmd := exec.CommandContext(ctx, mariadbBin, fullArgs...)
	uc.logger.Info("Executing MariaDB restore command", "command", cmd.String())

	backupFileHandle, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() { _ = backupFileHandle.Close() }()

	var inputReader io.Reader = backupFileHandle

	if backup.Encryption == backups_config.BackupEncryptionEncrypted {
		decryptReader, err := uc.setupDecryption(backupFileHandle, backup)
		if err != nil {
			return fmt.Errorf("failed to setup decryption: %w", err)
		}
		inputReader = decryptReader
	}

	zstdReader, err := zstd.NewReader(inputReader)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	cmd.Stdin = zstdReader

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	stderrCh := make(chan []byte, 1)
	go func() {
		output, _ := io.ReadAll(stderrPipe)
		stderrCh <- output
	}()

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start mariadb: %w", err)
	}

	waitErr := cmd.Wait()
	stderrOutput := <-stderrCh

	if config.IsShouldShutdown() {
		return fmt.Errorf("restore cancelled due to shutdown")
	}

	if waitErr != nil {
		return uc.handleMariadbRestoreError(database, waitErr, stderrOutput, mariadbBin)
	}

	return nil
}

func (uc *RestoreMariadbBackupUsecase) downloadBackupToTempFile(
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

	tempDir, err := os.MkdirTemp(config.GetEnv().TempFolder, "restore_"+uuid.New().String())
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	cleanupFunc := func() {
		_ = os.RemoveAll(tempDir)
	}

	tempBackupFile := filepath.Join(tempDir, "backup.sql.zst")

	uc.logger.Info(
		"Downloading backup file from storage to temporary file",
		"backupId", backup.ID,
		"tempFile", tempBackupFile,
		"encrypted", backup.Encryption == backups_config.BackupEncryptionEncrypted,
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

	_, err = uc.copyWithShutdownCheck(ctx, tempFile, rawReader)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to write backup to temporary file: %w", err)
	}

	uc.logger.Info("Backup file written to temporary location", "tempFile", tempBackupFile)
	return tempBackupFile, cleanupFunc, nil
}

func (uc *RestoreMariadbBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups.Backup,
) (io.Reader, error) {
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, fmt.Errorf("backup is encrypted but missing encryption metadata")
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key for decryption: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	decryptReader, err := encryption.NewDecryptionReader(
		reader,
		masterKey,
		backup.ID,
		salt,
		iv,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	uc.logger.Info("Using decryption for encrypted backup", "backupId", backup.ID)
	return decryptReader, nil
}

func (uc *RestoreMariadbBackupUsecase) createTempMyCnfFile(
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

func (uc *RestoreMariadbBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
) (int64, error) {
	buf := make([]byte, 16*1024*1024)
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

func (uc *RestoreMariadbBackupUsecase) handleMariadbRestoreError(
	database *databases.Database,
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
		backupDbName := "unknown"
		if database.Mariadb != nil && database.Mariadb.Database != nil {
			backupDbName = *database.Mariadb.Database
		}

		return fmt.Errorf(
			"target database does not exist (backup db %s). Create the database before restoring. stderr: %s",
			backupDbName,
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

	return errors.New(errorMsg)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
