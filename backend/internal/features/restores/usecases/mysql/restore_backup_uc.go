package usecases_mysql

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
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

type RestoreMysqlBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMysqlBackupUsecase) Execute(
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMysql {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"Restoring MySQL backup via mysql client",
		"restoreId", restore.ID,
		"backupId", backup.ID,
	)

	my := restoringToDB.Mysql
	if my == nil {
		return fmt.Errorf("mysql configuration is required for restore")
	}

	if my.Database == nil || *my.Database == "" {
		return fmt.Errorf("target database name is required for mysql restore")
	}

	args := []string{
		"--host=" + my.Host,
		"--port=" + strconv.Itoa(my.Port),
		"--user=" + my.Username,
		"--verbose",
	}

	if my.IsHttps {
		args = append(args, "--ssl-mode=REQUIRED")
	}

	if my.Database != nil && *my.Database != "" {
		args = append(args, *my.Database)
	}

	return uc.restoreFromStorage(
		originalDB,
		tools.GetMysqlExecutable(
			my.Version,
			tools.MysqlExecutableMysql,
			config.GetEnv().EnvMode,
			config.GetEnv().MysqlInstallDir,
		),
		args,
		my.Password,
		backup,
		storage,
		my,
	)
}

func (uc *RestoreMysqlBackupUsecase) restoreFromStorage(
	database *databases.Database,
	mysqlBin string,
	args []string,
	password string,
	backup *backups.Backup,
	storage *storages.Storage,
	myConfig *mysqltypes.MysqlDatabase,
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

	myCnfFile, err := uc.createTempMyCnfFile(myConfig, decryptedPassword)
	if err != nil {
		return fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer cleanupFunc()

	return uc.executeMysqlRestore(ctx, database, mysqlBin, args, myCnfFile, tempBackupFile, backup)
}

func (uc *RestoreMysqlBackupUsecase) executeMysqlRestore(
	ctx context.Context,
	database *databases.Database,
	mysqlBin string,
	args []string,
	myCnfFile string,
	backupFile string,
	backup *backups.Backup,
) error {
	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

	cmd := exec.CommandContext(ctx, mysqlBin, fullArgs...)
	uc.logger.Info("Executing MySQL restore command", "command", cmd.String())

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
		return fmt.Errorf("start mysql: %w", err)
	}

	waitErr := cmd.Wait()
	stderrOutput := <-stderrCh

	if config.IsShouldShutdown() {
		return fmt.Errorf("restore cancelled due to shutdown")
	}

	if waitErr != nil {
		return uc.handleMysqlRestoreError(database, waitErr, stderrOutput, mysqlBin)
	}

	return nil
}

func (uc *RestoreMysqlBackupUsecase) downloadBackupToTempFile(
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

func (uc *RestoreMysqlBackupUsecase) setupDecryption(
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

func (uc *RestoreMysqlBackupUsecase) createTempMyCnfFile(
	myConfig *mysqltypes.MysqlDatabase,
	password string,
) (string, error) {
	tempDir, err := os.MkdirTemp("", "mycnf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	myCnfFile := filepath.Join(tempDir, ".my.cnf")

	content := fmt.Sprintf(`[client]
user=%s
password="%s"
host=%s
port=%d
`, myConfig.Username, tools.EscapeMysqlPassword(password), myConfig.Host, myConfig.Port)

	if myConfig.IsHttps {
		content += "ssl-mode=REQUIRED\n"
	}

	err = os.WriteFile(myCnfFile, []byte(content), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write .my.cnf: %w", err)
	}

	return myCnfFile, nil
}

func (uc *RestoreMysqlBackupUsecase) copyWithShutdownCheck(
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

func (uc *RestoreMysqlBackupUsecase) handleMysqlRestoreError(
	database *databases.Database,
	waitErr error,
	stderrOutput []byte,
	mysqlBin string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf(
		"%s failed: %v – stderr: %s",
		filepath.Base(mysqlBin),
		waitErr,
		stderrStr,
	)

	if containsIgnoreCase(stderrStr, "access denied") {
		return fmt.Errorf(
			"MySQL access denied. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "can't connect") ||
		containsIgnoreCase(stderrStr, "connection refused") {
		return fmt.Errorf(
			"MySQL connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "unknown database") {
		backupDbName := "unknown"
		if database.Mysql != nil && database.Mysql.Database != nil {
			backupDbName = *database.Mysql.Database
		}

		return fmt.Errorf(
			"target database does not exist (backup db %s). Create the database before restoring. stderr: %s",
			backupDbName,
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") {
		return fmt.Errorf(
			"MySQL SSL connection failed. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MySQL connection timeout. stderr: %s",
			stderrStr,
		)
	}

	return errors.New(errorMsg)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
