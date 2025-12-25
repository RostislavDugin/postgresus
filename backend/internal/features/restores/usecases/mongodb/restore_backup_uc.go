package usecases_mongodb

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
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/backups/backups"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

const (
	restoreTimeout = 23 * time.Hour
)

type RestoreMongodbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMongodbBackupUsecase) Execute(
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMongodb {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"Restoring MongoDB backup via mongorestore",
		"restoreId", restore.ID,
		"backupId", backup.ID,
	)

	mdb := restoringToDB.Mongodb
	if mdb == nil {
		return fmt.Errorf("mongodb configuration is required for restore")
	}

	if mdb.Database == "" {
		return fmt.Errorf("target database name is required for mongorestore")
	}

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(restoringToDB.ID, mdb.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	sourceDatabase := ""
	if originalDB.Mongodb != nil {
		sourceDatabase = originalDB.Mongodb.Database
	}

	args := uc.buildMongorestoreArgs(mdb, decryptedPassword, sourceDatabase)

	return uc.restoreFromStorage(
		tools.GetMongodbExecutable(
			tools.MongodbExecutableMongorestore,
			config.GetEnv().EnvMode,
			config.GetEnv().MongodbInstallDir,
		),
		args,
		backup,
		storage,
	)
}

func (uc *RestoreMongodbBackupUsecase) buildMongorestoreArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
	sourceDatabase string,
) []string {
	uri := mdb.BuildMongodumpURI(password)

	args := []string{
		"--uri=" + uri,
		"--archive",
		"--gzip",
		"--drop",
	}

	if sourceDatabase != "" && sourceDatabase != mdb.Database {
		args = append(args, "--nsFrom="+sourceDatabase+".*")
		args = append(args, "--nsTo="+mdb.Database+".*")
	} else if mdb.Database != "" {
		args = append(args, "--nsInclude="+mdb.Database+".*")
	}

	return args
}

func (uc *RestoreMongodbBackupUsecase) restoreFromStorage(
	mongorestoreBin string,
	args []string,
	backup *backups.Backup,
	storage *storages.Storage,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), restoreTimeout)
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

	tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer cleanupFunc()

	return uc.executeMongoRestore(ctx, mongorestoreBin, args, tempBackupFile, backup)
}

func (uc *RestoreMongodbBackupUsecase) executeMongoRestore(
	ctx context.Context,
	mongorestoreBin string,
	args []string,
	backupFile string,
	backup *backups.Backup,
) error {
	cmd := exec.CommandContext(ctx, mongorestoreBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.Info(
		"Executing MongoDB restore command",
		"command",
		mongorestoreBin,
		"args",
		safeArgs,
	)

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

	cmd.Stdin = inputReader
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8", "LANG=C.UTF-8")

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
		return fmt.Errorf("start mongorestore: %w", err)
	}

	waitErr := cmd.Wait()
	stderrOutput := <-stderrCh

	if config.IsShouldShutdown() {
		return fmt.Errorf("restore cancelled due to shutdown")
	}

	if waitErr != nil {
		return uc.handleMongoRestoreError(waitErr, stderrOutput, mongorestoreBin)
	}

	return nil
}

func (uc *RestoreMongodbBackupUsecase) downloadBackupToTempFile(
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

	tempBackupFile := filepath.Join(tempDir, "backup.archive.gz")

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

func (uc *RestoreMongodbBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups.Backup,
) (io.Reader, error) {
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, errors.New("encrypted backup missing salt or IV")
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	decryptReader, err := encryption.NewDecryptionReader(
		reader,
		masterKey,
		backup.ID,
		salt,
		nonce,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	return decryptReader, nil
}

func (uc *RestoreMongodbBackupUsecase) copyWithShutdownCheck(
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

func (uc *RestoreMongodbBackupUsecase) handleMongoRestoreError(
	waitErr error,
	stderrOutput []byte,
	mongorestoreBin string,
) error {
	stderrStr := string(stderrOutput)

	if containsIgnoreCase(stderrStr, "authentication failed") {
		return fmt.Errorf(
			"MongoDB authentication failed. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "connection refused") ||
		containsIgnoreCase(stderrStr, "server selection error") {
		return fmt.Errorf(
			"MongoDB connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MongoDB connection timeout. stderr: %s",
			stderrStr,
		)
	}

	if len(stderrStr) > 0 {
		return fmt.Errorf(
			"%s failed: %w\nstderr: %s",
			filepath.Base(mongorestoreBin),
			waitErr,
			stderrStr,
		)
	}

	return fmt.Errorf("%s failed: %w", filepath.Base(mongorestoreBin), waitErr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
